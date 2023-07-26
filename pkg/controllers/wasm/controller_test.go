package wasm

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controllers"
)

//go:embed testdata/*
var testdata embed.FS

func testdataFS(t *testing.T) fs.FS {
	fs, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	return fs
}

func Test_Controller_Get(t *testing.T) {
	wasm, skip := compileTestController(t)
	if skip {
		return
	}

	ctx := context.Background()
	controller := New(ctx, wasm)

	resource, err := controller.Get(ctx, &controllers.GetRequest{
		FS: testdataFS(t),
		Request: controllers.Request{
			Group:     "test.cup.flipt.io",
			Version:   "v1alpha1",
			Kind:      "Resource",
			Namespace: "default",
		},
		Name: "foo",
	})
	require.NoError(t, err)

	assert.Equal(t, &core.Resource{
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
	}, resource)
}

func compileTestController(t *testing.T) ([]byte, bool) {
	t.Helper()

	if _, err := exec.LookPath("gotip"); err != nil {
		t.Skip("gotip required to run wazero based tests")
		return nil, true
	}

	tmp, err := os.CreateTemp("", "test-*.wasm")
	require.NoError(t, err)
	tmp.Close()

	t.Cleanup(func() {
		_ = os.Remove(tmp.Name())
	})

	cmd := exec.Command("sh", "-c", fmt.Sprintf("gotip build -o %[1]s testdata/main.go && cat %[1]s", tmp.Name()))
	cmd.Env = append([]string{
		"GOOS=wasip1",
		"GOARCH=wasm",
	}, os.Environ()...)

	data, err := cmd.CombinedOutput()
	if !assert.NoError(t, err) {
		t.Log(string(data))
		t.FailNow()
	}

	return data, false
}
