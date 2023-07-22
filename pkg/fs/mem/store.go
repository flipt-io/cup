package mem

import (
	"context"
	"fmt"
	"io/fs"

	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/controller"
)

// FilesystemStore is primarily used for testing.
// It approximates a real implementation using a set of fs.FS implementations.
// The implementations are indexed by source and revision internally.
// It does not truly support updates as of now as we have no way to virtualize
// a writeable FS for Wazero. It just supplies a read-only FS and the assumes
// no writes will be attempted.
// When the FS interface in Wazero is available we can change this behaviour.
type FilesystemStore struct {
	store map[string]map[string]fs.FS
}

// New constructs a new instance of FilesystemStore
func New() *FilesystemStore {
	return &FilesystemStore{store: map[string]map[string]fs.FS{}}
}

// AddFS registers a new fs.FS to be supplied on calls to View and Update
func (f *FilesystemStore) AddFS(source, revision string, ffs fs.FS) {
	src, ok := f.store[source]
	if !ok {
		src = map[string]fs.FS{}
		f.store[source] = src
	}

	src[revision] = ffs
}

// View invokes the provided function with an FSConfig which should enforce
// a read-only view for the requested source and revision
func (f *FilesystemStore) View(_ context.Context, source string, revision string, fn api.FSFunc) error {
	fs, err := f.fs(source, revision)
	if err != nil {
		return fmt.Errorf("view: %w", err)
	}

	return fn(controller.NewFSConfig(fs))
}

// Update invokes the provided function with an FSConfig which can be written to
// Any writes performed to the target during the execution of fn will be added,
// comitted, pushed and proposed for review on a target SCM
func (f *FilesystemStore) Update(_ context.Context, source string, revision string, fn api.FSFunc) (*api.Result, error) {
	fs, err := f.fs(source, revision)
	if err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	return &api.Result{}, fn(controller.NewFSConfig(fs))
}

func (f *FilesystemStore) fs(source, revision string) (fs.FS, error) {
	src, ok := f.store[source]
	if !ok {
		return nil, fmt.Errorf("source not found: %q", source)
	}

	fs, ok := src[revision]
	if !ok {
		return nil, fmt.Errorf("revision not found: \"%s:%s\"", source, revision)
	}

	return fs, nil
}
