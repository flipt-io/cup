package local

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.flipt.io/fidgit"
	"golang.org/x/exp/slog"
)

var _ fidgit.Source = (*Source)(nil)

type Source struct {
	path string

	ch chan fs.FS
}

func New(ctx context.Context, path string) *Source {
	src := &Source{path: path, ch: make(chan fs.FS, 1)}

	// prime the first fs and then periodically return
	// a new instace every 10 seconds
	src.ch <- os.DirFS(src.path)

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		defer close(src.ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				src.ch <- os.DirFS(src.path)
			}
		}
	}()

	return src
}

func (s *Source) Get(ctx context.Context) (fs.FS, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case fs, ok := <-s.ch:
		if !ok {
			return nil, errors.New("source has shutdown")
		}

		return fs, nil
	}
}

func (s *Source) Propose(_ context.Context, req fidgit.ProposeRequest) error {
	for _, change := range req.Changes {
		slog.Debug("Handling Change", "path", change.Path)

		rel, err := filepath.Rel(s.path, change.Path)
		if err != nil {
			return fmt.Errorf("local: proposing change: %w", err)
		}

		slog.Debug("Relative Path", "path", rel)

		if dir, _ := filepath.Split(rel); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("local: proposing change: %w", err)
			}
		}

		if err := os.WriteFile(rel, change.Contents, 0644); err != nil {
			return fmt.Errorf("local: proposing change: %w", err)
		}
	}

	return nil
}
