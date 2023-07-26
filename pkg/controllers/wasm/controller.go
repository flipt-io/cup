package wasm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controllers"
)

var (
	_ api.Controller = (*Controller)(nil)

	// ErrNotFound is returned when the requested resource cannot
	// be located by the WASM runtime implementation
	ErrNotFound = errors.New("resource not found")
)

type Controller struct {
	runtime wazero.Runtime
	wasm    []byte
}

func New(ctx context.Context, wasm []byte) *Controller {
	c := &Controller{
		runtime: wazero.NewRuntime(ctx),
		wasm:    wasm,
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, c.runtime)

	return c
}

func (c *Controller) Get(ctx context.Context, r *controllers.GetRequest) (_ *core.Resource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wasm.get: %s/%s: %w", r.Request, r.Name, err)
		}
	}()

	buf := &bytes.Buffer{}
	if err := c.exec(ctx,
		[]string{"get", r.Kind, r.Namespace, r.Name},
		func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.WithStdout(buf).WithFSConfig(wazero.NewFSConfig().WithFSMount(r.FS, "/"))
		}); err != nil {
		return nil, err
	}

	var resource core.Resource
	if err := json.NewDecoder(buf).Decode(&resource); err != nil {
		return nil, err
	}

	return &resource, nil
}

func (c *Controller) List(ctx context.Context, r *controllers.ListRequest) (resources []*core.Resource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wasm.list: %s: %w", r.Request, err)
		}
	}()

	buf := &bytes.Buffer{}
	if err = c.exec(ctx,
		[]string{"list", r.Kind, r.Namespace},
		func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.WithStdout(buf).WithFSConfig(wazero.NewFSConfig().WithFSMount(r.FS, "/"))
		}); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(buf)
	for err == nil {
		var resource core.Resource
		if err = dec.Decode(&resource); err != nil {
			break
		}

		resources = append(resources, &resource)
	}

	if err != io.EOF {
		return nil, err
	}

	return resources, nil
}

func (c *Controller) Put(ctx context.Context, r *controllers.PutRequest) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wasm.put: %s/%s: %w", r.Request, r.Name, err)
		}
	}()

	if r.FSConfig.Dir == nil {
		return errors.New("request directory not appropriate")
	}

	in := &bytes.Buffer{}
	if err := json.NewEncoder(in).Encode(r.Resource); err != nil {
		return err
	}

	if err = c.exec(ctx,
		[]string{"put", r.Kind, r.Namespace, r.Name},
		func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.
				WithStdin(in).
				WithFSConfig(wazero.NewFSConfig().WithDirMount(*r.FSConfig.Dir, "/"))
		}); err != nil {
		return err
	}

	return
}

func (c *Controller) Delete(ctx context.Context, r *controllers.DeleteRequest) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("wasm.delete: %s/%s: %w", r.Request, r.Name, err)
		}
	}()

	if r.FSConfig.Dir == nil {
		return errors.New("request directory not appropriate")
	}

	if err = c.exec(ctx,
		[]string{"delete", r.Kind, r.Namespace, r.Name},
		func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.
				WithFSConfig(wazero.NewFSConfig().WithDirMount(*r.FSConfig.Dir, "/"))
		}); err != nil {
		return fmt.Errorf("%s/%s: %w", r.Request, r.Name, err)
	}

	return
}

func (c *Controller) exec(ctx context.Context, args []string, fn func(wazero.ModuleConfig) wazero.ModuleConfig) error {
	config := fn(wazero.NewModuleConfig().
		WithStderr(os.Stderr)).
		WithArgs(append([]string{"wasi"}, args...)...)

	_, err := c.runtime.InstantiateWithConfig(ctx, c.wasm, config)
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			if exitErr.ExitCode() == 2 {
				return fmt.Errorf("exec: %w", ErrNotFound)
			}

			return fmt.Errorf("non-zero exit code: %w", exitErr)
		} else if !ok {
			return fmt.Errorf("%s: %w", args[0], err)
		}
	}

	return nil
}
