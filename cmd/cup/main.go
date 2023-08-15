package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"

	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/cmd/cup/config"
)

func main() {
	dir, err := ensureConfigDir()
	if err != nil {
		slog.Error("Exiting", "error", err)
		os.Exit(1)
	}

	app := &cli.App{
		Name:  "cup",
		Usage: "Manage remote cupd instances",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   path.Join(dir, "config.json"),
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "table",
			},
			&cli.StringFlag{
				Name:    "address",
				Aliases: []string{"a"},
			},
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
			},
			&cli.StringFlag{
				Name:    "level",
				Aliases: []string{"l"},
				Usage:   "set the logging level (default: \"info\")",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Access the local configuration for the cup CLI.",
				Subcommands: []*cli.Command{
					configCommand(),
				},
			},
			{
				Name:     "definitions",
				Aliases:  []string{"defs"},
				Category: "discovery",
				Usage:    "List the available resource definitions",
				Action: func(ctx *cli.Context) error {
					cfg, err := config.Parse(ctx)
					if err != nil {
						return err
					}

					return definitions(cfg, http.DefaultClient)
				},
			},
			{
				Name:     "get",
				Category: "resource",
				Usage:    "Get one or more resources",
				Action: func(ctx *cli.Context) error {
					cfg, err := config.Parse(ctx)
					if err != nil {
						return err
					}

					return get(cfg,
						http.DefaultClient,
						ctx.Args().First(),
						ctx.Args().Tail()...)
				},
			},
			{
				Name:     "apply",
				Category: "resource",
				Usage:    "Put a resource from file on stdin",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "f",
						Value:       "-",
						Usage:       "Path to the resource being applied",
						DefaultText: "(- STDIN)",
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg, err := config.Parse(ctx)
					if err != nil {
						return err
					}

					rd := os.Stdin
					if source := ctx.String("f"); source != "-" {
						rd, err = os.Open(source)
						if err != nil {
							return err
						}
					}

					return apply(cfg,
						http.DefaultClient,
						rd,
					)
				},
			},
			{
				Name:      "edit",
				Category:  "resource",
				Usage:     "Edit a resource",
				ArgsUsage: "<type> <name>",
				Action: func(ctx *cli.Context) error {
					cfg, err := config.Parse(ctx)
					if err != nil {
						return err
					}

					if l := ctx.Args().Len(); l != 2 {
						return fmt.Errorf("expected 2 arguments, found %d", l)
					}

					return edit(cfg,
						http.DefaultClient,
						ctx.Args().Get(0),
						ctx.Args().Get(1))
				},
			},
			{
				Name:      "delete",
				Category:  "resource",
				Usage:     "Delete a resource",
				ArgsUsage: "<type> <name>",
				Action: func(ctx *cli.Context) error {
					cfg, err := config.Parse(ctx)
					if err != nil {
						return err
					}

					if l := ctx.Args().Len(); l != 2 {
						return fmt.Errorf("expected 2 arguments, found %d", l)
					}

					return del(cfg,
						http.DefaultClient,
						ctx.Args().Get(0),
						ctx.Args().Get(1))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Exiting", "error", err)
		os.Exit(1)
	}
}

func ensureConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	var (
		cfgDir  = path.Join(dir, "cup")
		cfgPath = path.Join(cfgDir, "config.json")
	)

	_, err = os.Stat(cfgPath)
	if err == nil {
		// config already exists so just return it
		return cfgDir, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	_, err = os.Stat(cfgDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		// make directory if it does not exist
		if err := os.MkdirAll(cfgDir, 0755); err != nil {
			return "", err
		}
	}

	// write out default config
	fi, err := os.Create(cfgPath)
	if err != nil {
		return "", err
	}

	defer fi.Close()

	if err := json.NewEncoder(fi).Encode(config.Default()); err != nil {
		return "", err
	}

	return cfgDir, nil
}
