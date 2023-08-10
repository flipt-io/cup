package config

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	API       API
	Tailscale Tailscale
}

func (c *Config) FlagSet() *flag.FlagSet {
	set := flag.NewFlagSet("serve", flag.ContinueOnError)
	set.StringVar(&c.API.Address, "api-address", ":8181", "server listen address")
	set.StringVar(&c.API.Source.Type, "api-source", "local", "source type (one of [local, git])")
	set.StringVar(&c.API.Source.Local.Path, "api-local-path", ".", "path to local source directory")
	set.StringVar(&c.API.Source.Git.URL, "api-git-repo", "", "target git repository URL")
	set.StringVar(&c.API.Source.Git.SCM, "api-git-scm", "github", "SCM type (one of [github, gitea])")
	set.StringVar(&c.API.Resources, "api-resources", ".", "path to server configuration directory (controllers, definitions and bindings)")

	// Tailscale
	set.StringVar(&c.Tailscale.Hostname, "tailscale-hostname", "", "hostname to expose on Tailscale")
	set.StringVar(&c.Tailscale.AuthKey, "tailscale-auth-key", "", "Tailscale auth key (optional)")
	set.BoolVar(&c.Tailscale.Ephemeral, "tailscale-ephemeral", false, "join the network as an ephemeral node (optional)")

	return set
}

// Tailscale is configuration for [tsnet.Server].
type Tailscale struct {
	Hostname  string
	AuthKey   string
	Ephemeral bool
}

type API struct {
	Address   string
	Source    Source
	Resources string
}

type Source struct {
	Type  string      `json:"type"`
	Local LocalSource `json:"local,omitempty"`
	Git   GitSource   `json:"git,omitempty"`
}

type LocalSource struct {
	Path string `json:"path"`
}

type GitSource struct {
	URL string `json:"url"`
	SCM string `json:"scm"`
}

type GitURL struct {
	*url.URL
}

func ParseGitURL(v string) (_ *GitURL, err error) {
	g := &GitURL{}
	g.URL, err = url.Parse(v)
	return g, err
}

func (u *GitURL) Credentials() (user, pass string) {
	pass, _ = u.User.Password()
	return u.User.Username(), pass
}

func (u *GitURL) Host() string {
	return fmt.Sprintf("%s://%s", u.Scheme, u.URL.Host)
}

func (u *GitURL) OwnerRepo() (owner, repo string, err error) {
	parts := strings.SplitN(u.Path, "/", 3)
	if len(parts) < 3 {
		return "", "", fmt.Errorf("unexpected path: %q", u.Path)
	}

	return parts[1], strings.TrimSuffix(parts[2], ".git"), nil
}
