package main

import (
	"context"
	"flag"
	"net/http"
	"net/url"
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"go.flipt.io/fidgit"
	"go.flipt.io/fidgit/internal/runtime"
	"go.flipt.io/fidgit/internal/source/git"
	"go.flipt.io/fidgit/internal/source/local"
	"golang.org/x/exp/slog"
)

var (
	sourceType = flag.String("source", "local", "source type (local|git)")
	gitRepo    = flag.String("repository", "", "target upstream repository")
	authBasic  = flag.String("auth-basic", "", "basic authentication in the form username:password")
)

func main() {
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var source fidgit.Source
	switch *sourceType {
	case "local":
		source = local.New(ctx, ".")
	case "git":
		url, err := url.Parse(*gitRepo)
		if err != nil {
			logger.Error("Parsing Git URL", slog.String("url", *gitRepo), "error", err)
		}

		var auth transport.AuthMethod
		if url.User.Username() != "" {
			password, _ := url.User.Password()
			auth = &githttp.BasicAuth{
				Username: url.User.Username(),
				Password: password,
			}
		}

		// strip basic auth creds once they're configured
		// via the transport auth
		url.User = nil

		source, err = git.NewSource(ctx, url.String(), git.WithAuth(auth))
		if err != nil {
			logger.Error("Building Git Source", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("Source Unknown", slog.String("source", *sourceType))

	}

	manager, err := fidgit.NewService(source)
	if err != nil {
		slog.Error("Building Manager", "error", err)
		os.Exit(1)
	}

	factory, err := runtime.NewFactory(ctx, "flipt.wasm")
	if err != nil {
		slog.Error("Configuring Runtime", "error", err)
		os.Exit(1)
	}

	manager.RegisterFactory(factory.Build())

	manager.Start(context.Background())

	server := fidgit.NewServer(manager)

	http.Handle("/api/v1/", server)

	slog.Info("Listening", slog.String("addr", ":9191"))

	http.ListenAndServe(":9191", nil)
}
