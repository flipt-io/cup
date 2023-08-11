package template

import (
	"bytes"
	"encoding/json"
	"flag"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/api/core"
)

var resourceFoo = core.NamespacedObject[ResourceSpec]{
	APIVersion: "test.cup.flipt.io/v1alpha1",
	Kind:       "Resource",
	Metadata: core.NamespacedMetadata{
		Name:      "foo",
		Namespace: "default",
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: map[string]string{},
	},
	Spec: ResourceSpec{
		Foo: "bar",
	},
}

var (
	address   = flag.String("cup-address", "http://localhost:8181", "Address of cupd instance")
	namespace = flag.String("cup-namespace", "default", "Namespace context for cup operations")
)

func Test_Cup_Controller_Template(t *testing.T) {
	t.Run("cup definitions", func(t *testing.T) {
		stdout, _, err := cup(t, "-o", "json", "definitions")
		require.NoError(t, err)

		var definition core.ResourceDefinition
		require.NoError(t, json.Unmarshal(stdout, &definition))

		assert.Equal(t, core.ResourceDefinition{
			APIVersion: "cup.flipt.io/v1alpha1",
			Kind:       "ResourceDefinition",
			Metadata: core.Metadata{
				Name: "resources.test.cup.flipt.io",
			},
			Names: core.Names{
				Kind:     "Resource",
				Singular: "resource",
				Plural:   "resources",
			},
			Spec: core.ResourceDefinitionSpec{
				Group: "test.cup.flipt.io",
				Versions: map[string]json.RawMessage{
					"v1alpha1": indent(t, `{"type":"object","properties":{"spec":{"type":"object","properties":{"foo":{"type":"string"}},"additionalProperties":false}},"additionalProperties":false}`, "      "),
				},
			},
		}, definition)
	})

	t.Run("cup get resources", func(t *testing.T) {
		stdout, _, err := cup(t, "-o", "json", "get", "resources")
		require.NoError(t, err)

		var resource core.NamespacedObject[ResourceSpec]
		require.NoError(t, json.Unmarshal(stdout, &resource))

		assert.Equal(t, resourceFoo, resource)
	})

	t.Run("cup get resources foo", func(t *testing.T) {
		stdout, _, err := cup(t, "-o", "json", "get", "resources", "foo")
		require.NoError(t, err)

		var resource core.NamespacedObject[ResourceSpec]
		require.NoError(t, json.Unmarshal(stdout, &resource))

		assert.Equal(t, resourceFoo, resource)
	})
}

type ResourceSpec struct {
	Foo string `json:"foo"`
}

func cup(t *testing.T, args ...string) ([]byte, []byte, error) {
	t.Helper()

	path, err := exec.LookPath("cup")
	require.NoError(t, err, "failed to locate cup binary")

	var (
		stdout = bytes.Buffer{}
		stderr = bytes.Buffer{}
	)

	cmd := exec.Command(path, append([]string{"-a", *address, "-n", *namespace}, args...)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Log("stderr", string(stderr.Bytes()))
		return nil, nil, err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

func indent[B ~string | ~[]byte](t *testing.T, src B, prefix string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, json.Indent(buf, []byte(src), prefix, "  "))
	return buf.Bytes()
}
