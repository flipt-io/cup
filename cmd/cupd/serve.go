package main

import (
	"fmt"
	"net/http"
	"os"

	"code.gitea.io/sdk/gitea"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v53/github"
	"github.com/urfave/cli/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/config"
	"go.flipt.io/cup/pkg/controllers/template"
	"go.flipt.io/cup/pkg/controllers/wasm"
	"go.flipt.io/cup/pkg/source/git"
	scmgitea "go.flipt.io/cup/pkg/source/git/scm/gitea"
	scmgithub "go.flipt.io/cup/pkg/source/git/scm/github"
	"go.flipt.io/cup/pkg/source/local"
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

	var fs api.Source

	src := cfg.API.Source
	switch src.Type {
	case config.SourceTypeGit:
		user, pass := src.Git.Credentials()
		owner, repo, err := src.Git.OwnerRepo()
		if err != nil {
			return err
		}

		var scm git.SCM
		switch src.Git.SCM {
		case config.SCMTypeGitea:
			client, err := gitea.NewClient(src.Git.Host(), gitea.SetBasicAuth(user, pass))
			if err != nil {
				return err
			}

			scm = scmgitea.New(client, owner, repo)
		case config.SCMTypeGitHub:
			tp := github.BasicAuthTransport{
				Username: user,
				Password: pass,
			}

			client := github.NewClient(tp.Client())
			// resolve actual user name from API
			user, _, err := client.Users.Get(ctx.Context, "")
			if err != nil {
				return err
			}

			scm = scmgithub.New(client, owner, repo, user.GetName())
		default:
			return fmt.Errorf("scm type not supported: %q", src.Git.SCM)
		}

		fs, err = git.NewSource(ctx.Context, scm, src.Git.URL.String(), git.WithAuth(
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

	srv, err := api.NewServer(fs)
	if err != nil {
		return err
	}

	defs, err := cfg.DefinitionsByType()
	if err != nil {
		return err
	}

	for typ, binding := range cfg.API.Resources {
		var controller api.Controller

		controllerConf, ok := cfg.Controllers[binding.Controller]
		if !ok {
			return fmt.Errorf("unexpected controller: %q", binding.Controller)
		}

		def, err := defs.Get(typ)
		if err != nil {
			return err
		}

		switch controllerConf.Type {
		case config.ControllerTypeTemplate:
			controller = template.New()
		case config.ControllerTypeWASM:
			exec, err := os.ReadFile(controllerConf.WASM.Executable)
			if err != nil {
				return err
			}

			controller = wasm.New(ctx.Context, exec)
		default:
			return fmt.Errorf("controller type not supported: %q", controllerConf.Type)
		}

		if err := srv.Register(controller, def); err != nil {
			return fmt.Errorf("registering resource %q: %w", typ, err)
		}

		slog.Debug("Registered Resource Controller", "kind", typ, "controller", binding.Controller)
	}

	slog.Info("Listening...", "address", cfg.API.Address)

	return http.ListenAndServe(cfg.API.Address, srv)
}
