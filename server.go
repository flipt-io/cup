package fidgit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/exp/slog"
)

const apiV1Prefix = "/api/v1/"

type Server struct {
	*http.ServeMux

	service *Service
}

func NewServer(service *Service) *Server {
	return &Server{
		ServeMux: &http.ServeMux{},
		service:  service,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	logger := slog.With(slog.String("system", "server"))

	path := strings.TrimPrefix(r.URL.Path, apiV1Prefix)
	parts := strings.SplitN(path, "/", 5)
	if len(parts) < 4 {
		logger.Debug("Unexpected URL", slog.String("path", path))
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var (
		typ = Type{
			Group:   parts[0],
			Kind:    parts[1],
			Version: parts[2],
		}
		ns = parts[3]
		id string
	)

	if len(parts) > 4 {
		id = parts[4]
	}

	var (
		data []byte
		err  error
	)

	switch r.Method {
	case http.MethodGet:
		if id == "" {
			data, err = s.service.List(r.Context(), typ, Namespace(ns))
			if err != nil {
				logger.Error("List", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			data, err = s.service.Get(r.Context(), typ, Namespace(ns), ID(id))
			if err != nil {
				logger.Error("Get", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Put", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err = s.service.Put(r.Context(), typ, Namespace(ns), body)
		if err != nil {
			logger.Error("Put", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		if id == "" {
			http.Error(w, "delete: missing ID", http.StatusBadRequest)
			return
		}

		data, err = s.service.Delete(r.Context(), typ, Namespace(ns), ID(id))
		if err != nil {
			logger.Error("Delete", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, fmt.Sprintf("method %q", r.Method), http.StatusMethodNotAllowed)
		return
	}

	_, _ = io.Copy(w, bytes.NewReader(data))
	return
}
