package flipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/gobwas/glob"
	"go.flipt.io/fidgit"
	"go.flipt.io/fidgit/internal/ext"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	indexFile = ".flipt.yml"
	defaultNs = "default"
)

type Flag struct {
	Namespace string
	*ext.Flag
}

func (f *Flag) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Flag)
}

func (f *Flag) UnmarshalJSON(b []byte) error {
	var flag ext.Flag
	if err := json.Unmarshal(b, &flag); err != nil {
		return err
	}

	f.Flag = &flag

	return nil
}

func (f Flag) GetNamespace() fidgit.Namespace {
	return fidgit.Namespace(f.Namespace)
}

func (f Flag) GetID() fidgit.ID {
	return fidgit.ID(f.Key)
}

func (f Flag) GetTags() []fidgit.Tag {
	return nil
}

func (f *Flag) GetInternalContext() map[string]string {
	return nil
}

var _ (fidgit.RuntimeFactory[Flag]) = (*FlagCollectionFactory)(nil)

type FlagCollectionFactory struct{}

func (f *FlagCollectionFactory) GetType() fidgit.Type {
	return fidgit.Type{
		Group:   "flipt.io",
		Kind:    "Flag",
		Version: "v1alpha1",
	}
}

func (f *FlagCollectionFactory) GetTagKeys() []string {
	return nil
}

func (f *FlagCollectionFactory) CollectionFor(_ context.Context, source fs.FS) (fidgit.Runtime[Flag], error) {
	paths, err := listStateFiles(source)
	if err != nil {
		return nil, err
	}

	slog.Debug("Opening state files", "paths", paths)

	collection := &FlagCollection{
		docs: map[fidgit.Namespace]*document{},
	}

	for _, p := range paths {
		fi, err := source.Open(p)
		if err != nil {
			return nil, err
		}

		defer fi.Close()

		doc := new(ext.Document)
		if err := yaml.NewDecoder(fi).Decode(doc); err != nil {
			return nil, err
		}

		namespace := "default"
		if doc.Namespace != "" {
			namespace = doc.Namespace
		}

		collection.docs[fidgit.Namespace(namespace)] = &document{
			Document: *doc,
			path:     p,
		}

		for _, flag := range doc.Flags {
			collection.flags = append(collection.flags, &Flag{
				Namespace: namespace,
				Flag:      flag,
			})
		}
	}

	return collection, nil
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

type document struct {
	ext.Document
	path string
}

type FlagCollection struct {
	docs  map[fidgit.Namespace]*document
	flags []*Flag
}

func (f *FlagCollection) ListAll(_ context.Context) ([]*Flag, error) {
	return f.flags, nil
}

func (f *FlagCollection) Put(_ context.Context, ns fidgit.Namespace, flag *Flag) ([]fidgit.File, error) {
	flag.Namespace = string(ns)

	doc, err := f.getDocument(ns)
	if err != nil {
		return nil, err
	}

	var (
		flags = make([]*ext.Flag, 0, len(doc.Flags))
		found bool
	)

	// copy flags as we're mutating the slice
	// to remove the entry before marshalling
	for _, ef := range doc.Flags {
		if ef.Key == flag.Key {
			flags = append(flags, flag.Flag)
			found = true
			continue
		}

		flags = append(flags, ef)
	}

	if !found {
		flags = append(flags, flag.Flag)
		slices.SortFunc(flags, func(i, j *ext.Flag) bool {
			return i.Key < j.Key
		})
	}

	return updateDocument(doc.Document, doc.path, flags)
}

func (f *FlagCollection) Delete(_ context.Context, ns fidgit.Namespace, id fidgit.ID) ([]fidgit.File, error) {
	doc, err := f.getDocument(ns)
	if err != nil {
		return nil, err
	}

	var (
		flags = make([]*ext.Flag, 0, len(doc.Flags))
		found bool
	)

	// copy flags as we're mutating the slice
	// to remove the entry before marshalling
	for _, ef := range doc.Flags {
		if ef.Key == string(id) {
			found = true
			continue
		}

		flags = append(flags, ef)
	}

	if !found {
		return nil, fmt.Errorf("flag %s/%s: not found", ns, id)
	}

	return updateDocument(doc.Document, doc.path, flags)
}

func updateDocument(doc ext.Document, path string, flags []*ext.Flag) ([]fidgit.File, error) {
	doc.Flags = flags

	buf := &bytes.Buffer{}
	if err := yaml.NewEncoder(buf).Encode(&doc); err != nil {
		return nil, err
	}

	return []fidgit.File{
		{Path: path, Contents: buf.Bytes()},
	}, nil
}

func (f *FlagCollection) getDocument(ns fidgit.Namespace) (*document, error) {
	doc, ok := f.docs[ns]
	if !ok {
		return nil, fmt.Errorf("namespace %q: not found", ns)
	}

	return doc, nil
}
