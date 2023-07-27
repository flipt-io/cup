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
				Name:  "ctl",
				Usage: "Access the resource API of a cupd instance",
				Subcommands: []*cli.Command{
					{
						Name:    "sources",
						Aliases: []string{"s"},
						Usage:   "List the available sources",
						Action: func(ctx *cli.Context) error {
							cfg, err := parseConfig(ctx)
							if err != nil {
								return err
							}

							return sources(cfg, http.DefaultClient)
						},
					},
					{
						Name:    "definitions",
						Aliases: []string{"d"},
						Usage:   "List the available resource definitions for a target source",
						Action: func(ctx *cli.Context) error {
							cfg, err := parseConfig(ctx)
							if err != nil {
								return err
							}

							return definitions(cfg, http.DefaultClient)
						},
					},
					{
						Name:    "list",
						Aliases: []string{"l"},
						Usage:   "List the available resources of a given definition for the target source",
						Action: func(ctx *cli.Context) error {
							cfg, err := parseConfig(ctx)
							if err != nil {
								return err
							}

							return list(cfg, http.DefaultClient, ctx.Args().First())
						},
					},
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
