package flipt

import (
	"context"
	"io/fs"

	"go.flipt.io/fidgit"
)

type Flag struct {
	Namespace   string
	Key         string
	Description string
	Enabled     bool
}

func (f *Flag) GetNamespace() fidgit.Namespace {
	return fidgit.Namespace(f.Namespace)
}

func (f *Flag) GetID() fidgit.ID {
	return fidgit.ID(f.Key)
}

func (f *Flag) GetTags() []fidgit.Tag {
	return nil
}

func (f *Flag) GetInternalContext() map[string]string {
	return nil
}

var _ (fidgit.CollectionFactory[*Flag]) = (*FlagCollectionFactory)(nil)

type FlagCollectionFactory struct{}

func (f *FlagCollectionFactory) GetType() fidgit.Type {
	return fidgit.Type{
		Kind:    "io.flipt.Flag",
		Version: "v1alpha1",
	}
}

func (f *FlagCollectionFactory) GetTagKeys() []string {
	return nil
}

func (f *FlagCollectionFactory) CollectionFor(_ context.Context, _ fs.FS) (fidgit.CollectionRuntime[*Flag], error) {
	return &FlagCollection{}, nil
}

type FlagCollection struct{}

func (f *FlagCollection) ListAll(_ context.Context) ([]*Flag, error) {
	//TODO(georgemac): read all Flipt flags from disk via index
	panic("not implemented") // TODO: Implement
}

func (f *FlagCollection) Put(_ context.Context, _ *Flag) error {
	//TODO(georgemac): locate file marshal out new or updated entry
	panic("not implemented") // TODO: Implement
}

func (f *FlagCollection) Delete(_ context.Context, _ fidgit.Namespace, _ fidgit.ID) error {
	//TODO(georgemac): locate file marshal out removal of entry
	panic("not implemented") // TODO: Implement
}
