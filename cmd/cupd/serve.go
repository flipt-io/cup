package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"code.gitea.io/sdk/gitea"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v53/github"
	"go.flipt.io/cup/pkg/api"
	apiconfig "go.flipt.io/cup/pkg/api/config"
	"go.flipt.io/cup/pkg/config"
	"go.flipt.io/cup/pkg/source/git"
	scmgitea "go.flipt.io/cup/pkg/source/git/scm/gitea"
	scmgithub "go.flipt.io/cup/pkg/source/git/scm/github"
	"go.flipt.io/cup/pkg/source/local"
	"tailscale.com/tsnet"
)

func serve(ctx context.Context, cfg *config.Config) error {
	var fs api.Source

	src := cfg.API.Source
	switch src.Type {
	case "git":
		gitURL, err := config.ParseGitURL(src.Git.URL)
		if err != nil {
			return err
		}

		user, pass := gitURL.Credentials()
		owner, repo, err := gitURL.OwnerRepo()
		if err != nil {
			return err
		}

		var scm git.SCM
		switch src.Git.SCM {
		case "gitea":
			client, err := gitea.NewClient(gitURL.Host(), gitea.SetBasicAuth(user, pass))
			if err != nil {
				return err
			}

			scm = scmgitea.New(client, owner, repo)
		case "github":
			tp := github.BasicAuthTransport{
				Username: user,
				Password: pass,
			}

			client := github.NewClient(tp.Client())
			// resolve actual user name from API
			user, _, err := client.Users.Get(ctx, "")
			if err != nil {
				return err
			}

			scm = scmgithub.New(client, owner, repo, user.GetName())
		default:
			return fmt.Errorf("scm type not supported: %q", src.Git.SCM)
		}

		fs, err = git.NewSource(ctx, scm, gitURL.String(), git.WithAuth(
			&githttp.BasicAuth{
				Username: user,
				Password: pass,
			},
		))
		if err != nil {
			return err
		}
	case "local":
		fs = local.New(src.Local.Path)
	}

	apiConfig, err := apiconfig.New(ctx, cfg)
	if err != nil {
		return err
	}

	var listener net.Listener
	if cfg.Tailscale.Hostname == "" {
		listener, err = net.Listen("tcp", cfg.API.Address)
		if err != nil {
			return err
		}
		defer listener.Close()
	} else {
		s := &tsnet.Server{
			Hostname:  cfg.Tailscale.Hostname,
			AuthKey:   cfg.Tailscale.AuthKey,
			Ephemeral: cfg.Tailscale.Ephemeral,
		}
		defer s.Close()

		listener, err = s.Listen("tcp", cfg.API.Address)
		if err != nil {
			return err
		}
		defer listener.Close()

		apiConfig.TailscaleClient, err = s.LocalClient()
		if err != nil {
			return err
		}
	}

	srv, err := api.NewServer(fs, apiConfig)
	if err != nil {
		return err
	}

	slog.Info("Listening...", "address", listener.Addr())
	return http.Serve(listener, srv)
}
