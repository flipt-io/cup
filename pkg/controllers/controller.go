package controllers

import (
	"io/fs"

	"go.flipt.io/cup/pkg/api/core"
)

type Controller struct{}

type Request struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
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
