package build

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path"

	"dagger.io/dagger"
	"github.com/containerd/containerd/platforms"
	"golang.org/x/sync/errgroup"
)

const (
	goBuildCachePath = "/root/.cache/go-build"
	goModCachePath   = "/go/pkg/mod"
)

// SupportedPlatforms is the set of platforms cup currently can be
// build for using these Dagger pipelines.
var SupportedPlatforms = []dagger.Platform{
	"linux/amd64",
	"linux/arm64",
}

type ContainerFn func(*dagger.Container) *dagger.Container

type Options struct {
	platforms        []dagger.Platform
	variantExtension ContainerFn
}

func defaultOptions() Options {
	return Options{
		platforms: SupportedPlatforms,
		variantExtension: func(c *dagger.Container) *dagger.Container {
			return c
		},
	}
}

type Option func(*Options)

func SetPlatform(platform dagger.Platform) Option {
	return func(bo *Options) {
		bo.platforms = []dagger.Platform{platform}
	}
}

func WithVariantExtention(fn ContainerFn) Option {
	return func(o *Options) {
		o.variantExtension = fn
	}
}

func Base(ctx context.Context, client *dagger.Client, opts ...Option) (*dagger.Container, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	base := client.Container().
		From("golang:1.21-alpine3.18").
		WithEnvVariable("GOCACHE", goBuildCachePath).
		WithEnvVariable("GOMODCACHE", goModCachePath).
		WithExec([]string{"apk", "add", "gcc", "build-base"}).
		WithDirectory("/src", client.Host().Directory(".", dagger.HostDirectoryOpts{
			Exclude: []string{
				"./docs/",
			},
		}), dagger.ContainerWithDirectoryOpts{
			Include: []string{
				"./build/",
				"./cmd/",
				"./pkg/",
				"./ext/",
				"./sdk/",
				"./go.work",
				"./go.work.sum",
				"./go.mod",
				"./go.sum",
			},
		}).
		WithWorkdir("/src")

	sumContents, err := base.File("go.work.sum").Contents(ctx)
	if err != nil {
		return nil, err
	}

	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(sumContents)))
	var (
		cacheGoBuild = client.CacheVolume(fmt.Sprintf("go-build-%s", sum))
		cacheGoMod   = client.CacheVolume(fmt.Sprintf("go-mod-%s", sum))
	)

	base = base.
		WithMountedCache(goBuildCachePath, cacheGoBuild).
		WithMountedCache(goModCachePath, cacheGoMod)

	type result struct {
		path   string
		binary *dagger.File
	}

	var (
		group errgroup.Group
		ch    = make(chan result)
	)

	for _, name := range []string{"cup", "cupd"} {
		for _, platform := range o.platforms {
			var (
				name     = name
				platform = platform
			)

			group.Go(func() error {
				p := platforms.MustParse(string(platform))

				binary := fmt.Sprintf("bin/%s/%s/%s", p.OS, p.Architecture, name)
				build, err := base.
					WithEnvVariable("GOOS", p.OS).
					WithEnvVariable("GOARCH", p.Architecture).
					WithExec([]string{"sh", "-c", fmt.Sprintf("go build -o %s ./cmd/%s/*.go", binary, name)}).
					Sync(ctx)
				if err != nil {
					return err
				}

				ch <- result{
					path:   binary,
					binary: build.File(binary),
				}

				return nil
			})
		}
	}

	go func() {
		defer close(ch)
		err = group.Wait()
	}()

	for result := range ch {
		base = base.WithFile(result.path, result.binary)
	}

	if err != nil {
		return nil, err
	}

	return base.Sync(ctx)
}

func Variants(ctx context.Context, client *dagger.Client, base *dagger.Container, opts ...Option) (variants []*dagger.Container, err error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	var (
		group errgroup.Group
		ch    = make(chan *dagger.Container)
	)

	for _, platform := range o.platforms {
		platform := platform
		group.Go(func() error {
			container, err := o.variantExtension(Variant(client, base, platform)).
				Sync(ctx)
			if err != nil {
				return err
			}

			ch <- container

			return nil
		})
	}

	go func() {
		defer close(ch)
		err = group.Wait()
	}()

	for container := range ch {
		variants = append(variants, container)
	}

	if err != nil {
		return nil, err
	}

	return
}

func Variant(client *dagger.Client, base *dagger.Container, platform dagger.Platform) *dagger.Container {
	var (
		p      = platforms.MustParse(string(platform))
		binDir = path.Join("bin", p.OS, p.Architecture)
	)

	return client.
		Container(dagger.ContainerOpts{Platform: platform}).
		From("alpine:3.18").
		WithExec([]string{"mkdir", "-p", "/var/run/cupd", "/etc/cupd/config"}).
		WithWorkdir("/var/run/cupd").
		WithFile("/usr/local/bin/cupd", base.File(path.Join(binDir, "cupd"))).
		WithFile("/usr/local/bin/cup", base.File(path.Join(binDir, "cup"))).
		WithDefaultArgs(dagger.ContainerWithDefaultArgsOpts{
			Args: []string{"/usr/local/bin/cupd", "serve", "-api-resources", "/etc/cupd/config"},
		})
}
