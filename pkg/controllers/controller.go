package controllers

import (
	"io/fs"
	"path"

	"go.flipt.io/cup/pkg/api/core"
)

type Controller struct{}

type Request struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
}

func (r Request) String() string {
	return path.Join(r.Group, r.Version, r.Kind, r.Namespace)
}

type GetRequest struct {
	Request
	FS   fs.FS
	Name string
}

type ListRequest struct {
	Request
	FS     fs.FS
	Labels [][2]string
}

type PutRequest struct {
	Request
	FSConfig
	Name     string
	Resource *core.Resource
}

type DeleteRequest struct {
	Request
	FSConfig
	Name string
}
