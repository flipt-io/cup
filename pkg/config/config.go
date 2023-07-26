package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.flipt.io/cup/pkg/api/core"
)

type Config struct {
	API     API               `json:"api"`
	Sources map[string]Source `json:"sources"`
}

func Parse(path string) (_ *Config, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parsing config: %w", err)
		}
	}()

	fi, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer fi.Close()

	var conf Config
	if err = json.NewDecoder(fi).Decode(&conf); err != nil {
		return
	}

	return &conf, nil
}

type API struct {
	Address string `json:"address"`
}

type SourceType string

const (
	SourceTypeGit   = SourceType("git")
	SourceTypeLocal = SourceType("local")
)

type Source struct {
	Type      SourceType           `json:"type"`
	Local     *LocalSource         `json:"local,omitempty"`
	Git       *GitSource           `json:"git,omitempty"`
	Resources []ResourceDefinition `json:"resources"`
}

type LocalSource struct {
	Path string `json:"path"`
}

type SCMType string

const (
	SCMTypeGitea  = SCMType("gitea")
	SCMTypeGitHub = SCMType("github")
)

type GitSource struct {
	URL *url.URL `json:"url"`
	SCM SCMType  `json:"scm"`
}

func (s *GitSource) Credentials() (user, pass string) {
	pass, _ = s.URL.User.Password()
	return s.URL.User.Username(), pass
}

func (s *GitSource) Host() string {
	return fmt.Sprintf("%s://%s", s.URL.Scheme, s.URL.Host)
}

func (s *GitSource) OwnerRepo() (owner, repo string, err error) {
	parts := strings.SplitN(s.URL.Path, "/", 2)
	if len(parts) < 3 {
		return "", "", fmt.Errorf("unexpected path: %q", s.URL.Path)
	}

	return parts[1], strings.TrimSuffix(parts[2], ".git"), nil
}

type ResourceDefinition struct {
	Controller Controller               `json:"controller"`
	Path       *string                  `json:"path.omitempty"`
	Inline     *core.ResourceDefinition `json:"inline,omitempty"`
}

func (r ResourceDefinition) Definition() (*core.ResourceDefinition, error) {
	if r.Inline != nil {
		return r.Inline, nil
	}

	if r.Path == nil {
		return nil, errors.New("resource definition requires either path or inline definition")
	}

	fi, err := os.Open(*r.Path)
	if err != nil {
		return nil, err
	}

	defer fi.Close()

	var def core.ResourceDefinition
	if err := json.NewDecoder(fi).Decode(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

type ControllerType string

const (
	ControllerTypeTemplate = ControllerType("template")
	ControllerTypeWASM     = ControllerType("wasm")
)

type Controller struct {
	Type     ControllerType      `json:"type"`
	Template *TemplateController `json:"template,omitempty"`
	WASM     *WASMController     `json:"wasm,omitempty"`
}

type TemplateController struct {
}

type WASMController struct {
}
