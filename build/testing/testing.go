package testing

import (
	"context"
	"time"

	"dagger.io/dagger"
	"go.flipt.io/cup/build/testing/integration"
)

func Integration(ctx context.Context, client *dagger.Client, base, cupd *dagger.Container) error {
	base = base.WithWorkdir("build/testing/template")
	cupd = cupd.WithMountedDirectory("/etc/cupd/config", base.Directory("testdata/config"))

	{
		pipeline := base.Pipeline("local template")

		cupd := cupd.
			WithEnvVariable("BUST", time.Now().String()).
			WithMountedDirectory("/work", pipeline.Directory("testdata/base")).
			WithWorkdir("/work")

		if err := template(ctx, client, pipeline, cupd); err != nil {
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

		if err := template(ctx, client, pipeline, cupd); err != nil {
			return err
		}
	}

	return nil
}

func template(ctx context.Context, client *dagger.Client, base, cupd *dagger.Container) (err error) {
	cupd, err = cupd.
		WithEnvVariable("BUST", time.Now().String()).
		WithExposedPort(8181).
		Sync(ctx)
	if err != nil {
		return err
	}

	_, err = cupd.Endpoint(ctx, dagger.ContainerEndpointOpts{
		Scheme: "http",
	})
	if err != nil {
		return err
	}

	_, err = base.
		WithServiceBinding("cupd", cupd.WithExec(nil)).
		WithExec([]string{"go", "test", "-cup-address", "http://cupd:8181", "."}).
		Sync(ctx)

	return err
}
