package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

func main() {
	dir, err := ensureConfigDir()
	if err != nil {
		slog.Error("Exiting", "error", err)
		os.Exit(1)
	}

	app := &cli.App{
		Name: "cup",
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
				Usage:    "List the available resource definitions for a target source",
				Action: func(ctx *cli.Context) error {
					cfg, err := parseConfig(ctx)
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
					cfg, err := parseConfig(ctx)
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
						Usage:       "Path to source to apply to target cupd",
						DefaultText: "(- STDIN)",
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg, err := parseConfig(ctx)
					if err != nil {
						return err
					}

					return apply(cfg,
						http.DefaultClient,
						ctx.String("f"),
					)
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

	if err := json.NewEncoder(fi).Encode(defaultConfig()); err != nil {
		return "", err
	}

	return cfgDir, nil
}
