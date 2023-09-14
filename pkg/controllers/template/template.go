package template

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controllers"
	"go.flipt.io/cup/pkg/encoding"
	"golang.org/x/exp/slog"
)

const (
	defaultListTmpl     = `{{ .Namespace }}/{{ .Group }}-{{ .Version }}-{{ .Kind }}-*.json`
	defaultResourceTmpl = `{{ .Namespace }}/{{ .Group }}-{{ .Version }}-{{ .Kind }}-{{ .Name }}.json`
)

var funcs = template.FuncMap{
	"replace": strings.ReplaceAll,
}

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
	encoding     ResourceEncoding
	listTmpl     *template.Template
	resourceTmpl *template.Template
}

// New constructs and configures a new *Controller.
// By default it uses a JSON encoding which can be overriden via WithResourceEncoding.
func New(opts ...containers.Option[Controller]) *Controller {
	controller := &Controller{
		encoding: &encoding.JSONEncoding[core.Resource]{
			Prefix: "",
			Indent: "  ",
		},
		listTmpl: template.Must(template.New("list").
			Funcs(funcs).
			Parse(defaultListTmpl),
		),
		resourceTmpl: template.Must(template.New("resource").
			Funcs(funcs).
			Parse(defaultResourceTmpl),
		),
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

// WithListTemplate lets you override the default namespace template
// for identifying which files should be respected and returned for a given
// resource type and namespace
func WithListTemplate(tmpl string) containers.Option[Controller] {
	if tmpl == "" {
		return func(c *Controller) {}
	}

	return func(c *Controller) {
		c.listTmpl = template.Must(template.New("list").
			Funcs(funcs).
			Parse(tmpl),
		)
	}
}

// WithResourceTemplate lets you override the default resource template
// for identifying which file relates to the requested resource
func WithResourceTemplate(tmpl string) containers.Option[Controller] {
	if tmpl == "" {
		return func(c *Controller) {}
	}

	return func(c *Controller) {
		c.resourceTmpl = template.Must(template.New("resource").
			Funcs(funcs).
			Parse(tmpl),
		)
	}
}

func (c *Controller) Get(_ context.Context, req *controllers.GetRequest) (_ *core.Resource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("get: %w", err)
		}
	}()

	buf := &bytes.Buffer{}
	if err := c.resourceTmpl.Execute(buf, req); err != nil {
		return nil, err
	}

	fi, err := req.FS.Open(buf.String())
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	return c.encoding.NewDecoder(fi).Decode()
}

// List finds all the resources on the provided FS in the folder { namespace }
// The result set is filtered by any specified labels.
func (c *Controller) List(_ context.Context, req *controllers.ListRequest) (resources []*core.Resource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list: %w", err)
		}
	}()

	buf := &bytes.Buffer{}
	if err := c.listTmpl.Execute(buf, req); err != nil {
		return nil, err
	}

	matches, err := fs.Glob(req.FS, buf.String())
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		fi, err := req.FS.Open(match)
		if err != nil {
			return nil, err
		}

		if err := func() error {
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
		}(); err != nil {
			return nil, err
		}
	}

	return
}

// Put for now is a silent noop as we dont have a writable filesystem abstraction
func (c *Controller) Put(_ context.Context, req *controllers.PutRequest) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("put: %w", err)
		}
	}()

	buf := &bytes.Buffer{}
	if err := c.resourceTmpl.Execute(buf, req); err != nil {
		return fmt.Errorf("put: %w", err)
	}

	fi, err := req.FSConfig.ToFS().OpenFile(buf.String(), os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer fi.Close()

	return c.encoding.NewEncoder(fi).Encode(req.Resource)
}

// Delete for now is a silent noop as we dont have a writable filesystem abstraction
func (c *Controller) Delete(_ context.Context, req *controllers.DeleteRequest) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("delete: %w", err)
		}
	}()

	buf := &bytes.Buffer{}
	if err := c.resourceTmpl.Execute(buf, req); err != nil {
		return err
	}

	slog.Debug("Removing file", "path", buf.String())

	return req.FSConfig.ToFS().Remove(buf.String())
}
