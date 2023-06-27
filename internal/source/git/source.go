package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
	"go.flipt.io/fidgit/internal/containers"
	"go.flipt.io/fidgit/internal/gitfs"
	"golang.org/x/exp/slog"
)

// Source is an implementation of storage/fs.FSSource
// This implementation is backed by a Git repository and it tracks an upstream reference.
// When subscribing to this source, the upstream reference is tracked
// by polling the upstream on a configurable interval.
type Source struct {
	logger *slog.Logger
	repo   *git.Repository

	url      string
	ref      string
	hash     plumbing.Hash
	interval time.Duration
	auth     transport.AuthMethod

	ch chan fs.FS
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

// NewSource constructs and configures a Source.
// The source uses the connection and credential details provided to build
// fs.FS implementations around a target git repository.
func NewSource(ctx context.Context, url string, opts ...containers.Option[Source]) (_ *Source, err error) {
	source := &Source{
		logger:   slog.With(slog.String("repository", url)),
		url:      url,
		ref:      "main",
		interval: 30 * time.Second,
		ch:       make(chan fs.FS, 1),
	}
	containers.ApplyAll(source, opts...)

	field := slog.String("ref", plumbing.NewBranchReferenceName(source.ref).String())
	if source.hash != plumbing.ZeroHash {
		field = slog.String("SHA", source.hash.String())
	}

	source.logger = source.logger.With(field)

	source.repo, err = git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		Auth: source.auth,
		URL:  source.url,
	})
	if err != nil {
		return nil, err
	}

	// prime the first fs and then poll for updates
	// from the upstream git repo.
	fs, err := source.build()
	if err != nil {
		return nil, err
	}

	source.ch <- fs

	go source.subscribe(ctx)

	return source, nil
}

// Get builds a new fs.FS based on the configure Git remote and reference.
func (s *Source) Get(ctx context.Context) (fs.FS, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case fs, ok := <-s.ch:
		if !ok {
			return nil, errors.New("source has closed")
		}

		return fs, nil
	}
}

func (s *Source) build() (fs.FS, error) {
	if s.hash != plumbing.ZeroHash {
		return gitfs.NewFromRepoHash(s.repo, s.hash)
	}

	return gitfs.NewFromRepo(s.repo, gitfs.WithReference(plumbing.NewRemoteReferenceName("origin", s.ref)))
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

			fs, err := s.build()
			if err != nil {
				s.logger.Error("failed creating gitfs", "error", err)
				continue
			}

			s.ch <- fs

			s.logger.Debug("finished fetching from remote")
		}
	}
}
