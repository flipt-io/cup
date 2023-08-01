package wasm

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controllers"
)

//go:embed testdata/*
var testdata embed.FS

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

func Test_Controller_List(t *testing.T) {
	wasm, skip := compileTestController(t)
	if skip {
		return
	}

	ctx := context.Background()
	controller := New(ctx, wasm)

	resources, err := controller.List(ctx, &controllers.ListRequest{
		FS: testdataFS(t),
		Request: controllers.Request{
			Group:     "test.cup.flipt.io",
			Version:   "v1alpha1",
			Kind:      "Resource",
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []*core.Resource{
		{
			APIVersion: "test.cup.flipt.io/v1alpha1",
			Kind:       "Resource",
			Metadata: core.NamespacedMetadata{
				Namespace: "default",
				Name:      "bar",
				Labels: map[string]string{
					"baz": "bar",
				},
				Annotations: map[string]string{},
			},
			Spec: []byte(`{}`),
		},
		{
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
	}, resources)
}

func Test_Controller_Put(t *testing.T) {
	wasm, skip := compileTestController(t)
	if skip {
		return
	}

	ctx := context.Background()
	controller := New(ctx, wasm)

	// copy test data into tmp dir
	dir := testdataFSCopy(t)

	err := controller.Put(ctx, &controllers.PutRequest{
		FSConfig: controllers.NewDirFSConfig(dir),
		Request: controllers.Request{
			Group:     "test.cup.flipt.io",
			Version:   "v1alpha1",
			Kind:      "Resource",
			Namespace: "default",
		},
		Resource: &core.Resource{
			APIVersion: "test.cup.flipt.io/v1alpha1",
			Kind:       "Resource",
			Metadata: core.NamespacedMetadata{
				Namespace: "default",
				Name:      "baz",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{},
			},
			Spec: []byte(`{}`),
		},
	})
	require.NoError(t, err)

	var resource core.Resource
	fi, err := os.Open(path.Join(dir, "test.cup.flipt.io-v1alpha1-Resource-default-baz.json"))
	require.NoError(t, err)

	err = json.NewDecoder(fi).Decode(&resource)
	require.NoError(t, err)

	assert.Equal(t, &core.Resource{
		APIVersion: "test.cup.flipt.io/v1alpha1",
		Kind:       "Resource",
		Metadata: core.NamespacedMetadata{
			Namespace: "default",
			Name:      "baz",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{},
		},
		Spec: []byte(`{}`),
	}, &resource)
}

func Test_Controller_Delete(t *testing.T) {
	wasm, skip := compileTestController(t)
	if skip {
		return
	}

	ctx := context.Background()
	controller := New(ctx, wasm)

	// copy test data into tmp dir
	dir := testdataFSCopy(t)

	err := controller.Delete(ctx, &controllers.DeleteRequest{
		FSConfig: controllers.NewDirFSConfig(dir),
		Request: controllers.Request{
			Group:     "test.cup.flipt.io",
			Version:   "v1alpha1",
			Kind:      "Resource",
			Namespace: "default",
		},
		Name: "foo",
	})
	require.NoError(t, err)

	_, err = os.Open(path.Join(dir, "test.cup.flipt.io-v1alpha1-Resource-default-foo.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func testdataFS(t *testing.T) fs.FS {
	fs, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	return fs
}

func testdataFSCopy(t *testing.T) string {
	testdata := testdataFS(t)
	dir := t.TempDir()
	fs.WalkDir(testdata, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fi, err := testdata.Open(p)
		if err != nil {
			return err
		}

		defer fi.Close()

		dst, err := os.Create(path.Join(dir, p))
		if err != nil {
			return err
		}

		defer dst.Close()

		_, err = io.Copy(dst, fi)
		return err
	})

	return dir
}

func compileTestController(t *testing.T) ([]byte, bool) {
	t.Helper()

	goCommand := "go"

	cmd := exec.Command(goCommand, "tool", "dist", "list")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)

	if !bytes.Contains(out, []byte(`wasip1/wasm`)) {
		goCommand = "gotip"

		if _, err := exec.LookPath("gotip"); err != nil {
			t.Skip("go support for wasip1 required to run tests")
			return nil, true
		}
	}

	tmp, err := os.CreateTemp("", "test-*.wasm")
	require.NoError(t, err)
	tmp.Close()

	t.Cleanup(func() {
		_ = os.Remove(tmp.Name())
	})

	cmd = exec.Command("sh", "-c", fmt.Sprintf("%s build -o %[1]s testdata/main.go && cat %[1]s", goCommand, tmp.Name()))
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
