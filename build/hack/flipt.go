package hack

import (
	"context"

	"dagger.io/dagger"
)

func FliptController(ctx context.Context, client *dagger.Client, base *dagger.Container) func(*dagger.Container) *dagger.Container {
	flipt := base.WithEnvVariable("GOOS", "wasip1").
		WithEnvVariable("GOARCH", "wasm").
		WithExec([]string{
			"go", "build", "-o", "flipt.wasm", "./ext/controllers/flipt.io/v1alpha1/cmd/flipt/...",
		}).
		File("flipt.wasm")

	return func(c *dagger.Container) *dagger.Container {
		return c.
			WithDirectory("/etc/cupd/config", client.Host().Directory("build/hack/config")).
			WithFile("/etc/cupd/config/flipt.wasm", flipt)
	}
}
