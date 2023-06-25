package fidgit

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"golang.org/x/exp/slog"
)

const apiV1Prefix = "/api/v1/"

type Server struct {
	*http.ServeMux
}

func NewServer() *Server {
	return &Server{ServeMux: &http.ServeMux{}}
}

func (s *Server) RegisterCollection(c *Collection) {
	prefix := path.Join(apiV1Prefix, c.typ.Kind, c.typ.Version) + "/"

	slog.Debug("Registering Collection", slog.String("path", prefix))

	s.Handle(prefix, http.StripPrefix(prefix, &collectionServer{c}))
}

type collectionServer struct {
	*Collection
}

func (c *collectionServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	logger := slog.With(slog.String("system", "server"))

	ns, id, _ := strings.Cut(r.URL.Path, "/")
	if ns == "" {
		http.Error(w, "namespace must be provided", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if id == "" {
			if err := c.List(r.Context(), Namespace(ns), w); err != nil {
				logger.Error("List", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if err := c.Get(r.Context(), Namespace(ns), ID(id), w); err != nil {
			logger.Error("Get", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Put", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := c.Put(r.Context(), body); err != nil {
			logger.Error("Put", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodDelete:
		if id == "" {
			http.Error(w, "delete: missing ID", http.StatusBadRequest)
			return
		}

		if err := c.Delete(r.Context(), Namespace(ns), ID(id)); err != nil {
			logger.Error("Delete", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	http.Error(w, fmt.Sprintf("method %q", r.Method), http.StatusMethodNotAllowed)
	return
}
