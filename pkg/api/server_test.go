package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controllers/template"
	"go.flipt.io/cup/pkg/encoding"
	"go.flipt.io/cup/pkg/source/mem"
	"golang.org/x/exp/slog"
)

var testDef = &core.ResourceDefinition{
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
			"v1alpha1": []byte(`{"type":"object"}`),
		},
	},
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)))

	os.Exit(m.Run())
}

func Test_Server_Definitions(t *testing.T) {
	var (
		fss   = mem.New()
		cntrl = template.New()
	)

	server, err := api.NewServer(fss)
	require.NoError(t, err)

	server.Register(cntrl, testDef)

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/apis")
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var definitions map[string]*core.ResourceDefinition
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&definitions))

	assert.Equal(t, map[string]*core.ResourceDefinition{
		"test.cup.flipt.io/v1alpha1/resources": testDef,
	}, definitions)
}

func Test_Server_Get(t *testing.T) {
	fss := mem.New()
	fss.AddFS("main", osfs.New("testdata"))

	cntrl := template.New()

	server, err := api.NewServer(fss)
	require.NoError(t, err)

	require.NoError(t, server.Register(cntrl, testDef))

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	path := "/apis/test.cup.flipt.io/v1alpha1/namespaces/default/resources/foo"
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var resource *core.Resource
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&resource))

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

func Test_Server_List(t *testing.T) {
	fss := mem.New()
	fss.AddFS("main", osfs.New("testdata"))
	cntrl := template.New()

	server, err := api.NewServer(fss)
	require.NoError(t, err)

	require.NoError(t, server.Register(cntrl, testDef))

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	path := "/apis/test.cup.flipt.io/v1alpha1/namespaces/default/resources"
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	decoder := encoding.NewJSONDecoder[core.Resource](resp.Body)

	resources, err := encoding.DecodeAll[core.Resource](decoder)
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

func Test_Server_Put(t *testing.T) {
	fs := memfs.New()
	fss := mem.New()
	fss.AddFS("main", fs)

	cntrl := template.New()

	server, err := api.NewServer(fss)
	require.NoError(t, err)

	require.NoError(t, server.Register(cntrl, testDef))

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	path := "/apis/test.cup.flipt.io/v1alpha1/namespaces/default/resources/baz"
	body := bytes.NewReader([]byte(bazPayload))

	req, err := http.NewRequest("PUT", srv.URL+path, body)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	if !assert.Equal(t, http.StatusOK, resp.StatusCode) {
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		t.Log(string(data))
		t.FailNow()
	}

	var result api.Result
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, &api.Result{}, &result)

	fi, err := fs.Open("default/test.cup.flipt.io-v1alpha1-Resource-baz.json")
	require.NoError(t, err)

	defer fi.Close()

	data, err := io.ReadAll(fi)
	require.NoError(t, err)

	expected := &bytes.Buffer{}
	require.NoError(t, json.Compact(expected, []byte(bazPayload)))
	expected.Write([]byte{'\n'})

	assert.Equal(t, expected.Bytes(), data)
}

func Test_Server_Delete(t *testing.T) {
	var (
		fs      = memfs.New()
		fsPath  = "default/test.cup.flipt.io-v1alpha1-Resource-baz.json"
		fi, err = fs.Create(fsPath)
	)
	require.NoError(t, err)

	io.Copy(fi, strings.NewReader(bazPayload))
	_ = fi.Close()

	fss := mem.New()
	fss.AddFS("main", fs)
	cntrl := template.New()

	server, err := api.NewServer(fss)
	require.NoError(t, err)

	require.NoError(t, server.Register(cntrl, testDef))

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	path := "/apis/test.cup.flipt.io/v1alpha1/namespaces/default/resources/baz"
	req, err := http.NewRequest("DELETE", srv.URL+path, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	if !assert.Equal(t, http.StatusOK, resp.StatusCode) {
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		t.Log(string(data))
		t.FailNow()
	}

	var result api.Result
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, &api.Result{}, &result)

	_, err = fs.Open(fsPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

const bazPayload = `{
    "apiVersion": "test.cup.flipt.io/v1alpha1",
    "kind": "Resource",
    "metadata": {
        "namespace": "default",
        "name": "baz",
        "labels": {
            "foo": "bar"
        },
        "annotations": {}
    },
    "spec": {}
}`
