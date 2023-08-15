package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/peterbourgon/ff/v3/ffyaml"
	"go.flipt.io/cup/pkg/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	set := flag.NewFlagSet("cupd", flag.ContinueOnError)
	_ = set.String("config", "", "server config file")

	cfg := &config.Config{}
	root := &ffcli.Command{
		Name:    "cupd",
		FlagSet: set,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("CUPD"),
			ff.WithConfigFileParser((&ffyaml.ParseConfig{
				Delimiter: "-",
			}).Parse),
			ff.WithConfigFileFlag("config"),
		},
		Subcommands: []*ffcli.Command{
			{
				Name:       "serve",
				ShortUsage: "cupd serve [flags]",
				ShortHelp:  "Run the cupd server",
				FlagSet:    cfg.FlagSet(),
				Options: []ff.Option{
					ff.WithEnvVarPrefix("CUPD"),
					ff.WithConfigFileParser((&ffyaml.ParseConfig{
						Delimiter: "-",
					}).Parse),
					ff.WithConfigFileFlag("config"),
				},
				Exec: func(ctx context.Context, args []string) error {
					return serve(ctx, cfg)
				},
			},
		},
	}

	if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			slog.Error("Exiting...", "error", err)
			os.Exit(1)
		}
	}
}
