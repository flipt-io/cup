package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sync"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/oklog/ulid/v2"
	"github.com/xeipuuv/gojsonschema"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controllers"
	"golang.org/x/exp/slog"
)

// ViewFunc is a function provided to Source.View.
// It is provided with a read-only view of the target source.
type ViewFunc func(fs.FS) error

// UpdateFunc is a function passed to a Source implementation to be invoked
// over a provided FSConfig in a call to Update.
type UpdateFunc func(controllers.FSConfig) error

// Result is the result of performing an update on a target Source.
type Result struct {
	ID ulid.ULID
}

// Source is the abstraction around a target source filesystem.
// It is used by the API server to both read and propose changes based on the
// operations requested.
type Source interface {
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
	Get(context.Context, *controllers.GetRequest) (*core.Resource, error)
	List(context.Context, *controllers.ListRequest) ([]*core.Resource, error)
	Put(context.Context, *controllers.PutRequest) error
	Delete(context.Context, *controllers.DeleteRequest) error
}

// Server is the core api.Server for cupd.
// It handles exposing all the sources, definitions and the resources themselves.
type Server struct {
	mu          sync.RWMutex
	mux         *chi.Mux
	fs          Source
	definitions containers.MapStore[string, *core.ResourceDefinition]
	rev         string
}

// NewServer constructs and configures a new instance of *api.Server
// It uses the provided controller and filesystem store to build and serve
// requests for sources, definitions and resources.
func NewServer(fs Source) (*Server, error) {
	s := &Server{
		mux:         chi.NewMux(),
		fs:          fs,
		definitions: containers.MapStore[string, *core.ResourceDefinition]{},
		rev:         "main",
	}

	s.mux.Use(middleware.Logger)
	s.mux.Use(cors.AllowAll().Handler)

	s.mux.Get("/apis", s.handleSourceDefinitions)

	return s, nil
}

// ServeHTTP delegates to the underlying chi.Mux router.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.mux.ServeHTTP(w, r)
}

func (s *Server) addDefinition(def *core.ResourceDefinition, version string) {
	kind := path.Join(def.Spec.Group, version, def.Names.Plural)

	slog.Debug("Adding Definition", "kind", kind)

	s.definitions[kind] = def
}

// Register adds a new controller and definition with a particular filesystem to the server.
// This may happen dynamically in the future, so it is guarded with a write lock.
func (s *Server) Register(cntl Controller, def *core.ResourceDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for version, schema := range def.Spec.Versions {
		var (
			version = version
			prefix  = fmt.Sprintf("/apis/%s/%s/namespaces/{ns}/%s", def.Spec.Group, version, def.Names.Plural)
			named   = prefix + "/{name}"
		)

		// update sources map
		s.addDefinition(def, version)

		slog.Debug("Registering routes", "prefix", prefix)

		schema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schema))
		if err != nil {
			return err
		}

		// list kind
		s.mux.Get(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := s.fs.View(r.Context(), s.rev, func(f fs.FS) error {
				resources, err := cntl.List(r.Context(), &controllers.ListRequest{
					Request: controllers.Request{
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
			if err := s.fs.View(r.Context(), s.rev, func(f fs.FS) error {
				resource, err := cntl.Get(r.Context(), &controllers.GetRequest{
					Request: controllers.Request{
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
			var resource core.Resource
			if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			res, err := schema.Validate(gojsonschema.NewBytesLoader(resource.Spec))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if !res.Valid() {
				http.Error(w, fmt.Sprintf("%v", res.Errors()), http.StatusBadRequest)
				return
			}

			message := fmt.Sprintf(
				"feat: update %s/%s %s/%s",
				resource.APIVersion, resource.Kind,
				resource.Metadata.Namespace, resource.Metadata.Name,
			)

			result, err := s.fs.Update(r.Context(), s.rev, message, func(f controllers.FSConfig) error {
				return cntl.Put(r.Context(), &controllers.PutRequest{
					Request: controllers.Request{
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
			var (
				namespace = chi.URLParamFromCtx(r.Context(), "ns")
				name      = chi.URLParamFromCtx(r.Context(), "name")
				message   = fmt.Sprintf(
					"feat: delete %s/%s/%s %s/%s",
					def.Spec.Group, version, def.Names.Plural,
					namespace, name,
				)
			)

			result, err := s.fs.Update(r.Context(), s.rev, message, func(f controllers.FSConfig) error {
				return cntl.Delete(r.Context(), &controllers.DeleteRequest{
					Request: controllers.Request{
						Group:     def.Spec.Group,
						Version:   version,
						Kind:      def.Names.Kind,
						Namespace: namespace,
					},
					FSConfig: f,
					Name:     name,
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

	return nil
}

func (s *Server) handleSourceDefinitions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := json.NewEncoder(w).Encode(s.definitions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
