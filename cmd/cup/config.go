package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:    "context",
		Aliases: []string{"ctx"},
		Action: func(ctx *cli.Context) error {
			cfg, err := parseConfig(ctx)
			if err != nil {
				return err
			}

			wr := writer()
			fmt.Fprintln(wr, "NAME\tADDRESS\tSOURCE\tNAMESPACE\tCURRENT\t")
			for name, ctx := range cfg.Contexts {
				var current string
				if name == cfg.CurrentContext {
					current = "*"
				}

				namespace := "default"
				if ctx.Namespace != "" {
					namespace = ctx.Namespace
				}

				fmt.Fprintf(wr, "%s\t%s\t%s\t%s\t%s\t\n", name, ctx.Address, ctx.Source, namespace, current)
			}
			return wr.Flush()
		},
		Subcommands: []*cli.Command{
			{
				Name: "set",
				Action: func(ctx *cli.Context) error {
					key, value, match := strings.Cut(ctx.Args().First(), "=")
					if !match {
						return fmt.Errorf(
							"expected argument in the form key=value, found %q",
							ctx.Args().First(),
						)
					}

					return configSet(ctx, key, value)
				},
			},
		},
	}
}

type config struct {
	CurrentContext string              `json:"current_context"`
	Contexts       map[string]*context `json:"contexts"`
}

func defaultConfig() config {
	return config{
		CurrentContext: "default",
		Contexts: map[string]*context{
			"default": {
				Address:   "http://localhost:8181",
				Source:    "default",
				Namespace: "default",
			},
		},
	}
}

func (c config) Address() string {
	return c.Contexts[c.CurrentContext].Address
}

func (c config) Source() string {
	return c.Contexts[c.CurrentContext].Source
}

func (c config) Namespace() string {
	current := c.Contexts[c.CurrentContext]
	if current.Namespace != "" {
		return current.Namespace
	}

	return "default"
}

type context struct {
	Address   string `json:"address"`
	Source    string `json:"source"`
	Namespace string `json:"namespace"`
}

func parseConfig(ctx *cli.Context) (config, error) {
	fi, err := os.Open(ctx.String("config"))
	if err != nil {
		return config{}, err
	}

	defer fi.Close()

	var conf config
	if err := json.NewDecoder(fi).Decode(&conf); err != nil {
		return config{}, err
	}

	return conf, nil
}

func configSet(ctx *cli.Context, key, value string) error {
	fi, err := os.OpenFile(ctx.String("config"), os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("opening config: %w", err)
	}

	defer fi.Close()

	var conf config
	if err := json.NewDecoder(fi).Decode(&conf); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	switch key {
	case "address":
		conf.Contexts[conf.CurrentContext].Address = value
	case "source":
		conf.Contexts[conf.CurrentContext].Source = value
	case "namespace":
		conf.Contexts[conf.CurrentContext].Namespace = value
	default:
		return fmt.Errorf("unknown config key: %q (should be one of [address, source, namespace])", key)
	}

	if err := fi.Truncate(0); err != nil {
		return fmt.Errorf("truncating config: %w", err)
	}

	if _, err := fi.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking config: %w", err)
	}

	return json.NewEncoder(fi).Encode(&conf)
}
