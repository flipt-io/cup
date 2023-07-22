package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/api/core"
	"go.flipt.io/cup/pkg/controllers/simple"
	"go.flipt.io/cup/pkg/fs/mem"
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
		Group:      "test.cup.flipt.io",
		Controller: core.ResourceDefinitionController{},
		Versions: map[string]json.RawMessage{
			"v1alpha1": []byte("null"),
		},
	},
}

func Test_Server_Source(t *testing.T) {
	fss := mem.New()
	server, err := api.NewServer(fss)
	require.NoError(t, err)

	cntrl := simple.New(testDef)
	server.RegisterController("cup", cntrl)

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/apis")
	require.NoError(t, err)

	defer resp.Body.Close()

	var sources []string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sources))

	assert.Equal(t, []string{"cup"}, sources)
}

func Test_Server_SourceDefinitions(t *testing.T) {
	fss := mem.New()
	server, err := api.NewServer(fss)
	require.NoError(t, err)

	cntrl := simple.New(testDef)
	server.RegisterController("cup", cntrl)

	srv := httptest.NewServer(server)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/apis/cup")
	require.NoError(t, err)

	defer resp.Body.Close()

	var definitions map[string]*core.ResourceDefinition
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&definitions))

	assert.Equal(t, map[string]*core.ResourceDefinition{
		"test.cup.flipt.io/v1alpha1/Resource": testDef,
	}, definitions)
}
