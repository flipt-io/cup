package local

import (
	"context"

	"github.com/go-git/go-billy/v5/osfs"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/billyfs"
	"go.flipt.io/cup/pkg/controllers"
)

// Source implements the abstraction required by an *api.Server
// to read and update a target source filesystem.
// This implementation works directly over the host.
type Source struct {
	path string
}

// New constructs and configures a new instance of *Source
// for the provided path.
func New(path string) *Source {
	return &Source{path: path}
}

// View invokes the provided function with an fs.FS which should enforce
// a read-only view for the requested source and revision.
func (f *Source) View(_ context.Context, revision string, fn api.ViewFunc) error {
	return fn(billyfs.New(osfs.New(f.path)))
}

// Update invokes the provided function with an FSConfig which can be written to
// Any writes performed to the target during the execution of fn will be added,
// comitted, pushed and proposed for review on a target SCM.
func (f *Source) Update(_ context.Context, revision string, message string, fn api.UpdateFunc) (*api.Result, error) {
	return &api.Result{}, fn(controllers.FSConfig{
		FS:  osfs.New(f.path),
		Dir: &f.path,
	})
}
