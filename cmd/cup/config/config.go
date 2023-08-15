package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

type Config struct {
	CurrentContext string              `json:"current_context"`
	Output         string              `json:"-"`
	Contexts       map[string]*Context `json:"contexts"`
}

func Default() Config {
	return Config{
		CurrentContext: "default",
		Contexts: map[string]*Context{
			"default": {
				Address:   "http://localhost:8181",
				Namespace: "default",
			},
		},
	}
}

func (c Config) Address() string {
	return c.Contexts[c.CurrentContext].Address
}

func (c Config) Namespace() string {
	current := c.Contexts[c.CurrentContext]
	if current.Namespace != "" {
		return current.Namespace
	}

	return "default"
}

type Context struct {
	Address   string `json:"address"`
	Namespace string `json:"namespace"`
}

func Parse(ctx *cli.Context) (Config, error) {
	fi, err := os.Open(ctx.String("config"))
	if err != nil {
		return Config{}, err
	}

	defer fi.Close()

	var conf Config
	if err := json.NewDecoder(fi).Decode(&conf); err != nil {
		return Config{}, err
	}

	conf.Output = ctx.String("output")
	current := conf.Contexts[conf.CurrentContext]
	if ctx.IsSet("address") {
		current.Address = ctx.String("address")
	}

	if ctx.IsSet("namespace") {
		current.Namespace = ctx.String("namespace")
	}

	var level slog.Level
	if l := ctx.String("level"); l != "" {
		if err := level.UnmarshalText([]byte(l)); err != nil {
			return conf, err
		}
	}

	switch conf.Output {
	case "json":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})))
	default:
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})))
	}

	return conf, nil
}

func Set(ctx *cli.Context, key, value string) error {
	fi, err := os.OpenFile(ctx.String("config"), os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("opening config: %w", err)
	}

	defer fi.Close()

	var conf Config
	if err := json.NewDecoder(fi).Decode(&conf); err != nil {

		return fmt.Errorf("parsing config: %w", err)
	}

	switch key {
	case "address":
		conf.Contexts[conf.CurrentContext].Address = value
	case "namespace":
		conf.Contexts[conf.CurrentContext].Namespace = value
	default:
		return fmt.Errorf("unknown config key: %q (should be one of [address, namespace])", key)
	}

	if err := fi.Truncate(0); err != nil {
		return fmt.Errorf("truncating config: %w", err)
	}

	if _, err := fi.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking config: %w", err)
	}

	return json.NewEncoder(fi).Encode(&conf)
}
