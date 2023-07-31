package mem

import (
	"context"
	"fmt"

	"github.com/go-git/go-billy/v5"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/billyfs"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controllers"
)

var _ api.Source = (*Source)(nil)

// Source is primarily used for testing.
// The implementations are indexed by revision internally.
// It supports writes through the billyfs abstraction.
// However, instead of proposals, these are direct writes to the underlying filesystem.
type Source struct {
	revs containers.MapStore[string, billy.Filesystem]
}

// New constructs a new instance of a Source
func New() *Source {
	return &Source{revs: containers.MapStore[string, billy.Filesystem]{}}
}

// AddFS registers a new fs.FS to be supplied on calls to View and Update
func (f *Source) AddFS(revision string, ffs billy.Filesystem) {
	f.revs[revision] = ffs
}

// View invokes the provided function with an FSConfig which should enforce
// a read-only view for the requested source and revision
func (f *Source) View(_ context.Context, revision string, fn api.ViewFunc) error {
	fs, err := f.revs.Get(revision)
	if err != nil {
		return fmt.Errorf("view: %w", err)
	}

	return fn(billyfs.New(fs))
}

// Update invokes the provided function with an FSConfig which can be written to
// Any writes performed to the target during the execution of fn will be added,
// comitted, pushed and proposed for review on a target SCM
func (f *Source) Update(_ context.Context, revision, _ string, fn api.UpdateFunc) (*api.Result, error) {
	fs, err := f.revs.Get(revision)
	if err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	return &api.Result{}, fn(controllers.NewFSConfig(fs))
}
