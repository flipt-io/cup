package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	require := require.New(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		EntrySet(r, "foo", "bar")
		fmt.Fprintln(w, "ok")
	})

	sh := &mockSlogHandler{}

	handler := New(sh)(innerHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("User-Agent", "testing")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(http.StatusOK, recorder.Code)
	require.Equal("ok\n", recorder.Body.String())

	m := map[string]string{}
	for _, a := range sh.attrs {
		m[a.Key] = a.Value.String()
	}

	require.Equal(map[string]string{
		"foo":         "bar",
		"http_method": "GET",
		"http_proto":  "HTTP/1.1",
		"http_scheme": "http",
		"remote_addr": "127.0.0.1",
		"uri":         "http://example.com/",
		"user_agent":  "testing",
	}, m)
}

type mockSlogHandler struct {
	attrs []slog.Attr
}

func (s *mockSlogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (s *mockSlogHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (s *mockSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	s.attrs = append(s.attrs, attrs...)
	return s
}

func (s *mockSlogHandler) WithGroup(name string) slog.Handler {
	panic(name)
}
