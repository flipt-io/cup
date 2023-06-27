package fidgit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/exp/slog"
)

type Source interface {
	Get(context.Context) (fs.FS, string, error)
	Propose(context.Context, ProposeRequest) (*Proposal, error)
}

type ProposeRequest struct {
	Revision string   `json:"revision"`
	Changes  []Change `json:"changes"`
}

type Change struct {
	Path     string
	Message  string
	Contents []byte
}

type Proposal struct {
	Status string  `json:"status"`
	ID     *string `json:"id,omitempty"`
}

type entry struct {
	revision string
	fn       FactoryFunc
}

type Service struct {
	source Source

	mu     sync.RWMutex
	latest map[Type]*entry
	cache  *lru.Cache[cacheKey, *Collection]

	once sync.Once
	done chan struct{}
}

type cacheKey struct {
	Type
	Revision string
}

func NewService(source Source) (*Service, error) {
	service := &Service{
		source: source,
		latest: map[Type]*entry{},
		done:   make(chan struct{}),
	}

	cache, err := lru.New[cacheKey, *Collection](2)
	if err != nil {
		return nil, err
	}

	service.cache = cache

	return service, nil
}

func (m *Service) RegisterFactory(typ Type, fn FactoryFunc) {
	m.latest[typ] = &entry{fn: fn}
}

func (m *Service) Start(ctx context.Context) error {
	err := errors.New("manager already started")

	m.once.Do(func() {
		err = m.updateCache(ctx)
		go func() {
			defer close(m.done)
			for {
				if err := ctx.Err(); err != nil {
					return
				}

				if err := m.updateCache(ctx); err != nil {
					slog.Error("Updating collections", "error", err)
				}
			}
		}()
	})

	return err
}

func (m *Service) collection(typ Type, rev string) (*Collection, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if rev == "" {
		entry, ok := m.latest[typ]
		if !ok {
			return nil, "", fmt.Errorf("collection %q: not found", typ)
		}

		rev = entry.revision
	}

	c, ok := m.cache.Get(cacheKey{typ, rev})
	if !ok {
		return nil, "", fmt.Errorf("collection %q (%s): not found", typ, rev)
	}

	return c, rev, nil
}

func (m *Service) Get(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, _, err := m.collection(typ, "")
	if err != nil {
		return nil, err
	}

	return c.Get(ctx, ns, id)
}

func (m *Service) List(ctx context.Context, typ Type, ns Namespace) ([]byte, error) {
	c, _, err := m.collection(typ, "")
	if err != nil {
		return nil, err
	}

	return c.List(ctx, ns)
}

func (m *Service) Put(ctx context.Context, typ Type, req CollectionPutRequest) ([]byte, error) {
	var rev string
	if req.Revision != nil {
		rev = *req.Revision
	}

	c, rev, err := m.collection(typ, rev)
	if err != nil {
		return nil, err
	}

	changes, err := c.Put(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("putting item: %w", err)
	}

	proposal, err := m.source.Propose(ctx, ProposeRequest{
		Changes:  changes,
		Revision: rev,
	})
	if err != nil {
		return nil, fmt.Errorf("proposing changes: %w", err)
	}

	return json.Marshal(proposal)
}

func (m *Service) Delete(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, rev, err := m.collection(typ, "")
	if err != nil {
		return nil, err
	}

	changes, err := c.Delete(ctx, ns, id)
	if err != nil {
		return nil, err
	}

	propsal, err := m.source.Propose(ctx, ProposeRequest{
		Changes:  changes,
		Revision: rev,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(propsal)
}

func (m *Service) updateCache(ctx context.Context) error {
	fs, revision, err := m.source.Get(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for t, entry := range m.latest {
		entry.revision = revision

		c, err := entry.fn(ctx, fs)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		m.cache.Add(cacheKey{t, revision}, c)
	}

	return errors.Join(errs...)
}

func (m *Service) Wait() {
	<-m.done
}
