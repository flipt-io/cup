package fidgit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sync"

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
	// GetInternalContext is used by the runtime implementation
	// to optimize lookups and will not intended
	// for external consumption.
	GetInternalContext() map[string]string
}

type Type struct {
	Kind    string
	Version string
}

func (t Type) String() string {
	return path.Join(t.Kind, t.Version)
}

type CollectionRuntime[I Item] interface {
	ListAll(context.Context) ([]I, error)
	Put(context.Context, I) error
	Delete(context.Context, Namespace, ID) error
}

type namespace struct {
	entries []json.RawMessage
	index   map[ID]json.RawMessage
}

type Collection struct {
	typ     Type
	tagKeys []string
	logger  *slog.Logger

	mu             sync.RWMutex
	updateSnapshot func(context.Context, fs.FS) error
	index          map[Namespace]namespace
	put            func(context.Context, []byte) error
	del            func(context.Context, Namespace, ID) error
}

type CollectionFactory[I Item] interface {
	GetType() Type
	GetTagKeys() []string
	CollectionFor(context.Context, fs.FS) (CollectionRuntime[I], error)
}

func CollectionFor[I Item](ctx context.Context, f CollectionFactory[I]) (*Collection, error) {
	collection := Collection{
		typ:     f.GetType(),
		tagKeys: f.GetTagKeys(),
		logger: slog.With(
			slog.String("system", "collection"),
			slog.String("kind", f.GetType().Kind),
			slog.String("version", f.GetType().Version),
		),
	}

	collection.updateSnapshot = func(ctx context.Context, ffs fs.FS) error {
		r, err := f.CollectionFor(ctx, ffs)
		if err != nil {
			return err
		}

		all, err := r.ListAll(ctx)
		if err != nil {
			return err
		}

		index := map[Namespace]namespace{}
		for _, item := range all {
			raw, err := json.Marshal(item)
			if err != nil {
				return err
			}

			ns, ok := index[item.GetNamespace()]
			if !ok {
				ns := namespace{
					index: map[ID]json.RawMessage{},
				}
				index[item.GetNamespace()] = ns
			}

			ns.entries = append(ns.entries, raw)
			ns.index[item.GetID()] = raw
		}

		collection.mu.Lock()
		defer collection.mu.Unlock()

		collection.index = index
		collection.del = r.Delete
		collection.put = func(ctx context.Context, b []byte) error {
			var i I
			if err := json.Unmarshal(b, &i); err != nil {
				return err
			}

			collection.logger.Debug("Put",
				slog.String("namespace", string(i.GetNamespace())),
				slog.String("id", string(i.GetID())))

			if err := r.Put(ctx, i); err != nil {
				n, id := i.GetNamespace(), i.GetID()
				return fmt.Errorf("%s: item %s/%s: %w", f.GetType(), n, id, err)
			}
			return nil
		}

		return nil
	}

	return &collection, nil
}

func (c *Collection) Get(ctx context.Context, n Namespace, id ID, w io.Writer) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("Get",
		slog.String("namespace", string(n)),
		slog.String("id", string(id)))

	if ns, ok := c.index[n]; ok {
		if entry, ok := ns.index[id]; ok {
			_, err := w.Write(entry)
			return err
		}
	}

	return fmt.Errorf("%s: item %s/%s: not found", c.typ, n, id)
}

func (c *Collection) List(ctx context.Context, n Namespace, w io.Writer) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("List",
		slog.String("namespace", string(n)))

	if ns, ok := c.index[n]; ok {
		return json.NewEncoder(w).Encode(ns.entries)
	}

	return fmt.Errorf("%s: namespace %s: not found", c.typ, n)
}

func (c *Collection) Put(ctx context.Context, item []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.put(ctx, item)
}

func (c *Collection) Delete(ctx context.Context, n Namespace, id ID) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("Delete",
		slog.String("namespace", string(n)),
		slog.String("id", string(id)))

	return c.del(ctx, n, id)
}
