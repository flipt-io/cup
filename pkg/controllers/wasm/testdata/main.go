package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go.flipt.io/cup/pkg/api/core"
)

var data = map[string]map[string]map[string]*core.Resource{
	"Resource": map[string]map[string]*core.Resource{
		"default": map[string]*core.Resource{
			"foo": &core.Resource{
				APIVersion: "test.cup.flipt.io/v1alpha1",
				Kind:       "Resource",
				Metadata: core.NamespacedMetadata{
					Namespace: "default",
					Name:      "foo",
					Labels: map[string]string{
						"bar": "baz",
					},
					Annotations: map[string]string{},
				},
				Spec: []byte(`{}`),
			},
		},
	},
}

func main() {
	switch os.Args[1] {
	case "get":
		namespaces, ok := data[os.Args[2]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unexpected kind: %q", os.Args[2])
			os.Exit(1)
		}

		namespace, ok := namespaces[os.Args[3]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown namespace: %s/%s", os.Args[2], os.Args[3])
			os.Exit(2)
		}

		resource, ok := namespace[os.Args[4]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown resource: %s/%s/%s", os.Args[2], os.Args[3], os.Args[4])
			os.Exit(2)
		}

		if err := json.NewEncoder(os.Stdout).Encode(resource); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		return
	default:
		fmt.Fprintf(os.Stderr, "unexpected command %q", os.Args[1])
		os.Exit(1)
	}
}
