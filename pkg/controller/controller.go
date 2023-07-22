package controller

import "go.flipt.io/cup/pkg/api/core"

type Controller struct{}

type Request struct {
	FSConfig
	Group     string
	Version   string
	Kind      string
	Namespace string
}

type GetRequest struct {
	Request
	Name string
}

type ListRequest struct {
	Request
	Labels [][2]string
}

type PutRequest struct {
	Request
	Resource *core.Resource
}

type DeleteRequest struct {
	Request
	Name string
}
