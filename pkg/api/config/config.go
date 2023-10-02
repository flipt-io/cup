package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"

	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/config"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controllers/template"
	"go.flipt.io/cup/pkg/controllers/wasm"
)

func New(ctx context.Context, cfg *config.Config) (*api.Configuration, error) {
	c := &api.Configuration{
		Definitions:  containers.MapStore[string, *core.ResourceDefinition]{},
		Controllers:  containers.MapStore[string, api.Controller]{},
		Bindings:     containers.MapStore[string, *core.Binding]{},
		Transformers: containers.MapStore[string, *core.Transformer]{},
	}

	dir := os.DirFS(cfg.API.Resources)
	return c, fs.WalkDir(dir, ".", func(p string, d fs.DirEntry, err error) (e error) {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(p, ".json") {
			slog.Debug("skipping parsing api file", "path", p)
			return nil
		}

		defer func() {
			if e != nil {
				e = fmt.Errorf("parsing api resource %q: %w", p, e)
			}
		}()

		var (
			buf = &bytes.Buffer{}
			r   core.Object[json.RawMessage]
		)

		fi, err := dir.Open(p)
		if err != nil {
			return err
		}

		defer fi.Close()

		if err := json.NewDecoder(io.TeeReader(fi, buf)).Decode(&r); err != nil {
			return fmt.Errorf("parsing resource %w", err)
		}

		if err := r.Validate(); err != nil {
			return err
		}

		slog.Debug("parsing resource", "kind", r.Kind, "name", r.Metadata.Name)

		switch r.Kind {
		case core.ResourceDefinitionKind:
			var def core.ResourceDefinition
			if err := json.NewDecoder(buf).Decode(&def); err != nil {
				return err
			}

			for version := range def.Spec.Versions {
				c.Definitions[path.Join(def.Spec.Group, version, def.Names.Plural)] = &def
			}
		case core.ControllerKind:
			if err := core.DecodeController(
				buf,
				func(tc core.TemplateController) error {
					c.Controllers[tc.Metadata.Name] = template.New(
						template.WithListTemplate(tc.Spec.Spec.ListTemplate),
						template.WithResourceTemplate(tc.Spec.Spec.ResourceTemplate),
					)
					return nil
				},
				func(w core.WASMController) error {
					bytes, err := fs.ReadFile(dir, w.Spec.Spec.Path)
					if err != nil {
						return err
					}

					c.Controllers[w.Metadata.Name] = wasm.New(ctx, bytes)
					return nil
				},
			); err != nil {
				return err
			}
		case core.BindingKind:
			var binding core.Binding
			if err := json.NewDecoder(buf).Decode(&binding); err != nil {
				return fmt.Errorf("parsing binding %q: %w", r.Metadata.Name, err)
			}

			c.Bindings[binding.Metadata.Name] = &binding
		case core.TransformerKind:
			var transformer core.Transformer

			if err := json.NewDecoder(buf).Decode(&transformer); err != nil {
				return fmt.Errorf("parsing transformer: %w", err)
			}

			c.Transformers[transformer.Spec.Kind] = &transformer
		}

		return nil
	})
}
