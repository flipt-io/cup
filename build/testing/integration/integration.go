package integration

import (
	"context"

	"dagger.io/dagger"
	"gopkg.in/yaml.v2"
)

const (
	SCMUser     = "cup"
	SCMPassword = "password"
	SCMEmail    = "dev@cup.flipt.io"
)

func SCM(ctx context.Context, client *dagger.Client, dir *dagger.Directory) (*dagger.Container, string, error) {
	gitea := client.Container().
		From("gitea/gitea:latest").
		WithExposedPort(3000)

	endp, err := gitea.Endpoint(ctx, dagger.ContainerEndpointOpts{
		Scheme: "http",
	})
	if err != nil {
		return nil, "", err
	}

	gitea = gitea.WithExec(nil)

	conf := config{
		URL: endp,
		Admin: admin{
			Username: SCMUser,
			Password: SCMPassword,
			Email:    SCMEmail,
		},
		Repositories: []repository{
			{
				Name: "config",
				Contents: []content{
					{
						Path:    "/work/base",
						Message: "feat: initial commit",
					},
				},
			},
		},
	}

	contents, err := yaml.Marshal(&conf)
	if err != nil {
		return nil, "", err
	}

	_, err = client.Container().
		From("ghcr.io/flipt-io/stew:latest").
		WithWorkdir("/work").
		WithDirectory("/work/base", dir).
		WithNewFile("/etc/stew/config.yml", dagger.ContainerWithNewFileOpts{
			Contents: string(contents),
		}).
		WithServiceBinding("gitea", gitea).
		WithExec(nil).
		Sync(ctx)
	if err != nil {
		return nil, "", err
	}

	return gitea, endp, nil
}

type config struct {
	URL          string       `yaml:"url"`
	Admin        admin        `json:"admin"`
	Repositories []repository `json:"repositories"`
}

type admin struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type repository struct {
	Name     string    `json:"name"`
	Contents []content `json:"contents"`
}

type content struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}
