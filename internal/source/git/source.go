package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gofrs/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.flipt.io/cup"
	"go.flipt.io/cup/internal/containers"
	"golang.org/x/exp/slog"
)

var _ cup.Source = (*Source)(nil)

// Source is an implementation of storage/fs.FSSource
// This implementation is backed by a Git repository and it tracks an upstream reference.
// When subscribing to this source, the upstream reference is tracked
// by polling the upstream on a configurable interval.
type Source struct {
	logger  *slog.Logger
	repo    *git.Repository
	storage *memory.Storage

	url      string
	ref      string
	hash     plumbing.Hash
	interval time.Duration
	auth     transport.AuthMethod

	mu    sync.Mutex
	cache *lru.Cache[plumbing.Hash, fs.FS]

	ch chan *revisionFS
}

// WithRef configures the target reference to be used when fetching
// and building fs.FS implementations.
// If it is a valid hash, then the fixed SHA value is used.
// Otherwise, it is treated as a reference in the origin upstream.
func WithRef(ref string) containers.Option[Source] {
	return func(s *Source) {
		if plumbing.IsHash(ref) {
			s.hash = plumbing.NewHash(ref)
			return
		}

		s.ref = ref
	}
}

// WithPollInterval configures the interval in which origin is polled to
// discover any updates to the target reference.
func WithPollInterval(tick time.Duration) containers.Option[Source] {
	return func(s *Source) {
		s.interval = tick
	}
}

// WithAuth returns an option which configures the auth method used
// by the provided source.
func WithAuth(auth transport.AuthMethod) containers.Option[Source] {
	return func(s *Source) {
		s.auth = auth
	}
}

type revisionFS struct {
	fs       fs.FS
	revision string
}

// NewSource constructs and configures a Source.
// The source uses the connection and credential details provided to build
// fs.FS implementations around a target git repository.
func NewSource(ctx context.Context, url string, opts ...containers.Option[Source]) (_ *Source, err error) {
	source := &Source{
		logger:   slog.With(slog.String("repository", url)),
		url:      url,
		ref:      "main",
		interval: 30 * time.Second,
		ch:       make(chan *revisionFS, 1),
	}
	containers.ApplyAll(source, opts...)

	source.cache, err = lru.New[plumbing.Hash, fs.FS](2)
	if err != nil {
		return nil, err
	}

	field := slog.String("ref", plumbing.NewBranchReferenceName(source.ref).String())
	if source.hash != plumbing.ZeroHash {
		field = slog.String("SHA", source.hash.String())
	}

	source.logger = source.logger.With(field)

	source.storage = memory.NewStorage()
	source.repo, err = git.Clone(source.storage, nil, &git.CloneOptions{
		Auth: source.auth,
		URL:  source.url,
	})
	if err != nil {
		return nil, err
	}

	// prime the first fs and then poll for updates
	// from the upstream git repo.
	fs, err := source.build(source.hash)
	if err != nil {
		return nil, err
	}

	source.ch <- fs

	go source.subscribe(ctx)

	return source, nil
}

// Get builds a new fs.FS based on the configure Git remote and reference.
func (s *Source) Get(ctx context.Context) (fs.FS, string, error) {
	if s.hash != plumbing.ZeroHash {
		fs, ok := s.cache.Get(s.hash)
		if !ok {
			return nil, "", fmt.Errorf("FS not found: %s", s.hash)
		}

		return fs, s.hash.String(), nil
	}

	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case fs, ok := <-s.ch:
		if !ok {
			return nil, "", errors.New("source has closed")
		}

		return fs.fs, fs.revision, nil
	}
}

