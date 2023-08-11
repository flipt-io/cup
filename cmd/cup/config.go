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

			wr := writer()
			fmt.Fprintln(wr, "NAME\tADDRESS\tNAMESPACE\tCURRENT\t")
			for name, ctx := range cfg.Contexts {
				var current string
				if name == cfg.CurrentContext {
					current = "*"
				}

				namespace := "default"
				if ctx.Namespace != "" {
					namespace = ctx.Namespace
				}

				fmt.Fprintf(wr, "%s\t%s\t%s\t%s\t\n", name, ctx.Address, namespace, current)
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

					return config.Set(ctx, key, value)
				},
			},
		},
	}
}
