package core

import (
	"errors"
	"fmt"
)

const APIVersion = "cup.flipt.io/v1alpha1"

type Object[T any] struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       T        `json:"spec"`
}

// Metadata contains Resource metadata include name, labels and annotations
type Metadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func (n Object[T]) Validate() error {
	if n.APIVersion != APIVersion {
		return fmt.Errorf("unexpected APIVersion: %q", n.APIVersion)
	}

	if n.Metadata.Name == "" {
		return errors.New("name cannot be empty")
	}

	return nil
}

type NamespacedObject[T any] struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   NamespacedMetadata `json:"metadata"`
	Spec       T                  `json:"spec"`
}

// NamespacedMetadata contains Resource metadata include namespace, name, labels and annotations
type NamespacedMetadata struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func (n NamespacedObject[T]) Validate() error {
	if n.APIVersion != APIVersion {
		return fmt.Errorf("unexpected APIVersion: %q", n.APIVersion)
	}

	if n.Metadata.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if n.Metadata.Name == "" {
		return errors.New("name cannot be empty")
	}

	return nil
}
