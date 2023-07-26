package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/oklog/ulid/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controller"
	"go.flipt.io/cup/pkg/gitfs"
	"golang.org/x/exp/slog"
)

// Proposal is the internal representation of what becomes a pull or merge request
// on a target SCM.
// It contains the fields necessary to identify the proposed branch and describe
// the change.
type Proposal struct {
	ID    ulid.ULID
	Head  string
	Base  string
	Title string
	Body  string
}

// ProposalResponse is a structure which will contain any details identifying
// the successful proposition.
type ProposalResponse struct{}

// SCM is an abstraction around repositories and source control providers.
type SCM interface {
	Propose(context.Context, Proposal) (ProposalResponse, error)
}

// FilesystemStore is an implementation of api.FilesystemStore
// This implementation is backed by a Git repository and it tracks an upstream reference.
// When subscribing to this source, the upstream reference is tracked
// by polling the upstream on a configurable interval.
type Filesystem struct {
	logger  *slog.Logger
	repo    *git.Repository
	storage *memory.Storage

	url      string
	scm      SCM
	interval time.Duration
	auth     transport.AuthMethod

	// notify is used for informing listeners
	// during tests that a fetch was performed
	// and the state was updated
	notify chan struct{}
}

// WithPollInterval configures the interval in which origin is polled to
// discover any updates to the target reference.
func WithPollInterval(tick time.Duration) containers.Option[Filesystem] {
	return func(s *Filesystem) {
		s.interval = tick
	}
}

// WithAuth returns an option which configures the auth method used
// by the provided source.
func WithAuth(auth transport.AuthMethod) containers.Option[Filesystem] {
	return func(s *Filesystem) {
		s.auth = auth
	}
}

// NewFilesystem constructs and configures a Git backend Filesystem.
// The implementation uses the connection and credential details provided to support
// view and update requests for use in the api server.
func NewFilesystem(ctx context.Context, scm SCM, url string, opts ...containers.Option[Filesystem]) (_ *Filesystem, err error) {
	fs := &Filesystem{
		logger:   slog.With(slog.String("repository", url)),
		url:      url,
		scm:      scm,
		interval: 30 * time.Second,
		notify:   make(chan struct{}, 1),
	}
	containers.ApplyAll(fs, opts...)

	fs.storage = memory.NewStorage()
	fs.repo, err = git.Clone(fs.storage, nil, &git.CloneOptions{
		Auth: fs.auth,
		URL:  fs.url,
	})
	if err != nil {
		return nil, err
	}

	go fs.pollRefs(ctx)

	return fs, nil
}

// View builds a new fs.FS based on the configure Git remote and reference.
// It call the provided function with the derived fs.FS.
func (s *Filesystem) View(ctx context.Context, rev string, fn api.ViewFunc) error {
	hash, err := s.resolve(rev)
	if err != nil {
		return err
	}

	fs, err := gitfs.NewFromRepoHash(s.repo, hash)
	if err != nil {
		return err
	}

	return fn(fs)
}

// Update builds a worktree in a temporary directory for the provided revision over the configured Git repository.
// The provided function is called with the checked out worktree.
// Any changes made during the function call to the underlying worktree are added commit and pushed to the
// target Git repository.
// Once pushed a proposal is made on the configured SCM.
func (s *Filesystem) Update(ctx context.Context, rev, message string, fn api.UpdateFunc) (*api.Result, error) {
	hash, err := s.resolve(rev)
	if err != nil {
		return nil, err
	}

	// shallow copy the store without the existing index
	store := &memory.Storage{
		ReferenceStorage: s.storage.ReferenceStorage,
		ConfigStorage:    s.storage.ConfigStorage,
		ShallowStorage:   s.storage.ShallowStorage,
		ObjectStorage:    s.storage.ObjectStorage,
		ModuleStorage:    s.storage.ModuleStorage,
	}

	dir, err := os.MkdirTemp("", "cup-proposal-*")
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	// open repository on store with in-memory workspace
	repo, err := git.Open(store, osfs.New(dir))
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	work, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("open worktree: %w", err)
	}

	result := &api.Result{
		ID: ulid.Make(),
	}

	// create proposal branch (cup/proposal/$id)
	branch := fmt.Sprintf("cup/proposal/%s", result.ID)
	if err := repo.CreateBranch(&config.Branch{
		Name:   branch,
		Remote: "origin",
	}); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	if err := work.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Create: true,
		Hash:   hash,
	}); err != nil {
		return nil, fmt.Errorf("checkout branch: %w", err)
	}

	if err := fn(controller.NewDirFSConfig(dir)); err != nil {
		return nil, fmt.Errorf("execute proposal: %w", err)
	}

	if err := work.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return nil, fmt.Errorf("adding changes: %w", err)
	}

	_, err = work.Commit(message, &git.CommitOptions{
		Author:    &object.Signature{Email: "cup@flipt.io", Name: "cup"},
		Committer: &object.Signature{Email: "cup@flipt.io", Name: "cup"},
	})
	if err != nil {
		return nil, fmt.Errorf("committing changes: %w", err)
	}

	s.logger.Debug("Pushing Changes", slog.String("branch", branch))

	b, err := repo.Branch(branch)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("Branch", slog.String("name", b.Name), slog.String("remote", b.Remote))

	// push to proposed branch
	if err := repo.PushContext(ctx, &git.PushOptions{
		Auth:       s.auth,
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("%s:refs/heads/%s", branch, branch)),
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)),
		},
	}); err != nil {
		return nil, fmt.Errorf("pushing changes: %w", err)
	}

	if _, err := s.scm.Propose(ctx, Proposal{
		Head:  branch,
		Base:  rev,
		Title: message,
		Body:  message,
	}); err != nil {
		return nil, fmt.Errorf("proposing change: %w", err)
	}

	return result, nil
}

func (s *Filesystem) resolve(r string) (plumbing.Hash, error) {
	if plumbing.IsHash(r) {
		return plumbing.NewHash(r), nil
	}

	ref, err := s.repo.Reference(plumbing.NewRemoteReferenceName("origin", r), true)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	return ref.Hash(), nil
}

func (s *Filesystem) pollRefs(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.repo.FetchContext(ctx, &git.FetchOptions{
				Auth: s.auth,
			}); err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					slog.Debug("References are all up to date")

					continue
				}

				slog.Error("Fetching references", "error", err)
			}

			// attempt to notify any listeners
			// but dont block if nothing is listening
			// this is used for tests
			select {
			case s.notify <- struct{}{}:
			default:
			}
		}
	}
}
