package fidgit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"golang.org/x/exp/slog"
)

type Source interface {
	Get(context.Context) (fs.FS, error)
	Propose(context.Context, ProposeRequest) error
}

type ProposeRequest struct {
	Revision string `json:"revision"`
	Changes  []File `json:"changes"`
}

type File struct {
	Path     string
	Contents []byte
}

type Proposal struct {
	Status string  `json:"status"`
	ID     *string `json:"id,omitempty"`
}

type Service struct {
	collections map[Type]*Collection
	source      Source

	once sync.Once
	done chan struct{}
}

func NewService(source Source) *Service {
	return &Service{
		source:      source,
		collections: map[Type]*Collection{},
		done:        make(chan struct{}),
	}
}

func (m *Service) RegisterCollection(c *Collection) {
	m.collections[c.typ] = c
}

func (m *Service) Start(ctx context.Context) error {
	err := errors.New("manager already started")

	m.once.Do(func() {
		err = m.update(ctx)
		go func() {
			defer close(m.done)
			for {
				if err := ctx.Err(); err != nil {
					return
				}

				if err := m.update(ctx); err != nil {
					slog.Error("Updating collections", "error", err)
				}
			}
		}()
	})

	return err
}

func (m *Service) collection(typ Type) (*Collection, error) {
	c, ok := m.collections[typ]
	if !ok {
		return nil, fmt.Errorf("collection %q: not found", typ)
	}

	return c, nil
}

func (m *Service) Get(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, err := m.collection(typ)
	if err != nil {
		return nil, err
	}

	return c.Get(ctx, ns, id)
}

func (m *Service) List(ctx context.Context, typ Type, ns Namespace) ([]byte, error) {
	c, err := m.collection(typ)
	if err != nil {
		return nil, err
	}

	return c.List(ctx, ns)
}

func (m *Service) Put(ctx context.Context, typ Type, ns Namespace, body []byte) ([]byte, error) {
	c, err := m.collection(typ)
	if err != nil {
		return nil, err
	}

	changes, err := c.Put(ctx, ns, body)
	if err != nil {
		return nil, err
	}

	if err := m.source.Propose(ctx, ProposeRequest{
		Changes: changes,
	}); err != nil {
		return nil, err
	}

	return json.Marshal(&Proposal{
		Status: "done",
	})
}

func (m *Service) Delete(ctx context.Context, typ Type, ns Namespace, id ID) ([]byte, error) {
	c, err := m.collection(typ)
	if err != nil {
		return nil, err
	}

	changes, err := c.Delete(ctx, ns, id)
	if err != nil {
		return nil, err
	}

	if err := m.source.Propose(ctx, ProposeRequest{
		Changes: changes,
	}); err != nil {
		return nil, err
	}

	return json.Marshal(&Proposal{
		Status: "done",
	})
}

func (m *Service) update(ctx context.Context) error {
	fs, err := m.source.Get(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, c := range m.collections {
		if err := c.updateSnapshot(ctx, fs); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (m *Service) Wait() {
	<-m.done
}
