package testing

import (
	"context"
	"fmt"
	"time"

	"dagger.io/dagger"
	"github.com/containerd/containerd/platforms"
	"go.flipt.io/cup/build/testing/integration"
)

func Integration(ctx context.Context, client *dagger.Client, base, cupd *dagger.Container) error {
	platform, err := client.DefaultPlatform(ctx)
	if err != nil {
		return err
	}

	p := platforms.MustParse(string(platform))

	base = base.WithFile("/usr/local/bin/cup", base.File(fmt.Sprintf("bin/%s/%s/cup", p.OS, p.Architecture))).
		WithWorkdir("build/testing/template")

	cupd = cupd.WithMountedDirectory("/etc/cupd/config", base.Directory("testdata/config"))

	{
		pipeline := base.Pipeline("local template")

		cupd, err := cupdService(ctx, client, pipeline, cupd.
			WithMountedDirectory("/work", pipeline.Directory("testdata/base")).
			WithWorkdir("/work"))
		if err != nil {
			return err
		}

		_, err = base.
			WithServiceBinding("cupd", cupd).
			WithExec([]string{"go", "test", "-cup-address", "http://cupd:8181", "."}).
			Sync(ctx)

		if err != nil {
			return err
		}
	}

	{
		pipeline := base.Pipeline("git template")

		cupd, err := integration.SCM(
			ctx,
			client,
			cupd.WithEnvVariable("BUST", time.Now().String()),
			pipeline.Directory("testdata/base"),
		)
		if err != nil {
			return err
		}

		if cupd, err = cupdService(ctx, client, pipeline, cupd); err != nil {
			return err
		}

		_, err = base.
			WithServiceBinding("cupd", cupd).
			WithExec([]string{"go", "test", "-cup-address", "http://cupd:8181", "-cup-proposes", "."}).
			Sync(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func cupdService(ctx context.Context, client *dagger.Client, base, cupd *dagger.Container) (_ *dagger.Container, err error) {
	cupd, err = cupd.
		WithEnvVariable("BUST", time.Now().String()).
		WithExposedPort(8181).
		Sync(ctx)
	if err != nil {
		return nil, err
	}

	_, err = cupd.Endpoint(ctx, dagger.ContainerEndpointOpts{
		Scheme: "http",
	})
	if err != nil {
		return nil, err
	}

	return cupd.WithExec(nil), nil
}
