package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/urfave/cli/v2"
)

const (
	goBuildCachePath = "/root/.cache/go-build"
	goModCachePath   = "/go/pkg/mod"
)

func main() {
	app := &cli.App{
		Name: "build",
		Commands: []*cli.Command{
			{
				Name: "build",
				Action: func(ctx *cli.Context) error {
					return build(ctx.Context)
				},
			},
			{
				Name: "test",
				Subcommands: []*cli.Command{
					{
						Name: "unit",
						Action: func(ctx *cli.Context) error {
							return test(ctx.Context)
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func test(ctx context.Context) error {
	return withBase(ctx, func(ctx context.Context, client *dagger.Client, base *dagger.Container) error {
		_, err := base.
			WithEnvVariable("CGO_ENABLED", "1").
			WithExec([]string{"go", "test", "-race", "-v", "./..."}).
			Sync(ctx)
		return err
	})
}

func build(ctx context.Context) error {
	return withBase(ctx, func(ctx context.Context, client *dagger.Client, base *dagger.Container) error {
		_, err := base.
			WithExec([]string{"go", "install", "./..."}).
			Sync(ctx)
		return err
	})
}

func withBase(ctx context.Context, fn func(ctx context.Context, client *dagger.Client, base *dagger.Container) error) error {
	return withClient(ctx, func(ctx context.Context, client *dagger.Client) error {
		base := client.Container().
			From("golang:1.21rc3-alpine3.18").
			WithEnvVariable("GOCACHE", goBuildCachePath).
			WithEnvVariable("GOMODCACHE", goModCachePath).
			WithExec([]string{"apk", "add", "gcc", "build-base"}).
			WithMountedDirectory("/src", client.Host().Directory(".")).
			WithWorkdir("/src")

		sumContents, err := base.File("go.work.sum").Contents(ctx)
		if err != nil {
			return err
		}

		sum := fmt.Sprintf("%x", sha256.Sum256([]byte(sumContents)))
		var (
			cacheGoBuild = client.CacheVolume(fmt.Sprintf("go-build-%s", sum))
			cacheGoMod   = client.CacheVolume(fmt.Sprintf("go-mod-%s", sum))
		)

		return fn(ctx, client, base.
			WithMountedCache(goBuildCachePath, cacheGoBuild).
			WithMountedCache(goModCachePath, cacheGoMod).
			WithExec([]string{"go", "install", "golang.org/dl/gotip@latest"}))
	})
}

func withClient(ctx context.Context, fn func(ctx context.Context, client *dagger.Client) error) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	return fn(ctx, client)
}
