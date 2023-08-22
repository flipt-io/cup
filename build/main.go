package main

import (
	"context"
	"fmt"
	"os"

	"dagger.io/dagger"
	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/build/build"
	"go.flipt.io/cup/build/hack"
	"go.flipt.io/cup/build/testing"
	"golang.org/x/exp/slog"
)

const (
	goBuildCachePath = "/root/.cache/go-build"
	goModCachePath   = "/go/pkg/mod"
)

var (
	publishFlags = []cli.Flag{
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
	}
)

func main() {
	app := &cli.App{
		Name: "build",
		Commands: []*cli.Command{
			{
				Name: "base",
				Action: func(ctx *cli.Context) error {
					return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
						_, err := build.Base(ctx.Context, client)
						return err
					})
				},
			},
			{
				Name: "image",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Required: true,
						Usage:    "output file path for OCI tarball",
					},
				},
				Action: func(ctx *cli.Context) error {
					return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
						base, err := build.Base(ctx.Context, client)
						if err != nil {
							return err
						}

						_, err = build.Variant(client, base, platform).Export(
							ctx.Context,
							ctx.String("output"),
						)

						return err
					})
				},
			},
			{
				Name:  "publish",
				Flags: publishFlags,
				Action: func(ctx *cli.Context) error {
					return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
						variants, err := buildVariants(ctx, client)
						if err != nil {
							return err
						}

						tag, err := publishVariants(ctx, client, variants)
						if err != nil {
							return err
						}

						slog.Info("Published Image", "tag", tag)

						return nil
					})
				},
			},
			{
				Name: "test",
				Subcommands: []*cli.Command{
					{
						Name: "unit",
						Action: func(ctx *cli.Context) error {
							return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
								// fix tests to just build and run against platform in context
								base, err := build.Base(ctx.Context, client, build.SetPlatform(platform))
								if err != nil {
									return err
								}

								_, err = base.WithEnvVariable("CGO_ENABLED", "1").
									WithExec([]string{"go", "test", "-race", "-v", "./..."}).
									Sync(ctx.Context)

								return err
							})
						},
					},
					{
						Name: "integration",
						Action: func(ctx *cli.Context) error {
							return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
								// fix tests to just build and run against platform in context
								base, err := build.Base(ctx.Context, client, build.SetPlatform(platform))
								if err != nil {
									return err
								}

								cup, err := build.Variant(client, base, platform).Sync(ctx.Context)
								if err != nil {
									return err
								}

								return testing.Integration(ctx.Context, client, base, cup)
							})
						},
					},
				},
			},
			{
				Name: "hack",
				Subcommands: []*cli.Command{
					{
						Name:        "fliptcup",
						Description: "Build cup images with baked-in flipt controller",
						Subcommands: []*cli.Command{
							{
								Name: "image",
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "output",
										Aliases:  []string{"o"},
										Required: true,
										Usage:    "output file path for OCI tarball",
									},
								},
								Action: func(ctx *cli.Context) error {
									return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
										base, err := build.Base(ctx.Context, client)
										if err != nil {
											return err
										}

										_, err = hack.FliptController(
											ctx.Context,
											client,
											base,
										)(build.Variant(client, base, platform)).Export(
											ctx.Context,
											ctx.String("output"),
										)

										return err
									})
								},
							},
							{
								Name:  "publish",
								Flags: publishFlags,
								Action: func(ctx *cli.Context) error {
									return withClient(ctx.Context, func(client *dagger.Client, platform dagger.Platform) error {
										base, err := build.Base(ctx.Context, client)
										if err != nil {
											return err
										}

										variants, err := build.Variants(ctx.Context, client, base, build.WithVariantExtention(
											hack.FliptController(ctx.Context, client, base),
										))
										if err != nil {
											return err
										}

										tag, err := publishVariants(ctx, client, variants)
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
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func publishVariants(ctx *cli.Context, client *dagger.Client, variants []*dagger.Container) (string, error) {
	var (
		registry = ctx.String("registry")
		username = ctx.String("username")
		password = ctx.String("password")
	)

	return client.Container().WithRegistryAuth(
		registry,
		username,
		client.SetSecret("registry-password", password),
	).Publish(ctx.Context,
		fmt.Sprintf("%s/%s/%s", registry, username, ctx.String("image-name")),
		dagger.ContainerPublishOpts{
			PlatformVariants: variants,
		},
	)

}

func buildVariants(ctx *cli.Context, client *dagger.Client, opts ...build.Option) ([]*dagger.Container, error) {
	base, err := build.Base(ctx.Context, client)
	if err != nil {
		return nil, err
	}

	return build.Variants(ctx.Context, client, base, opts...)
}

func withClient(ctx context.Context, fn func(client *dagger.Client, platform dagger.Platform) error) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
	if err != nil {
		return err
	}
	defer client.Close()

	platform, err := client.DefaultPlatform(ctx)
	if err != nil {
		return err
	}

	return fn(client, platform)
}
