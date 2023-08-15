package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"go.flipt.io/cup/ext/controllers/flipt.io/v1alpha1/pkg/ext"

	"github.com/gobwas/glob"
	sdk "go.flipt.io/cup/sdk/controller/go"
	"gopkg.in/yaml.v3"
)

var (
	ErrNotFound = errors.New("not found")
)

const (
	indexFile = ".flipt.yml"
	defaultNs = "default"
)

func main() {
	cli := sdk.NewCLI()
	cli.RegisterKind("Flag", sdk.NewKindController[Flag](&flagController{}))
	cli.RegisterKind("Segment", sdk.NewKindController[Segment](&segmentController{}))
	cli.Run(context.Background(), os.Args...)
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
