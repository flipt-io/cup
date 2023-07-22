package core

import (
	"encoding/json"
)

// ResourceDefinition represents a definition of a particular resource Kind and its versions
type ResourceDefinition struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   Metadata               `json:"metadata"`
	Names      Names                  `json:"names"`
	Spec       ResourceDefinitionSpec `json:"spec"`
}

type Names struct {
	Kind     string `json:"kind"`
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
}

type ResourceDefinitionSpec struct {
	Group      string                       `json:"group"`
	Controller ResourceDefinitionController `json:"controller"`
	Versions   map[string]json.RawMessage   `json:"schema,omitempty"`
}

type ResourceDefinitionController struct {
	Path string `json:"path"`
}

// Metadata contains Resource metadata include name, labels and annotations
type Metadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// Resource is the core API resource definition used to communicate
// the various available resources on the wire
type Resource struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   NamespacedMetadata `json:"metadata"`
	Spec       json.RawMessage    `json:"spec"`
}

// NamespacedMetadata contains Resource metadata include namespace, name, labels and annotations
type NamespacedMetadata struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}
