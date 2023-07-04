package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/gobwas/glob"
	"go.flipt.io/cup"
	"go.flipt.io/cup/internal/ext"
	sdk "go.flipt.io/cup/sdk/go"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	indexFile = ".flipt.yml"
	defaultNs = "default"
)

func main() {
	if err := sdk.New(cup.Type{
		Group:   "flipt.io",
		Kind:    "Flag",
		Version: "v1",
	}, sdk.Typed[flag](&runtime{})).
		Run(context.Background(), os.Args...); err != nil {
		panic(err)
	}
}

type runtime struct{}

func (r *runtime) ListAll(ctx context.Context, enc sdk.TypedEncoder[flag]) error {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("ListAll: %v", err))
		}
	}()

	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		for _, f := range document.Flags {
			if err := enc.Encode(flag{
				Namespace: document.Namespace,
				ID:        f.Key,
				Payload: payload{
					Name:        f.Name,
					Description: f.Description,
					Enabled:     f.Enabled,
					Variants:    f.Variants,
					Rules:       f.Rules,
				},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *runtime) Put(ctx context.Context, flag *flag, enc sdk.TypedEncoder[cup.Change]) error {
	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != string(flag.Namespace) {
			return nil
		}

		var found bool
		for _, f := range document.Flags {
			if f.Key != string(flag.ID) {
				continue
			}

			found = true

			f.Description = flag.Payload.Description
			f.Enabled = flag.Payload.Enabled
			f.Variants = flag.Payload.Variants
			f.Rules = flag.Payload.Rules
		}

		action := "update"
		if !found {
			action = "create"
			document.Flags = append(document.Flags, &ext.Flag{
				Key:         flag.ID,
				Name:        flag.Payload.Name,
				Description: flag.Payload.Description,
				Enabled:     flag.Payload.Enabled,
				Variants:    flag.Payload.Variants,
				Rules:       flag.Payload.Rules,
			})
		}

		buf := &bytes.Buffer{}
		if err := yaml.NewEncoder(buf).Encode(document); err != nil {
			return err
		}

		return enc.Encode(cup.Change{
			Message:  fmt.Sprintf("feat: %s flag \"%s/%s\"", action, flag.Namespace, flag.ID),
			Path:     path,
			Contents: buf.Bytes(),
		})
	})
}

func (r *runtime) Delete(ctx context.Context, namespace cup.Namespace, id cup.ID, enc sdk.TypedEncoder[cup.Change]) error {
	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != string(namespace) {
			return nil
		}

		var found bool
		for i, f := range document.Flags {
			if f.Key != string(id) {
				continue
			}

			document.Flags = append(document.Flags[:i], document.Flags[i+1:]...)

			found = true
		}

		if !found {
			return nil
		}

		buf := &bytes.Buffer{}
		if err := yaml.NewEncoder(buf).Encode(document); err != nil {
			return err
		}

		return enc.Encode(cup.Change{
			Message:  fmt.Sprintf("feat: delete flag \"%s/%s\"", namespace, id),
			Path:     path,
			Contents: buf.Bytes(),
		})
	})
}

type flag struct {
	Namespace string  `json:"namespace"`
	ID        string  `json:"id"`
	Payload   payload `json:"payload"`
}

type payload struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
	Variants    []*ext.Variant `json:"variants"`
	Rules       []*ext.Rule    `json:"rules"`
}

var errFinish = errors.New("finish")

func walkDocuments(source fs.FS, fn func(path string, document *ext.Document) error) error {
	paths, err := listStateFiles(source)
	if err != nil {
		return err
	}

	for _, p := range paths {
		fi, err := source.Open(p)
		if err != nil {
			return err
		}

		defer fi.Close()

		doc := new(ext.Document)
		if err := yaml.NewDecoder(fi).Decode(doc); err != nil {
			return err
		}

		if doc.Namespace == "" {
			doc.Namespace = "default"
		}

		if err := fn(p, doc); err != nil {
			if errors.Is(err, errFinish) {
				return nil
			}

			return err
		}
	}

	return nil
}

// FliptIndex represents the structure of a well-known file ".flipt.yml"
// at the root of an FS.
type FliptIndex struct {
	Version string   `yaml:"version,omitempty"`
	Include []string `yaml:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}

func listStateFiles(source fs.FS) ([]string, error) {
	// This is the default variable + value for the FliptIndex. It will preserve its value if
	// a .flipt.yml can not be read for whatever reason.
	idx := FliptIndex{
		Version: "1.0",
		Include: []string{
			"**features.yml", "**features.yaml", "**.features.yml", "**.features.yaml",
		},
	}

	// Read index file
	inFile, err := source.Open(indexFile)
	if err == nil {
		if derr := yaml.NewDecoder(inFile).Decode(&idx); derr != nil {
			return nil, fmt.Errorf("yaml: %w", derr)
		}
	}

	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		} else {
			slog.Debug("index file does not exist, defaulting...", slog.String("file", indexFile), "error", err)
		}
	}

	var includes []glob.Glob
	for _, g := range idx.Include {
		glob, err := glob.Compile(g)
		if err != nil {
			return nil, fmt.Errorf("compiling include glob: %w", err)
		}

		includes = append(includes, glob)
	}

	filenames := make([]string, 0)
	if err := fs.WalkDir(source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		for _, glob := range includes {
			if glob.Match(path) {
				filenames = append(filenames, path)
				return nil
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if len(idx.Exclude) > 0 {
		var excludes []glob.Glob
		for _, g := range idx.Exclude {
			glob, err := glob.Compile(g)
			if err != nil {
				return nil, fmt.Errorf("compiling include glob: %w", err)
			}

			excludes = append(excludes, glob)
		}

	OUTER:
		for i := range filenames {
			for _, glob := range excludes {
				if glob.Match(filenames[i]) {
					filenames = append(filenames[:i], filenames[i+1:]...)
					continue OUTER
				}
			}
		}
	}

	return filenames, nil
}
