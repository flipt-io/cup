package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controller"
)

// ViewFunc is a function provided to FilesystemStore.View.
// It is provided with a read-only view of the target filesystem.
type ViewFunc func(fs.FS) error

// UpdateFunc is a function passed to a FilesystemStore implementation to be invoked
// over a provided FSConfig in a call to Update.
type UpdateFunc func(controller.FSConfig) error

// Result is the result of performing an update on a target FilesystemStore.
type Result struct {
	ID ulid.ULID
}

// Filesystem is the abstraction around a target source filesystem.
// It is used by the API server to both read and propose changes based on the
// operations requested.
type Filesystem interface {
	// View invokes the provided function with an fs.FS which should enforce
	// a read-only view for the requested source and revision
	View(_ context.Context, revision string, fn ViewFunc) error
	// Update invokes the provided function with an FSConfig which can be written to
	// Any writes performed to the target during the execution of fn will be added,
	// comitted, pushed and proposed for review on a target SCM
	Update(_ context.Context, revision, message string, fn UpdateFunc) (*Result, error)
}

// Controller is the core controller interface for handling interactions with a
// single resource type.
type Controller interface {
	Definition() *core.ResourceDefinition
	Get(context.Context, *controller.GetRequest) (*core.Resource, error)
	List(context.Context, *controller.ListRequest) ([]*core.Resource, error)
	Put(context.Context, *controller.PutRequest) error
	Delete(context.Context, *controller.DeleteRequest) error
}

// Server is the core api.Server for cupd.
// It handles exposing all the sources, definitions and the resources themselves.
type Server struct {
	mu      sync.RWMutex
	mux     *chi.Mux
	sources map[string]map[string]*core.ResourceDefinition
	rev     string
}

// NewServer constructs and configures a new instance of *api.Server
// It uses the provided controller and filesystem store to build and serve
// requests for sources, definitions and resources.
func NewServer() (*Server, error) {
	s := &Server{
		mux:     chi.NewMux(),
		sources: map[string]map[string]*core.ResourceDefinition{},
		rev:     "main",
	}

	s.mux.Get("/apis", s.handleSources)
	s.mux.Get("/apis/{source}", s.handleSourceDefinitions)

	return s, nil
}

// ServeHTTP delegates to the underlying chi.Mux router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.mux.ServeHTTP(w, r)
}

func (s *Server) addDefinition(source string, gvk string, def *core.ResourceDefinition) {
	src, ok := s.sources[source]
	if !ok {
		src = map[string]*core.ResourceDefinition{}
		s.sources[source] = src
	}

	src[gvk] = def
}

// RegisterController adds a new controller and definition with a particular filesystem to the server.
// This may happen dynamically in the future, so it is guarded with a write lock.
func (s *Server) RegisterController(source string, fss Filesystem, cntl Controller) {
	s.mu.Lock()
	defer s.mu.Unlock()

	def := cntl.Definition()
	for version := range def.Spec.Versions {
		var (
			version = version
			prefix  = fmt.Sprintf("/apis/%s/%s/%s/%s/namespaces/{ns}", source, def.Spec.Group, version, def.Names.Plural)
			named   = prefix + "/{name}"
		)

		// update sources map
		s.addDefinition(source, path.Join(def.Spec.Group, version, def.Names.Kind), def)

		// list kind
		s.mux.Get(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := fss.View(r.Context(), s.rev, func(f fs.FS) error {
				resources, err := cntl.List(r.Context(), &controller.ListRequest{
					Request: controller.Request{
						Group:     def.Spec.Group,
						Version:   version,
						Kind:      def.Names.Kind,
						Namespace: chi.URLParamFromCtx(r.Context(), "ns"),
					},
					FS: f,
				})
				if err != nil {
					return err
				}

				enc := json.NewEncoder(w)
				for _, resource := range resources {
					if err := enc.Encode(resource); err != nil {
						return err
					}
				}

				return nil
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}))

		// get kind
		s.mux.Get(named, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := fss.View(r.Context(), s.rev, func(f fs.FS) error {
				resource, err := cntl.Get(r.Context(), &controller.GetRequest{
					Request: controller.Request{
						Group:     def.Spec.Group,
						Version:   version,
						Kind:      def.Names.Kind,
						Namespace: chi.URLParamFromCtx(r.Context(), "ns"),
					},
					FS:   f,
					Name: chi.URLParamFromCtx(r.Context(), "name"),
				})
				if err != nil {
					return err
				}

				return json.NewEncoder(w).Encode(resource)
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}))

		// put kind
		s.mux.Put(named, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO(georgemac): derive a suitable message
			var message string
			result, err := fss.Update(r.Context(), s.rev, message, func(f controller.FSConfig) error {
				var resource core.Resource
				if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
					return err
				}

				return cntl.Put(r.Context(), &controller.PutRequest{
					Request: controller.Request{
						Group:     def.Spec.Group,
						Version:   version,
						Kind:      def.Names.Kind,
						Namespace: chi.URLParamFromCtx(r.Context(), "ns"),
					},
					FSConfig: f,
					Name:     chi.URLParamFromCtx(r.Context(), "name"),
					Resource: &resource,
				})
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}))

		// delete kind
		s.mux.Delete(named, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO(georgemac): derive a suitable message
			var message string
			result, err := fss.Update(r.Context(), s.rev, message, func(f controller.FSConfig) error {
				return cntl.Delete(r.Context(), &controller.DeleteRequest{
					Request: controller.Request{
						Group:     def.Spec.Group,
						Version:   version,
						Kind:      def.Names.Kind,
						Namespace: chi.URLParamFromCtx(r.Context(), "ns"),
					},
					FSConfig: f,
					Name:     chi.URLParamFromCtx(r.Context(), "name"),
				})
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}))
	}
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	var sources []string
	for src := range s.sources {
		sources = append(sources, src)
	}

	sort.Strings(sources)

	if err := json.NewEncoder(w).Encode(&sources); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSourceDefinitions(w http.ResponseWriter, r *http.Request) {
	src := chi.URLParamFromCtx(r.Context(), "source")
	definitions, ok := s.sources[src]
	if !ok {
		http.Error(w, fmt.Sprintf("source not found: %q", src), http.StatusNotFound)
		return
	}

	if err := json.NewEncoder(w).Encode(&definitions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
