package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"text/tabwriter"

	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/encoding"
)

func definitions(cfg config, client *http.Client) error {
	definitions, err := getDefintions(cfg, client)
	if err != nil {
		return err
	}

	var names []string
	for name := range definitions {
		names = append(names, name)
	}

	sort.Strings(names)

	wr := writer()
	fmt.Fprintln(wr, "NAME\tAPIVERSION\tKIND\t")
	for _, name := range names {
		def := definitions[name]
		for version := range def.Spec.Versions {
			fmt.Fprintf(wr, "%s\t%s/%s\t%s\t\n", def.Names.Plural, def.Spec.Group, version, def.Names.Kind)
		}
	}

	return wr.Flush()
}

func get(cfg config, client *http.Client, typ string, args ...string) error {
	group, version, kind, err := getGVK(cfg, client, typ)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	endpoint := fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s",
		cfg.Address(),
		group,
		version,
		cfg.Namespace(),
		kind,
	)

	if len(args) == 1 {
		// get with a single argument will use get instead of list
		endpoint += "/" + args[0]
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %q", resp.Status)
	}

	dec := encoding.NewJSONDecoder[core.Resource](resp.Body)
	resources, err := encoding.DecodeAll[core.Resource](dec)
	if err != nil {
		return fmt.Errorf("decoding resources: %w", err)
	}

	var out encoding.TypedEncoder[core.Resource]
	switch cfg.Output {
	case "table":
		table := newTableEncoding(
			func(r *core.Resource) []string {
				return []string{r.Metadata.Namespace, r.Metadata.Name}
			},
			"NAMESPACE", "NAME",
		)

		out = table

		defer table.Flush()
	case "json":
		enc := encoding.NewJSONEncoder[core.Resource](os.Stdout)
		enc.SetIndent("", "  ")
		out = enc
	default:
		return fmt.Errorf("unexpected output type: %q", cfg.Output)
	}

	for _, resource := range resources {
		// filter by name client side
		if len(args) > 0 {
			var found bool
			for _, name := range args {
				if found = resource.Metadata.Name == name; found {
					break
				}
			}

			if !found {
				continue
			}
		}

		if err := out.Encode(resource); err != nil {
			return err
		}
	}

	return nil
}

func apply(cfg config, client *http.Client, source string) (err error) {
	var (
		buf = &bytes.Buffer{}
		rd  = os.Stdin
	)

	if source != "-" {
		rd, err = os.Open(source)
		if err != nil {
			return
		}
	}

	resource, err := encoding.
		NewJSONDecoder[core.Resource](io.TeeReader(rd, buf)).
		Decode()
	if err != nil {
		return err
	}

	defs, err := getDefintions(cfg, client)
	if err != nil {
		return err
	}

	gvk := path.Join(resource.APIVersion, resource.Kind)
	def, ok := defs[gvk]
	if !ok {
		return fmt.Errorf("unexpected resource kind: %q", gvk)
	}

	group, version, _ := strings.Cut(resource.APIVersion, "/")
	endpoint := fmt.Sprintf("%s/apis/%s/%s/namespaces/%s/%s/%s",
		cfg.Address(),
		group,
		version,
		resource.Metadata.Namespace,
		def.Names.Plural,
		resource.Metadata.Name,
	)

	req, err := http.NewRequest(http.MethodPut, endpoint, buf)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %q", resp.Status)
	}

	return nil
}

type tableEncoding[T any] struct {
	*tabwriter.Writer

	headers       []string
	rowFn         func(*T) []string
	headerPrinted bool
}

func newTableEncoding[T any](rowFn func(*T) []string, headers ...string) *tableEncoding[T] {
	return &tableEncoding[T]{Writer: writer(), rowFn: rowFn, headers: headers}
}

func (e *tableEncoding[T]) Encode(t *T) error {
	if !e.headerPrinted {
		fmt.Fprintln(e, strings.Join(e.headers, "\t")+"\t")
		e.headerPrinted = true
	}

	_, err := fmt.Fprintln(e, strings.Join(e.rowFn(t), "\t")+"\t")

	return err
}

func getGVK(cfg config, client *http.Client, typ string) (group, version, kind string, err error) {
	parts := strings.SplitN(typ, "/", 3)
	switch len(parts) {
	case 3:
		group, version, kind = parts[0], parts[1], parts[2]
	case 2, 1:
		kind = parts[0]
		if len(parts) > 1 {
			group, kind = parts[0], parts[1]
		}

		defs, err := getDefintions(cfg, client)
		if err != nil {
			return group, version, kind, err
		}

		var found bool
		for _, def := range defs {
			if group == "" || def.Spec.Group == group {
				if found = (def.Names.Kind == kind || def.Names.Plural == kind); found {
					// TODO(georgemac): we need a property for current returned version
					for version = range def.Spec.Versions {
					}

					if def.Names.Kind == kind {
						kind = def.Names.Plural
					}

					if group == "" {
						group = def.Spec.Group
					}

					break
				}
			}
		}

		if !found {
			return group, version, kind, fmt.Errorf("unknown resource kind: %q", typ)
		}
	}

	return
}

func getDefintions(cfg config, client *http.Client) (map[string]*core.ResourceDefinition, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.Address()+"/apis", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %q", resp.Status)
	}

	definitions := map[string]*core.ResourceDefinition{}
	if err := json.NewDecoder(resp.Body).Decode(&definitions); err != nil {
		return nil, err
	}

	return definitions, nil
}

func writer() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
}