func (s *Source) build(hash plumbing.Hash) (_ *revisionFS, err error) {
	if hash == plumbing.ZeroHash {
		ref, err := s.repo.Reference(plumbing.NewRemoteReferenceName("origin", s.ref), true)
		if err != nil {
			return nil, fmt.Errorf("resolving reference (%q): %w", s.ref, err)
		}

		hash = ref.Hash()

		slog.Debug("Resolved Reference", "ref", s.ref, "hash", hash)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// check LRU cache incase we have the entry already built
	if fs, ok := s.cache.Get(hash); ok {
		return &revisionFS{fs: fs, revision: hash.String()}, nil
	}

	dir, err := os.MkdirTemp("", "cup-*")
	if err != nil {
		return nil, err
	}

	repo, err := git.Open(s.shallowCopyStorage(), osfs.New(dir))
	if err != nil {
		return nil, err
	}

	work, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	if err := work.Checkout(&git.CheckoutOptions{
		Hash: hash,
	}); err != nil {
		return nil, err
	}

	fs := os.DirFS(dir)
	// add fs for hash to cache
	s.cache.Add(hash, fs)

	return &revisionFS{fs: fs, revision: hash.String()}, nil
}

// subscribe feeds gitfs implementations of fs.FS onto the provided channel.
// It blocks until the provided context is cancelled (it will be called in a goroutine).
// It closes the provided channel before it returns.
func (s *Source) subscribe(ctx context.Context) {
	defer close(s.ch)

	// NOTE: theres is no point subscribing to updates for a git Hash
	// as it is atomic and will never change.
	if s.hash != plumbing.ZeroHash {
		s.logger.Info("skipping subscribe as static SHA has been configured")
		return
	}

	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.logger.Debug("fetching from remote")
			if err := s.repo.Fetch(&git.FetchOptions{
				Auth: s.auth,
				RefSpecs: []config.RefSpec{
					config.RefSpec(fmt.Sprintf(
						"+%s:%s",
						plumbing.NewBranchReferenceName(s.ref),
						plumbing.NewRemoteReferenceName("origin", s.ref),
					)),
				},
			}); err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					s.logger.Debug("store already up to date")
					continue
				}

				s.logger.Error("failed fetching remote", "error", err)
				continue
			}

			fs, err := s.build(s.hash)
			if err != nil {
				s.logger.Error("failed creating gitfs", "error", err)
				continue
			}

			s.ch <- fs

			s.logger.Debug("finished fetching from remote")
		}
	}
}

func (s *Source) shallowCopyStorage() *memory.Storage {
	// shallow copy the store without the existing index
	return &memory.Storage{
		ReferenceStorage: s.storage.ReferenceStorage,
		ConfigStorage:    s.storage.ConfigStorage,
		ShallowStorage:   s.storage.ShallowStorage,
		ObjectStorage:    s.storage.ObjectStorage,
		ModuleStorage:    s.storage.ModuleStorage,
	}
}

func ptr[T any](t T) *T { return &t }

func (s *Source) Propose(ctx context.Context, r cup.ProposeRequest) (*cup.Proposal, error) {
	// validate revision
	if !plumbing.IsHash(r.Revision) {
		return nil, fmt.Errorf("ref is not valid hash: %q", r.Revision)
	}

	hash := plumbing.NewHash(r.Revision)

	// shallow copy the store without the existing index
	store := s.shallowCopyStorage()

	// open repository on store with in-memory workspace
	repo, err := git.Open(store, memfs.New())
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	work, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("open worktree: %w", err)
	}

	repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{s.url},
	})

	proposal := &cup.Proposal{
		ID: ptr(uuid.Must(uuid.NewV4()).String()),
	}
	// create proposal branch (cup/proposal/$id)
	branch := fmt.Sprintf("cup/proposal/%s", *proposal.ID)
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
		status, _ := work.Status()
		s.logger.Debug("status", slog.String("status", status.String()))

		return nil, fmt.Errorf("checkout branch: %w", err)
	}

	// for each requested change
	// make it to the associated document
	// and then add and commit the difference
	for _, change := range r.Changes {
		if change.Message == "" {
			slog.Debug("Skipping Change", slog.String("reason", "no message"))
			continue
		}

		s.logger.Debug("Processing Change",
			slog.String("message", change.Message),
			slog.String("path", change.Path))

		if len(change.Contents) == 0 {
			if _, err := work.Remove(change.Path); err != nil {
				return nil, err
			}
		} else {
			fi, err := work.Filesystem.OpenFile(change.Path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				return nil, err
			}
			defer fi.Close()

			_, err = io.Copy(fi, bytes.NewBuffer(change.Contents))
			if err != nil {
				return nil, err
			}

			_, err = work.Add(change.Path)
			if err != nil {
				return nil, err
			}
		}

		_, err = work.Commit(change.Message, &git.CommitOptions{
			Author:    &object.Signature{Email: "dev@flipt.io", Name: "Dev"},
			Committer: &object.Signature{Email: "dev@flipt.io", Name: "Dev"},
		})
		if err != nil {
			return nil, fmt.Errorf("committing changes: %w", err)
		}
	}

	s.logger.Debug("Pushing Changes", slog.String("branch", branch))

	b, err := repo.Branch(branch)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("branch", slog.String("name", b.Name), slog.String("remote", b.Remote))

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

	// open PR

	return proposal, nil
}
