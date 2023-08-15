package tailscale

import (
	"context"
	"log/slog"
	"net/http"

	"go.flipt.io/cup/pkg/api/logger"
	"tailscale.com/client/tailscale/apitype"
)

// Client is a local Tailscale client.
type Client interface {
	WhoIs(ctx context.Context, remoteAddr string) (*apitype.WhoIsResponse, error)
}

// AddWhoIs wraps an [http.Handler] in a middleware that adds identity information to the
// [http.Request] context. The identity can be retrieved using [WhoIs].
func AddWhoIs(client Client) func(h http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			who, err := client.WhoIs(ctx, r.RemoteAddr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logger.EntrySet(r, slog.Group("tailscale",
				slog.String("user_login_name", who.UserProfile.LoginName),
				slog.String("user_id", who.UserProfile.ID.String()),
			))
			next.ServeHTTP(w, r.WithContext(setWhoIs(ctx, who)))
		})
	}
}

// WhoIs extracts Tailscale identity information from a context.Context and returns it.
func WhoIs(ctx context.Context) *apitype.WhoIsResponse {
	return ctx.Value(whoIsCtxKey{}).(*apitype.WhoIsResponse)
}

type whoIsCtxKey struct{}

func setWhoIs(ctx context.Context, who *apitype.WhoIsResponse) context.Context {
	return context.WithValue(ctx, whoIsCtxKey{}, who)
}
