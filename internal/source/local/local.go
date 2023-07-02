package local

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.flipt.io/cup"
	"golang.org/x/exp/slog"
)

var _ cup.Source = (*Source)(nil)

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

func (s *Source) Get(ctx context.Context) (fs.FS, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case fs, ok := <-s.ch:
		if !ok {
			return nil, "", errors.New("source has shutdown")
		}

		return fs, "static", nil
	}
}

func (s *Source) Propose(_ context.Context, req cup.ProposeRequest) (*cup.Proposal, error) {
	for _, change := range req.Changes {
		slog.Debug("Handling Change", "path", change.Path, "message", change.Message)

		rel, err := filepath.Rel(s.path, change.Path)
		if err != nil {
			return nil, fmt.Errorf("local: proposing change: %w", err)
		}

		if dir, _ := filepath.Split(rel); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("local: proposing change: %w", err)
			}
		}

		if err := os.WriteFile(rel, change.Contents, 0644); err != nil {
			return nil, fmt.Errorf("local: proposing change: %w", err)
		}
	}

	return &cup.Proposal{Status: "done"}, nil
}
