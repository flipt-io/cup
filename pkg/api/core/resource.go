package core

import (
	"encoding/json"
)

const (
	ResourceDefinitionKind = "ResourceDefinition"
	BindingKind            = "Binding"
)

// Resource is the core API resource definition used to communicate
// the various available resources on the wire
type Resource NamespacedObject[json.RawMessage]

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
	Group    string                     `json:"group"`
	Versions map[string]json.RawMessage `json:"versions,omitempty"`
}

type Binding Object[BindingSpec]

type BindingSpec struct {
	Resources  []string
	Controller string
}
