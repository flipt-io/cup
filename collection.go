package fidgit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"

	"golang.org/x/exp/slog"
)

type Namespace string

type ID string

type Tag struct {
	Key, Value string
}

type Item interface {
	GetNamespace() Namespace
	GetID() ID
	GetTags() []Tag
}

type Type struct {
	Group   string
	Kind    string
	Version string
}

func (t Type) String() string {
	return path.Join(t.Group, t.Kind, t.Version)
}

type RuntimePutRequest[I Item] struct {
	Namespace Namespace
	Item      *I
	Revision  *string
}

type Runtime[I Item] interface {
	ListAll(context.Context) ([]*I, error)
	Put(context.Context, RuntimePutRequest[I]) ([]Change, error)
	Delete(context.Context, Namespace, ID) ([]Change, error)
}

type namespace struct {
	entries []json.RawMessage
	index   map[ID]json.RawMessage
}

type Collection struct {
	typ     Type
	tagKeys []string
	logger  *slog.Logger

	index map[Namespace]*namespace
	put   func(context.Context, CollectionPutRequest) ([]Change, error)
	del   func(context.Context, Namespace, ID) ([]Change, error)
}

type RuntimeFactory[I Item] interface {
	GetType() Type
	GetTagKeys() []string
	CollectionFor(context.Context, fs.FS) (Runtime[I], error)
}

type FactoryFunc func(context.Context, fs.FS) (*Collection, error)

func FactoryFor[I Item](f RuntimeFactory[I]) (Type, FactoryFunc) {
	return f.GetType(), func(ctx context.Context, ffs fs.FS) (*Collection, error) {
		collection := Collection{
			typ:     f.GetType(),
			tagKeys: f.GetTagKeys(),
			logger: slog.With(
				slog.String("system", "collection"),
				slog.String("group", f.GetType().Group),
				slog.String("kind", f.GetType().Kind),
				slog.String("version", f.GetType().Version),
			),
		}

		r, err := f.CollectionFor(ctx, ffs)
		if err != nil {
			return nil, err
		}

		all, err := r.ListAll(ctx)
		if err != nil {
			return nil, err
		}

		index := map[Namespace]*namespace{}
		for _, item := range all {
			raw, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}

			ns, ok := index[(*item).GetNamespace()]
			if !ok {
				ns = &namespace{
					index: map[ID]json.RawMessage{},
				}
				index[(*item).GetNamespace()] = ns
			}

			ns.entries = append(ns.entries, raw)
			ns.index[(*item).GetID()] = raw
		}

		collection.index = index
		collection.del = r.Delete
		collection.put = func(ctx context.Context, req CollectionPutRequest) ([]Change, error) {
			var i I
			if err := json.Unmarshal(req.Payload, &i); err != nil {
				return nil, fmt.Errorf("putting item: %w", err)
			}

			collection.logger.Debug("Put",
				slog.String("namespace", string(req.Namespace)),
				slog.String("id", string(i.GetID())))

			changes, err := r.Put(ctx, RuntimePutRequest[I]{
				Namespace: req.Namespace,
				Item:      &i,
				Revision:  req.Revision,
			})
			if err != nil {
				n, id := req.Namespace, i.GetID()
				return nil, fmt.Errorf("%s: item %s/%s: %w", f.GetType(), n, id, err)
			}

			return changes, nil
		}

		return &collection, nil
	}
}

func (c *Collection) Get(ctx context.Context, n Namespace, id ID) ([]byte, error) {
	c.logger.Debug("Get",
		slog.String("namespace", string(n)),
		slog.String("id", string(id)))

	if ns, ok := c.index[n]; ok {
		if entry, ok := ns.index[id]; ok {
			return entry, nil
		}
	}

	return nil, fmt.Errorf("%s: item %s/%s: not found", c.typ, n, id)
}

func (c *Collection) List(ctx context.Context, n Namespace) ([]byte, error) {
	c.logger.Debug("List",
		slog.String("namespace", string(n)))

	if ns, ok := c.index[n]; ok {
		return json.Marshal(ns.entries)
	}

	return nil, fmt.Errorf("%s: namespace %s: not found", c.typ, n)
}

type CollectionPutRequest struct {
	Namespace Namespace
	Revision  *string
	Payload   []byte
}

func (c *Collection) Put(ctx context.Context, req CollectionPutRequest) ([]Change, error) {
	return c.put(ctx, req)
}

func (c *Collection) Delete(ctx context.Context, n Namespace, id ID) ([]Change, error) {
	c.logger.Debug("Delete",
		slog.String("namespace", string(n)),
		slog.String("id", string(id)))

	return c.del(ctx, n, id)
}
