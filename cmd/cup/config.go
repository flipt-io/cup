package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/cmd/cup/config"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:    "context",
		Aliases: []string{"ctx"},
		Action: func(ctx *cli.Context) error {
			cfg, err := config.Parse(ctx)
			if err != nil {
				return err
			}

			type namedContext struct {
				*config.Context
				Name string `json:"name"`
			}

			enc, err := encoder(cfg, func(c *namedContext) [][]string {
				var current string
				if c.Name == cfg.CurrentContext {
					current = "*"
				}

				namespace := "default"
				if c.Namespace != "" {
					namespace = c.Namespace
				}

				return [][]string{{c.Name, c.Address, namespace, current}}
			}, "NAME", "ADDRESS", "NAMESPACE", "CURRENT")
			if err != nil {
				return err
			}

			for name, ctx := range cfg.Contexts {
				if err := enc.Encode(&namedContext{Name: name, Context: ctx}); err != nil {
					return err
				}
			}

			return enc.Flush()
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

					return config.Set(ctx, key, value)
				},
			},
		},
	}
}
