package main

import (
	"context"
	"net/http"
	"os"

	"go.flipt.io/fidgit"
	"go.flipt.io/fidgit/collections/flipt"
	"golang.org/x/exp/slog"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	server := fidgit.NewServer()

	collection, err := fidgit.CollectionFor[*flipt.Flag](context.Background(), &flipt.FlagCollectionFactory{})
	if err != nil {
		slog.Error("Building Collection", "error", err)
		os.Exit(1)
	}

	server.RegisterCollection(collection)

	http.Handle("/", server)

	slog.Info("Listening", slog.String("addr", ":9191"))

	http.ListenAndServe(":9191", nil)
}
