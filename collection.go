package fidgit

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
)

type Namespace string

type ID string

type Type struct {
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

func (t Type) String() string {
	return path.Join(t.Group, t.Kind, t.Version)
}

type namespace struct {
	entries []*Entry
	index   map[ID]*Entry
}

type Entry struct {
	Namespace Namespace       `json:"namespace"`
	ID        ID              `json:"id"`
	Payload   json.RawMessage `json:"payload"`
}

type Collection interface {
	Get(ctx context.Context, n Namespace, id ID) (*Entry, error)
	List(ctx context.Context, n Namespace) ([]*Entry, error)
	Put(ctx context.Context, entry *Entry) ([]Change, error)
	Delete(ctx context.Context, n Namespace, id ID) ([]Change, error)
}

type FactoryFunc func(context.Context, fs.FS) (Collection, error)
