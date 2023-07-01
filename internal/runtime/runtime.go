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
	"go.flipt.io/fidgit"
	"golang.org/x/exp/slog"
)

type Factory struct {
	logger *slog.Logger

	runtime wazero.Runtime

	wasm []byte
	typ  fidgit.Type
}

func NewFactory(ctx context.Context, path string) (_ *Factory, err error) {
	factory := &Factory{runtime: wazero.NewRuntime(ctx)}

	wasi_snapshot_preview1.MustInstantiate(ctx, factory.runtime)

	factory.wasm, err = os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := factory.invoke(ctx, buf, "type"); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(buf).Decode(&factory.typ); err != nil {
		return nil, err
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
	entries []*fidgit.Entry
	index   map[fidgit.ID]*fidgit.Entry
}

type Collection struct {
	*Factory
	fs     fs.FS
	logger *slog.Logger
	index  map[fidgit.Namespace]*namespace
}

func (f *Factory) invoke(ctx context.Context, dst io.Writer, command string, opts ...func(wazero.ModuleConfig) wazero.ModuleConfig) error {
	config := wazero.NewModuleConfig().
		WithStdout(dst).
		WithStderr(os.Stderr)

	for _, opt := range opts {
		config = opt(config)
	}

	// InstantiateModule runs the "_start" function, WASI's "main".
	// * Set the program name (arg[0]) to "wasi"; arg[1] should be "/test.txt".
	_, err := f.runtime.InstantiateWithConfig(ctx, f.wasm, config.WithArgs("wasi", command))
	if err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			return exitErr
		} else if !ok {
			return err
		}
	}

	return nil
}

func (f *Factory) Build() (fidgit.Type, fidgit.FactoryFunc) {
	return f.typ, func(ctx context.Context, dir fs.FS) (fidgit.Collection, error) {
		var (
			buf = &bytes.Buffer{}
			dec = json.NewDecoder(buf)
		)

		if err := f.invoke(ctx, buf, "list", func(mc wazero.ModuleConfig) wazero.ModuleConfig {
			return mc.WithFS(dir)
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
				index:   map[fidgit.Namespace]*namespace{},
				logger:  f.logger,
			}
		)

		var err error
		for err == nil {
			var entry fidgit.Entry
			if err = dec.Decode(&entry); err != nil {
				break
			}

			contents, ok := collection.index[entry.Namespace]
			if !ok {
				contents = &namespace{
					index: map[fidgit.ID]*fidgit.Entry{},
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

func (c *Collection) Get(ctx context.Context, n fidgit.Namespace, id fidgit.ID) (*fidgit.Entry, error) {
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

func (c *Collection) List(ctx context.Context, n fidgit.Namespace) ([]*fidgit.Entry, error) {
	c.logger.Debug("List",
		slog.String("namespace", string(n)))

	if ns, ok := c.index[n]; ok {
		return ns.entries, nil
	}

	return nil, fmt.Errorf("%s: namespace %s: not found", c.typ, n)
}

func (c *Collection) Put(ctx context.Context, entry *fidgit.Entry) ([]fidgit.Change, error) {
	var (
		in, out = &bytes.Buffer{}, &bytes.Buffer{}
		dec     = json.NewDecoder(out)
	)

	if err := json.NewEncoder(in).Encode(entry); err != nil {
		return nil, err
	}

	if err := c.invoke(ctx, out, "put", func(mc wazero.ModuleConfig) wazero.ModuleConfig {
		return mc.
			WithStdin(in).
			WithFS(c.fs)
	}); err != nil {
		return nil, err
	}

	var (
		changes []fidgit.Change
		err     error
	)
	for err == nil {
		var change fidgit.Change
		if err = dec.Decode(&change); err != nil {
			break
		}

		changes = append(changes, change)
	}

	return changes, nil
}

func (c *Collection) Delete(ctx context.Context, n fidgit.Namespace, id fidgit.ID) ([]fidgit.Change, error) {
	panic("not implemented") // TODO: Implement
}
