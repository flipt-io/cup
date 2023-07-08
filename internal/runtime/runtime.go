package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"go.flipt.io/cup"
	"golang.org/x/exp/slog"
)

type Factory struct {
	logger *slog.Logger

	runtime wazero.Runtime

	wasm []byte
	typ  cup.Type
}

func NewFactory(ctx context.Context, path string) (_ *Factory, err error) {
	factory := &Factory{runtime: wazero.NewRuntime(ctx)}

	wasi_snapshot_preview1.MustInstantiate(ctx, factory.runtime)

	factory.wasm, err = os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := factory.invoke(ctx, buf, []string{"type"}); err != nil {
		return nil, fmt.Errorf("getting type: %w", err)
	}

	if err := json.NewDecoder(buf).Decode(&factory.typ); err != nil {
		return nil, fmt.Errorf("decoding type: %w", err)
	}

	factory.logger = slog.With(
		slog.String("system", "collection"),
		slog.String("group", factory.typ.Group),
		slog.String("kind", factory.typ.Kind),
		slog.String("version", factory.typ.Version),
	)

	return factory, nil
}

type namespace struct {
	entries []*cup.Entry
	index   map[cup.ID]*cup.Entry
}

type Collection struct {
	*Factory
	fs     fs.FS
	logger *slog.Logger
	index  map[cup.Namespace]*namespace
}

func (f *Factory) Build() (cup.Type, cup.FactoryFunc) {
	return f.typ, func(ctx context.Context, dir fs.FS) (cup.Collection, error) {
		var (
			buf = &bytes.Buffer{}
			dec = json.NewDecoder(buf)
		)

		if err := f.invoke(ctx, buf, []string{"list"}, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.WithFSConfig(wazero.NewFSConfig().WithFSMount(dir, "/"))
		}); err != nil {
			return nil, err
		}

		type item struct {
			Namespace string `json:"namespace"`
			ID        string `json:"id"`
		}

		var (
			collection = &Collection{
				Factory: f,
				fs:      dir,
				index:   map[cup.Namespace]*namespace{},
				logger:  f.logger,
			}
		)

		var err error
		for err == nil {
			var entry cup.Entry
			if err = dec.Decode(&entry); err != nil {
				break
			}

			contents, ok := collection.index[entry.Namespace]
			if !ok {
				contents = &namespace{
					index: map[cup.ID]*cup.Entry{},
				}
				collection.index[entry.Namespace] = contents
			}

			contents.entries = append(contents.entries, &entry)
			contents.index[entry.ID] = &entry
		}

		if err != io.EOF {
			return nil, err
		}

		return collection, nil
	}
}

func (c *Collection) Get(ctx context.Context, n cup.Namespace, id cup.ID) (*cup.Entry, error) {
	c.logger.Debug("Get",
		slog.String("namespace", string(n)),
		slog.String("id", string(id)))

	if ns, ok := c.index[n]; ok {
		if entry, ok := ns.index[id]; ok {
			return entry, nil
		}
	}

	return nil, fmt.Errorf("%s: item %s/%s: not found", c.typ, n, id)
}

func (c *Collection) List(ctx context.Context, n cup.Namespace) ([]*cup.Entry, error) {
	c.logger.Debug("List",
		slog.String("namespace", string(n)))

	if ns, ok := c.index[n]; ok {
		return ns.entries, nil
	}

	return nil, fmt.Errorf("%s: namespace %s: not found", c.typ, n)
}

func (c *Collection) Put(ctx context.Context, entry *cup.Entry) ([]cup.Change, error) {
	in := &bytes.Buffer{}

	if err := json.NewEncoder(in).Encode(entry); err != nil {
		return nil, err
	}

	return c.mutate(ctx, []string{"put"}, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
		return mc.WithStdin(in)
	})
}

func (c *Collection) Delete(ctx context.Context, n cup.Namespace, id cup.ID) ([]cup.Change, error) {
	return c.mutate(ctx, []string{"delete", string(n), string(id)})
}

func (c *Collection) mutate(ctx context.Context, args []string, opts ...func(wazero.ModuleConfig) wazero.ModuleConfig) ([]cup.Change, error) {
	var (
		out = &bytes.Buffer{}
		dec = json.NewDecoder(out)
	)

	if err := c.invoke(ctx, out, args, append(opts, func(mc wazero.ModuleConfig) wazero.ModuleConfig {
		return mc.
			WithFSConfig(wazero.NewFSConfig().WithFSMount(c.fs, "/"))
	})...); err != nil {
		return nil, err
	}

	var (
		changes []cup.Change
		err     error
	)
	for err == nil {
		var change cup.Change
		if err = dec.Decode(&change); err != nil {
			break
		}

		changes = append(changes, change)
	}

	return changes, nil
}

func (f *Factory) invoke(ctx context.Context, dst io.Writer, args []string, opts ...func(wazero.ModuleConfig) wazero.ModuleConfig) error {
	config := wazero.NewModuleConfig().
		WithStdout(dst).
		WithStderr(os.Stderr)

	for _, opt := range opts {
		config = opt(config)
	}

	_, err := f.runtime.InstantiateWithConfig(ctx, f.wasm, config.WithArgs(append([]string{"wasi"}, args...)...))
	if err != nil {
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			return exitErr
		} else if !ok {
			return err
		}
	}

	return nil
}
