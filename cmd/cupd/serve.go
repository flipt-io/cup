package main

import (
	"fmt"
	"net/http"
	"os"

	"code.gitea.io/sdk/gitea"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/config"
	"go.flipt.io/cup/pkg/controllers/template"
	"go.flipt.io/cup/pkg/controllers/wasm"
	"go.flipt.io/cup/pkg/fs/git"
	scmgitea "go.flipt.io/cup/pkg/fs/git/scm/gitea"
	"go.flipt.io/cup/pkg/fs/local"
	"golang.org/x/exp/slog"
)

func serve(ctx *cli.Context) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	cfg, err := config.Parse(ctx.String("config"))
	if err != nil {
		return err
	}

	srv, err := api.NewServer()
	if err != nil {
		return err
	}

	for k, src := range cfg.Sources {
		var fs api.Filesystem

		switch src.Type {
		case config.SourceTypeGit:
			user, pass := src.Git.Credentials()

			var scm git.SCM
			switch src.Git.SCM {
			case config.SCMTypeGitea:
				owner, repo, err := src.Git.OwnerRepo()
				if err != nil {
					return err
				}

				client, err := gitea.NewClient(src.Git.Host(), gitea.SetBasicAuth(user, pass))
				if err != nil {
					return err
				}

				scm = scmgitea.New(client, owner, repo)
			default:
				return fmt.Errorf("scm type not supported: %q", src.Git.SCM)
			}

			fs, err = git.NewFilesystem(ctx.Context, scm, src.Git.URL.String(), git.WithAuth(
				&githttp.BasicAuth{
					Username: user,
					Password: pass,
				},
			))
			if err != nil {
				return err
			}
		case config.SourceTypeLocal:
			fs = local.New(src.Local.Path)
		}

		for _, resource := range src.Resources {
			var controller api.Controller
			def, err := resource.Definition()
			if err != nil {
				return err
			}

			switch resource.Controller.Type {
			case config.ControllerTypeTemplate:
				controller = template.New()
			case config.ControllerTypeWASM:
				exec, err := os.ReadFile(resource.Controller.WASM.Executable)
				if err != nil {
					return err
				}

				controller = wasm.New(ctx.Context, exec)
			default:
				return fmt.Errorf("controller type not supported: %q", resource.Controller.Type)
			}

			slog.Debug("Registering Controller", "apiVersion", def.APIVersion, "kind", def.Kind)

			srv.Register(k, fs, controller, def)
		}
	}

	slog.Info("Listening...", "address", cfg.API.Address)

	return http.ListenAndServe(cfg.API.Address, srv)
}
