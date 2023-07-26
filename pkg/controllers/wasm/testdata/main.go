package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"

	"go.flipt.io/cup/pkg/api/core"
)

var data = map[string]map[string]map[string]*core.Resource{}

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func addResource(def *core.Resource) {
	namespaces, ok := data[def.Kind]
	if !ok {
		namespaces = map[string]map[string]*core.Resource{}
		data[def.Kind] = namespaces
	}

	namespace, ok := namespaces[def.Metadata.Namespace]
	if !ok {
		namespace = map[string]*core.Resource{}
		namespaces[def.Metadata.Namespace] = namespace
	}

	namespace[def.Metadata.Name] = def
}

func main() {
	dfs := os.DirFS(".")
	matches, err := fs.Glob(dfs, "*.json")
	fatal(err)

	for _, path := range matches {
		fi, err := dfs.Open(path)
		fatal(err)

		var resource core.Resource

		fatal(json.NewDecoder(fi).Decode(&resource))

		addResource(&resource)

		_ = fi.Close()
	}

	switch os.Args[1] {
	case "get":
		namespace, err := getNamespace(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(2)
		}

		resource, ok := namespace[os.Args[4]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown resource: %s/%s/%s", os.Args[2], os.Args[3], os.Args[4])
			os.Exit(2)
		}

		err = json.NewEncoder(os.Stdout).Encode(resource)
		fatal(err)

		return
	case "list":
		namespace, err := getNamespace(os.Args[2], os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(2)
		}

		var names []string
		for name := range namespace {
			names = append(names, name)
		}

		sort.Strings(names)

		enc := json.NewEncoder(os.Stdout)
		for _, name := range names {
			enc.Encode(namespace[name])
		}
	case "put":
		var resource core.Resource
		err := json.NewDecoder(os.Stdin).Decode(&resource)
		fatal(err)

		group, version := path.Split(resource.APIVersion)

		fi, err := os.Create(fmt.Sprintf("%s-%s-%s-%s-%s.json", group[:len(group)-1], version, resource.Kind, resource.Metadata.Namespace, resource.Metadata.Name))
		fatal(err)
		defer fi.Close()

		err = json.NewEncoder(fi).Encode(&resource)
		fatal(err)
	case "delete":
		err := os.Remove(fmt.Sprintf("test.cup.flipt.io-v1alpha1-%s-%s-%s.json", os.Args[2], os.Args[3], os.Args[4]))
		fatal(err)
	default:
		fmt.Fprintf(os.Stderr, "unexpected command %q", os.Args[1])
		os.Exit(1)
	}
}

func getNamespace(kind, namespace string) (map[string]*core.Resource, error) {
	namespaces, ok := data[kind]
	if !ok {
		return nil, fmt.Errorf("unexpected kind: %q", kind)
	}

	ns, ok := namespaces[namespace]
	if !ok {
		return nil, fmt.Errorf("unknown namespace: %s/%s", kind, namespace)
	}

	return ns, nil
}
