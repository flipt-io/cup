package testing

import (
	"context"

	"dagger.io/dagger"
)

func Integration(ctx context.Context, client *dagger.Client, base, cup *dagger.Container) error {
	base = base.WithWorkdir("build/testing/template")
	cup = cup.
		WithMountedDirectory("/etc/cupd/config", base.Directory("testdata/config")).
		WithMountedDirectory("/work", base.Directory("testdata/base")).
		WithWorkdir("/work")

	_, err := template(ctx, client, base, cup).
		WithExec(nil).
		Sync(ctx)

	return err
}

func template(ctx context.Context, client *dagger.Client, base, cup *dagger.Container) *dagger.Container {
	return base.
		WithServiceBinding("cupd", cup.WithExec(nil)).
		WithDefaultArgs(dagger.ContainerWithDefaultArgsOpts{
			Args: []string{"go", "test", "-cup-address", "http://cupd:8181", "."},
		})
}
