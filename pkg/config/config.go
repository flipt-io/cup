package config

type SourceType string

const (
	SourceTypeGit   = SourceType("git")
	SourceTypeLocal = SourceType("local")
)

type Source struct {
	Name  string       `json:"name"`
	Type  SourceType   `json:"type"`
	Git   *GitSource   `json:"git"`
	Local *LocalSource `json:"local"`
}

type GitSource struct {
	Resources []ResourceDefinition `json:"resources"`
}

type LocalSource struct {
	Path      string               `json:"path"`
	Resources []ResourceDefinition `json:"resources"`
}

type ResourceDefinition struct {
	Path string `json:"path"`
}
