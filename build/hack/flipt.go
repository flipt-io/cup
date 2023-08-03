package hack

import (
	"context"

	"dagger.io/dagger"
)

func FliptCup(ctx context.Context, client *dagger.Client, base *dagger.Container, platform dagger.Platform) (*dagger.Container, error) {
	flipt := base.WithEnvVariable("GOOS", "wasip1").
		WithEnvVariable("GOARCH", "wasm").
		WithExec([]string{
			"go", "build", "-o", "flipt.wasm", "./ext/controllers/flipt.io/v1alpha1/cmd/flipt/...",
		}).
		File("flipt.wasm")

	return client.
		Container(dagger.ContainerOpts{Platform: platform}).
		From("alpine:3.18").
		WithExec([]string{"mkdir", "-p", "/var/run/cupd"}).
		WithWorkdir("/var/run/cupd").
		WithDirectory("/etc/cupd/config", client.Host().Directory("build/hack/config")).
		WithFile("/etc/cupd/config/flipt.wasm", flipt).
		WithFile("/usr/local/bin/cupd", base.File("/usr/local/bin/cupd")).
		WithFile("/usr/local/bin/cup", base.File("/usr/local/bin/cup")).
		WithDefaultArgs(dagger.ContainerWithDefaultArgsOpts{
			Args: []string{"/usr/local/bin/cupd", "serve", "-api-resources", "/etc/cupd/config"},
		}).
		Sync(ctx)
}
