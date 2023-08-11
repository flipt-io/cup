package core

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	ControllerKind = "Controller"

	ControllerSpecTypeTemplate = ControllerSpecType("template")
	ControllerSpecTypeWASM     = ControllerSpecType("wasm")
)

type ControllerSpecType string

type Controller[T any] Object[ControllerSpec[T]]

type TemplateController Controller[TemplateControllerSpec]

type WASMController Controller[WASMControllerSpec]

func DecodeController(
	r io.Reader,
	template func(TemplateController) error,
	wasm func(WASMController) error,
) error {
	var c Controller[json.RawMessage]
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return err
	}

	switch c.Spec.Type {
	case ControllerSpecTypeTemplate:
		cntrl := TemplateController{
			APIVersion: c.APIVersion,
			Kind:       c.Kind,
			Metadata:   c.Metadata,
			Spec: ControllerSpec[TemplateControllerSpec]{
				Type: c.Spec.Type,
			},
		}
		if len(c.Spec.Spec) > 0 {
			if err := json.Unmarshal([]byte(c.Spec.Spec), &cntrl.Spec.Spec); err != nil {
				return fmt.Errorf("parsing template spec: %w", err)
			}
		}

		return template(cntrl)
	case ControllerSpecTypeWASM:
		cntrl := WASMController{
			APIVersion: c.APIVersion,
			Kind:       c.Kind,
			Metadata:   c.Metadata,
			Spec: ControllerSpec[WASMControllerSpec]{
				Type: c.Spec.Type,
			},
		}

		if err := json.Unmarshal([]byte(c.Spec.Spec), &cntrl.Spec.Spec); err != nil {
			return err
		}

		return wasm(cntrl)
	default:
		return fmt.Errorf("unexpected controller type: %q", c.Spec.Type)
	}
}

type ControllerSpec[T any] struct {
	Type ControllerSpecType `json:"type"`
	Spec T                  `json:"spec"`
}

type TemplateControllerSpec struct {
	DirectoryTemplate string `json:"directory_template"`
	PathTemplate      string `json:"path_template"`
}

type WASMControllerSpec struct {
	Path string `json:"path"`
}
