package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
)

func TestWhoIs(t *testing.T) {
	require := require.New(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		whois := WhoIs(r.Context())
		require.Equal(tailcfg.UserID(123456), whois.UserProfile.ID)
		require.Equal("brettbuddin", whois.UserProfile.LoginName)
		fmt.Fprintln(w, "ok")
	})

	client := &mockClient{
		whois: &apitype.WhoIsResponse{
			UserProfile: &tailcfg.UserProfile{
				ID:        tailcfg.UserID(123456),
				LoginName: "brettbuddin",
			},
		},
	}
	handler := AddWhoIs(client)(innerHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(http.StatusOK, recorder.Code)
	require.Equal("ok\n", recorder.Body.String())
}

func TestWhoIs_Error(t *testing.T) {
	require := require.New(t)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	client := &mockClient{
		err: fmt.Errorf("oops!"),
	}
	handler := AddWhoIs(client)(innerHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(http.StatusInternalServerError, recorder.Code)
	require.Equal("oops!\n", recorder.Body.String())
}

type mockClient struct {
	whois *apitype.WhoIsResponse
	err   error
}

func (c *mockClient) WhoIs(ctx context.Context, remoteAddr string) (*apitype.WhoIsResponse, error) {
	return c.whois, c.err
}
