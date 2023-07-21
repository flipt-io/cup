package controller

type Controller struct{}

type GetRequest struct {
	FSConfig
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
}

type ListRequest struct {
	FSConfig
	Group     string
	Version   string
	Kind      string
	Namespace string
	Labels    [][2]string
}

type PutRequest struct{}

type DeleteRequest struct{}
