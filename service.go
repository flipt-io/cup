package cup

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
	Path     string `json:"path"`
	Message  string `json:"message"`
	Contents []byte `json:"contents"`
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
	cache  *lru.Cache[cacheKey, Collection]

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

	cache, err := lru.New[cacheKey, Collection](2)
	if err != nil {
		return nil, err
	}

	service.cache = cache

	return service, nil
}

func (s *Service) RegisterFactory(typ Type, fn FactoryFunc) {
	s.latest[typ] = &entry{fn: fn}
}

func (s *Service) Start(ctx context.Context) error {
	err := errors.New("manager already started")

	s.once.Do(func() {
		err = s.updateCache(ctx)
		go func() {
			defer close(s.done)
			for {
				if err := ctx.Err(); err != nil {
					return
				}

				if err := s.updateCache(ctx); err != nil {
					slog.Error("Updating collections", "error", err)
				}
			}
		}()
	})

	return err
}

func (s *Service) collection(typ Type, rev string) (Collection, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rev == "" {
		entry, ok := s.latest[typ]
		if !ok {
			return nil, "", fmt.Errorf("collection %q: not found", typ)
		}

		rev = entry.revision
	}

	c, ok := s.cache.Get(cacheKey{typ, rev})
	if !ok {
		return nil, "", fmt.Errorf("collection %q (%s): not found", typ, rev)
	}

	return c, rev, nil
}

func (s *Service) Get(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, _, err := s.collection(typ, "")
	if err != nil {
		return nil, err
	}

	entry, err := c.Get(ctx, ns, id)
	if err != nil {
		return nil, err
	}

	return json.Marshal(entry)
}

func (s *Service) List(ctx context.Context, typ Type, ns Namespace) ([]byte, error) {
	c, _, err := s.collection(typ, "")
	if err != nil {
		return nil, err
	}

	entries, err := c.List(ctx, ns)
	if err != nil {
		return nil, err
	}

	return json.Marshal(entries)
}

type ServicePutRequest struct {
	Revision *string
	Entry    *Entry
}

func (s *Service) Put(ctx context.Context, typ Type, req ServicePutRequest) ([]byte, error) {
	var rev string
	if req.Revision != nil {
		rev = *req.Revision
	}

	c, rev, err := s.collection(typ, rev)
	if err != nil {
		return nil, err
	}

	changes, err := c.Put(ctx, req.Entry)
	if err != nil {
		return nil, fmt.Errorf("putting item: %w", err)
	}

	proposal, err := s.source.Propose(ctx, ProposeRequest{
		Changes:  changes,
		Revision: rev,
	})
	if err != nil {
		return nil, fmt.Errorf("proposing changes: %w", err)
	}

	return json.Marshal(proposal)
}

func (s *Service) Delete(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, rev, err := s.collection(typ, "")
	if err != nil {
		return nil, err
	}

	changes, err := c.Delete(ctx, ns, id)
	if err != nil {
		return nil, err
	}

	propsal, err := s.source.Propose(ctx, ProposeRequest{
		Changes:  changes,
		Revision: rev,
	})
	if err != nil {
		return nil, err
	}

	return json.Marshal(propsal)
}

func (s *Service) updateCache(ctx context.Context) error {
	fs, revision, err := s.source.Get(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for t, entry := range s.latest {
		entry.revision = revision

		c, err := entry.fn(ctx, fs)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		s.cache.Add(cacheKey{t, revision}, c)
	}

	return errors.Join(errs...)
}

func (s *Service) Wait() {
	<-s.done
}
