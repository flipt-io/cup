package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
)

// New returns a middleware function that logs requests using structure logging.
func New(handler slog.Handler) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(
		&slogger{handler: handler},
	)
}

// EntrySet extends request's [*slog.Logger] with additional fields. Argumennts are converted to
// attributes as if by [(*slog.Logger).Log].
func EntrySet(r *http.Request, attrs ...any) {
	if entry, ok := middleware.GetLogEntry(r).(*logEntry); ok {
		entry.logger = entry.logger.With(attrs...)
	}
}

type slogger struct {
	handler slog.Handler
}

func (l *slogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	ctx := r.Context()

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	attrs := []slog.Attr{
		slog.String("http_scheme", scheme),
		slog.String("http_proto", r.Proto),
		slog.String("http_method", r.Method),
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
		slog.String("uri", fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)),
	}

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		attrs = append(attrs, slog.String("req_id", reqID))
	}

	entry := &logEntry{
		context: ctx,
		logger:  slog.New(l.handler.WithAttrs(attrs)),
	}
	entry.logger.Log(ctx, slog.LevelInfo, "request started")
	return entry
}

type logEntry struct {
	context context.Context
	logger  *slog.Logger
}

func (l *logEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra any) {
	l.logger.LogAttrs(l.context, slog.LevelInfo, "request complete",
		slog.Int("resp_status", status),
		slog.Int("resp_byte_length", bytes),
		slog.Float64("resp_elapsed_ms", float64(elapsed.Nanoseconds())/1000000.0),
	)
}

func (l *logEntry) Panic(v any, stack []byte) {
	l.logger.LogAttrs(l.context, slog.LevelInfo, "",
		slog.String("stack", string(stack)),
		slog.String("panic", fmt.Sprintf("%+v", v)),
	)
}
