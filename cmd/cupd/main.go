package main

import (
	"context"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"go.flipt.io/cup/pkg/config"
	"golang.org/x/exp/slog"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	cfg := &config.Config{}
	root := &ffcli.Command{
		Name: "cupd",
		Subcommands: []*ffcli.Command{
			{
				Name:       "serve",
				ShortUsage: "cupd serve [flags]",
				ShortHelp:  "Run the cupd server",
				FlagSet:    cfg.FlagSet(),
				Exec: func(ctx context.Context, args []string) error {
					return serve(ctx, cfg)
				},
			},
		},
	}

	if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		slog.Error("Exiting...", "error", err)
		os.Exit(1)
	}
}

const banner = `
     ) )
    ( (
  |======|
  |      |
  | cupd |
  '------'`
