package main

import (
	"context"
	"net/http"
	"os"

	"go.flipt.io/fidgit"
	"go.flipt.io/fidgit/collections/flipt"
	"go.flipt.io/fidgit/internal/source/local"
	"golang.org/x/exp/slog"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := fidgit.NewService(local.New(ctx, "."))

	collection, err := fidgit.CollectionFor[flipt.Flag](context.Background(), &flipt.FlagCollectionFactory{})
	if err != nil {
		slog.Error("Building Collection", "error", err)
		os.Exit(1)
	}

	manager.RegisterCollection(collection)

	manager.Start(context.Background())

	server := fidgit.NewServer(manager)

	http.Handle("/api/v1/", server)

	slog.Info("Listening", slog.String("addr", ":9191"))

	http.ListenAndServe(":9191", nil)
}
