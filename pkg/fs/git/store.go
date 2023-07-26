package git

import (
	"context"
	"fmt"

	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/containers"
)

var _ api.FilesystemStore = (*FilesystemStore)(nil)

type FilesystemStore struct {
	sources containers.MapStore[string, *Filesystem]
}

func NewFilesystemStore() *FilesystemStore {
	return &FilesystemStore{
		sources: containers.MapStore[string, *Filesystem]{},
	}
}

func (f *FilesystemStore) AddSource(source string, fs *Filesystem) {
	f.sources[source] = fs
}

// View invokes the provided function with an FSConfig which should enforce
// a read-only view for the requested source and revision
func (f *FilesystemStore) View(ctx context.Context, source string, revision string, fn api.ViewFunc) error {
	fs, err := f.sources.Get(source)
	if err != nil {
		return fmt.Errorf("view: %w", err)
	}

	return fs.View(ctx, revision, fn)
}

// Update invokes the provided function with an FSConfig which can be written to
// Any writes performed to the target during the execution of fn will be added,
// comitted, pushed and proposed for review on a target SCM
func (f *FilesystemStore) Update(ctx context.Context, source string, revision string, message string, fn api.UpdateFunc) (*api.Result, error) {
	fs, err := f.sources.Get(source)
	if err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	return fs.Update(ctx, revision, message, fn)
}
