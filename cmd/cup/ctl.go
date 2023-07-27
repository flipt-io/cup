package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/encoding"
)

func sources(cfg config, client *http.Client) error {
	address := cfg.Address()
	req, err := http.NewRequest(http.MethodGet, address+"/apis", nil)
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

	var sources []api.Source
	if err := json.NewDecoder(resp.Body).Decode(&sources); err != nil {
		return err
	}

	wr := writer()
	fmt.Fprintln(wr, "SOURCE\tRESOURCE COUNT\t")
	for _, src := range sources {
		fmt.Fprintf(wr, "%s\t%d\t\n", src.Name, src.Resources)
	}

	return wr.Flush()
}

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

func list(cfg config, client *http.Client, typ string) error {
	group, version, kind, err := getGVK(cfg, client, typ)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	endpoint := fmt.Sprintf("%s/apis/%s/%s/%s/%s/namespaces/%s",
		cfg.Address(),
		cfg.Source(),
		group,
		version,
		kind,
		cfg.Namespace(),
	)

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

	enc := encoding.NewJSONEncoding[core.Resource]()
	resources, err := encoding.DecodeAll(enc.NewDecoder(resp.Body))
	if err != nil {
		return fmt.Errorf("decoding resources: %w", err)
	}

	wr := writer()
	fmt.Fprintln(wr, "NAMESPACE\tNAME\t")
	for _, resource := range resources {
		fmt.Fprintf(wr, "%s\t%s\t\n", resource.Metadata.Namespace, resource.Metadata.Name)
	}

	return wr.Flush()
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
	req, err := http.NewRequest(http.MethodGet, cfg.Address()+"/apis/"+cfg.Source(), nil)
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
