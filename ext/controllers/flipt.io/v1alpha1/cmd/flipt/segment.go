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

type segmentController struct{}

func (c *segmentController) Get(ctx context.Context, namespace, name string, enc encoding.TypedEncoder[Segment]) error {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("Get: %v", err))
		}
	}()

	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		for _, s := range document.Segments {
			if s.Key == name {
				if err := enc.Encode(&Segment{
					Namespace: namespace,
					segment:   s,
				}); err != nil {
					return err
				}

				return nil
			}
		}

		return fmt.Errorf("segment %s/%s: %w", namespace, name, ErrNotFound)
	})
}

func (c *segmentController) List(ctx context.Context, namespace string, enc encoding.TypedEncoder[Segment]) error {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("List: %v", err))
		}
	}()

	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		for _, s := range document.Segments {
			if err := enc.Encode(&Segment{
				Namespace: namespace,
				segment:   s,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *segmentController) Put(ctx context.Context, namespace, name string, segment *Segment) error {
	dir := os.DirFS(".")
	return walkDocuments(dir, func(path string, document *ext.Document) error {
		if document.Namespace != string(segment.Namespace) {
			return nil
		}

		var found bool
		for i, s := range document.Segments {
			if found = s.Key == string(segment.segment.Key); found {
				document.Segments[i] = segment.segment
				break
			}
		}

		if !found {
			document.Segments = append(document.Segments, segment.segment)
		}

		fi, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		defer fi.Close()

		return yaml.NewEncoder(fi).Encode(document)
	})
}

func (c *segmentController) Delete(ctx context.Context, namespace, name string) error {
	return walkDocuments(os.DirFS("."), func(path string, document *ext.Document) error {
		if document.Namespace != namespace {
			return nil
		}

		var found bool
		for i, s := range document.Segments {
			if s.Key != name {
				continue
			}

			document.Segments = append(document.Segments[:i], document.Segments[i+1:]...)

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

type Segment struct {
	Namespace string
	segment   *ext.Segment
}

func (s *Segment) UnmarshalJSON(v []byte) error {
	var resource core.Resource
	if err := json.Unmarshal(v, &resource); err != nil {
		return err
	}

	if err := json.Unmarshal(resource.Spec, &s.segment); err != nil {
		return err
	}

	s.Namespace = resource.Metadata.Namespace
	s.segment.Key = resource.Metadata.Name
	s.segment.Name = resource.Metadata.Name

	return nil
}

func (s *Segment) MarshalJSON() (_ []byte, err error) {
	resource := core.Resource{
		APIVersion: "flipt.io/v1alpha1",
		Kind:       "Segment",
		Metadata: core.NamespacedMetadata{
			Namespace: s.Namespace,
			Name:      s.segment.Key,
		},
	}

	segment := *s.segment
	segment.Key = ""  // key will be carried by the resource name
	segment.Name = "" // key will be carried by the resource name

	resource.Spec, err = json.Marshal(&segment)
	if err != nil {
		return nil, fmt.Errorf("marshalling segment: %w", err)
	}

	return json.Marshal(&resource)
}
