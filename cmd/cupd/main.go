package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

func main() {
	app := &cli.App{
		Name: "cupd",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "cupd.json",
				Usage:   "Parse configuration from `FILE`",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "run the cupd server",
				Action:  serve,
			},
		},
	}

	fmt.Println(banner)

	if err := app.Run(os.Args); err != nil {
		slog.Error("Exiting", "error", err)
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
