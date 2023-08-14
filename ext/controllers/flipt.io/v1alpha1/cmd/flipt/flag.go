package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.flipt.io/cup/ext/controllers/flipt.io/v1alpha1/pkg/ext"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/encoding"
	"gopkg.in/yaml.v2"
)

type flagController struct{}

func (c *flagController) Get(ctx context.Context, namespace, name string, enc encoding.TypedEncoder[Flag]) error {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("Get: %v", err))
		}
	}()

	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		for _, f := range document.Flags {
			if f.Key == name {
				if err := enc.Encode(&Flag{
					Namespace: namespace,
					flag:      f,
				}); err != nil {
					return err
				}

				return nil
			}
		}

		return fmt.Errorf("flag %s/%s: %w", namespace, name, ErrNotFound)
	})
}

func (c *flagController) List(ctx context.Context, namespace string, enc encoding.TypedEncoder[Flag]) error {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("List: %v", err))
		}
	}()

	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		for _, f := range document.Flags {
			if err := enc.Encode(&Flag{
				Namespace: namespace,
				flag:      f,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *flagController) Put(ctx context.Context, namespace, name string, flag *Flag) error {
	dir := os.DirFS(".")
	return walkDocuments(dir, func(path string, document *ext.Document) error {
		if document.Namespace != string(flag.Namespace) {
			return nil
		}

		var found bool
		for i, f := range document.Flags {
			if found = f.Key == string(flag.flag.Key); found {
				document.Flags[i] = flag.flag
				break
			}
		}

		if !found {
			document.Flags = append(document.Flags, flag.flag)
		}

		fi, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		defer fi.Close()

		return yaml.NewEncoder(fi).Encode(document)
	})
}

func (c *flagController) Delete(ctx context.Context, namespace, name string) error {
	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		var found bool
		for i, f := range document.Flags {
			if f.Key != name {
				continue
			}

			document.Flags = append(document.Flags[:i], document.Flags[i+1:]...)

			found = true
		}

		if !found {
			return nil
		}

		fi, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		defer fi.Close()

		return yaml.NewEncoder(fi).Encode(document)
	})
}

type Flag struct {
	Namespace string
	flag      *ext.Flag
}

func (f *Flag) UnmarshalJSON(v []byte) error {
	var resource core.Resource
	if err := json.Unmarshal(v, &resource); err != nil {
		return err
	}

	if err := json.Unmarshal(resource.Spec, &f.flag); err != nil {
		return err
	}

	f.Namespace = resource.Metadata.Namespace
	f.flag.Key = resource.Metadata.Name
	f.flag.Name = resource.Metadata.Name

	return nil
}

func (f *Flag) MarshalJSON() (_ []byte, err error) {
	resource := core.Resource{
		APIVersion: "flipt.io/v1alpha1",
		Kind:       "Flag",
		Metadata: core.NamespacedMetadata{
			Namespace: f.Namespace,
			Name:      f.flag.Key,
		},
	}

	flag := *f.flag
	flag.Key = ""  // key will be carried by the resource name
	flag.Name = "" // key will be carried by the resource name

	resource.Spec, err = json.Marshal(&flag)
	if err != nil {
		return nil, fmt.Errorf("marshalling flag: %w", err)
	}

	return json.Marshal(&resource)
}
