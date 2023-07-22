package simple

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"

	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controller"
	"go.flipt.io/cup/pkg/encoding"
)

type ResourceEncoding interface {
	Extension() string
	NewEncoder(io.Writer) encoding.TypedEncoder[core.Resource]
	NewDecoder(io.Reader) encoding.TypedDecoder[core.Resource]
}

// Controller is mostly used for testing purposes (for now).
// It is a built-in controller implementation for cup.
// It simply organizes resources on the underlying filesystem by { namespace }/{ name }
// encoding them using the provided marshaller.
type Controller struct {
	definition *core.ResourceDefinition
	encoding   ResourceEncoding
}

// New constructs and configures a new *Controller.
// By default it uses a JSON encoding which can be overriden via WithResourceEncoding.
func New(def *core.ResourceDefinition, opts ...containers.Option[Controller]) *Controller {
	controller := &Controller{
		definition: def,
		encoding:   encoding.NewJSONEncoding[core.Resource](),
	}

	containers.ApplyAll(controller, opts...)

	return controller
}

// WithResourceEncoding overrides the default resource encoding.
func WithResourceEncoding(e ResourceEncoding) containers.Option[Controller] {
	return func(c *Controller) {
		c.encoding = e
	}
}

// Definition returns the core resource definition handled by the Controller.
func (c *Controller) Definition() *core.ResourceDefinition {
	return c.definition
}

func (c *Controller) Get(_ context.Context, req *controller.GetRequest) (*core.Resource, error) {
	fi, err := req.FSConfig.ToFS().Open(path.Join(req.Namespace, req.Name+"."+c.encoding.Extension()))
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}
	defer fi.Close()

	return c.encoding.NewDecoder(fi).Decode()
}

// List finds all the resources on the provided FS in the folder { namespace }
// The result set is filtered by any specified labels.
func (c *Controller) List(_ context.Context, req *controller.ListRequest) (resources []*core.Resource, _ error) {
	ffs := req.FSConfig.ToFS()
	return resources, fs.WalkDir(req.FSConfig.ToFS(), req.Namespace, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return fs.SkipDir
		}

		if ext := path.Ext(p); ext == "" || ext[1:] != c.encoding.Extension() {
			// skip files without expected extension
			return nil
		}

		fi, err := ffs.Open(p)
		if err != nil {
			return err
		}

		defer fi.Close()

		resource, err := c.encoding.NewDecoder(fi).Decode()
		if err != nil {
			return err
		}

		for _, kv := range req.Labels {
			// skip adding resource if any of the specified labels
			// do not match as expected
			if v, ok := resource.Metadata.Labels[kv[0]]; !ok || v != kv[1] {
				return nil
			}
		}

		resources = append(resources, resource)

		return nil
	})
}

// Put for now is a silent noop as we dont have a writable filesystem abstraction
func (c *Controller) Put(_ context.Context, _ *controller.PutRequest) error {
	return nil
}

// Delete for now is a silent noop as we dont have a writable filesystem abstraction
func (c *Controller) Delete(_ context.Context, _ *controller.DeleteRequest) error {
	return nil
}
