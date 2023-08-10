package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"

	"dagger.io/dagger"
	"github.com/containerd/containerd/platforms"
	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/build/hack"
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
			{
				Name: "hack",
				Subcommands: []*cli.Command{
					{
						Name: "fliptcup:build",
						Action: func(ctx *cli.Context) error {
							return withBase(ctx.Context, func(client *dagger.Client, base *dagger.Container, platform dagger.Platform) error {
								cup, err := hack.FliptCup(ctx.Context, client, base, platform)
								if err != nil {
									return err
								}

								_, err = cup.Export(ctx.Context, "fliptcup.tar")
								return err
							})
						},
					},
					{
						Name: "fliptcup:publish",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "registry",
								Value:   "ghcr.io",
								EnvVars: []string{"CUP_BUILD_REGISTRY"},
							},
							&cli.StringFlag{
								Name:    "username",
								Value:   "flipt-io",
								EnvVars: []string{"CUP_BUILD_USERNAME"},
							},
							&cli.StringFlag{
								Name:    "password",
								Value:   "",
								EnvVars: []string{"CUP_BUILD_PASSWORD"},
							},
							&cli.StringFlag{
								Name:    "image-name",
								Value:   "cup/flipt:latest",
								EnvVars: []string{"CUP_BUILD_IMAGE_NAME"},
							},
						},
						Action: func(ctx *cli.Context) error {
							var (
								variants  []*dagger.Container
								platforms = []dagger.Platform{
									"linux/amd64",
									"linux/arm64",
								}
								build = func(client *dagger.Client, base *dagger.Container, platform dagger.Platform) error {
									cup, err := hack.FliptCup(ctx.Context, client, base, platform)
									if err != nil {
										return err
									}

									variants = append(variants, cup)

									return nil
								}
							)

							for _, platform := range platforms {
								if err := withBase(ctx.Context, build, withPlatform(platform)); err != nil {
									return err
								}
							}

							return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
								var (
									registry = ctx.String("registry")
									username = ctx.String("username")
									password = ctx.String("password")
								)

								tag, err := client.Container().WithRegistryAuth(
									registry,
									username,
									client.SetSecret("registry-password", password),
								).Publish(ctx.Context,
									fmt.Sprintf("%s/%s/%s", registry, username, ctx.String("image-name")),
									dagger.ContainerPublishOpts{
										PlatformVariants: variants,
									},
								)
								if err != nil {
									return err
								}

								slog.Info("Published Image", "tag", tag)

								return nil
							})
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
	return withBase(ctx, func(client *dagger.Client, base *dagger.Container, platform dagger.Platform) error {
		_, err := base.
			WithEnvVariable("CGO_ENABLED", "1").
			WithExec([]string{"go", "test", "-race", "-v", "./..."}).
			Sync(ctx)
		return err
	})
}

func build(ctx context.Context) error {
	return withBase(ctx, func(client *dagger.Client, base *dagger.Container, platform dagger.Platform) error {
		_, err := base.
			WithExec([]string{"go", "install", "./..."}).
			Sync(ctx)
		return err
	})
}

func withBase(ctx context.Context, fn func(client *dagger.Client, base *dagger.Container, platform dagger.Platform) error, opts ...option) error {
	return withClient(ctx, func(client *dagger.Client, platform dagger.Platform) error {
		p := platforms.MustParse(string(platform))

		base := client.Container().
			From("golang:1.21rc3-alpine3.18").
			WithEnvVariable("GOCACHE", goBuildCachePath).
			WithEnvVariable("GOMODCACHE", goModCachePath).
			WithEnvVariable("GOOS", p.OS).
			WithEnvVariable("GOARCH", p.Architecture).
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

		return fn(client, base.
			WithMountedCache(goBuildCachePath, cacheGoBuild).
			WithMountedCache(goModCachePath, cacheGoMod).
			WithExec([]string{"go", "build", "-o", "/usr/local/bin/cupd", "./cmd/cupd/..."}).
			WithExec([]string{"go", "build", "-o", "/usr/local/bin/cup", "./cmd/cup/..."}),
			platform,
		)
	}, opts...)
}

type config struct {
	platform dagger.Platform
}

type option func(*config)

func withPlatform(platform dagger.Platform) option {
	return func(c *config) {
		c.platform = platform
	}
}

func withClient(ctx context.Context, fn func(client *dagger.Client, platform dagger.Platform) error, opts ...option) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	config := config{}
	config.platform, err = client.DefaultPlatform(ctx)
	if err != nil {
		return err
	}

	for _, opt := range opts {
		opt(&config)
	}

	return fn(client, config.platform)
}
