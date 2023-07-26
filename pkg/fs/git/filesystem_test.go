package git_test

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.flipt.io/cup/pkg/containers"
	"go.flipt.io/cup/pkg/controllers"
	"go.flipt.io/cup/pkg/fs/git"
	giteascm "go.flipt.io/cup/pkg/fs/git/scm/gitea"
	"golang.org/x/exp/slog"
)

var gitRepoURL = os.Getenv("TEST_GIT_REPO_URL")

func Test_Filesystem_View(t *testing.T) {
	ctx := context.Background()
	fss, _, skipped := testFilesystem(t, ctx)
	if skipped {
		return
	}

	files := map[string][]byte{}
	require.NoError(t, fss.View(ctx, "main", func(f fs.FS) error {
		fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			fi, err := f.Open(path)
			if err != nil {
				return err
			}

			defer fi.Close()

			data, err := io.ReadAll(fi)
			if err != nil {
				return err
			}

			files[path] = data
			return nil
		})

		return nil
	}))

	assert.Equal(t, testdataContents, files)
}

func Test_Filesystem_Update(t *testing.T) {
	ctx := context.Background()

	fss, scm, skipped := testFilesystem(t, ctx)
	if skipped {
		return
	}

	result, err := fss.Update(ctx, "main", "feat: add `baz` resource", func(f controllers.FSConfig) error {
		fi, err := f.ToFS().Create(bazPath)
		if err != nil {
			return err
		}

		defer fi.Close()

		if _, err := fi.Write(bazContents); err != nil {
			return err
		}

		return nil
	})
	require.NoError(t, err)

	require.NoError(t, scm.Merge(ctx, result.ID))

	// attempt to block until the Filesystem gets an update
	git.WaitForUpdate(t, fss, 30*time.Second)

	files := map[string][]byte{}
	require.NoError(t, fss.View(ctx, "main", func(f fs.FS) error {
		fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			fi, err := f.Open(path)
			if err != nil {
				return err
			}

			defer fi.Close()

			data, err := io.ReadAll(fi)
			if err != nil {
				return err
			}

			files[path] = data
			return nil
		})

		return nil
	}))

	assert.Equal(t, testdata([2]string{
		bazPath, string(bazContents),
	}), files)
}

type scm interface {
	git.SCM
	Merge(context.Context, ulid.ULID) error
}

func testFilesystem(t *testing.T, ctx context.Context, opts ...containers.Option[git.Filesystem]) (*git.Filesystem, scm, bool) {
	t.Helper()

	if gitRepoURL == "" {
		t.Skip("Set non-empty TEST_GIT_REPO_URL env var to run this test.")
		return nil, nil, true
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	u, err := url.Parse(gitRepoURL)
	require.NoError(t, err)

	password, _ := u.User.Password()

	client, err := gitea.NewClient(fmt.Sprintf("http://%s", u.Host),
		gitea.SetBasicAuth(u.User.Username(), password))
	require.NoError(t, err)

	owner, repo, _ := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")

	t.Log("owner:", owner, "repo:", strings.TrimSuffix(repo, ".git"))

	scm := giteascm.New(client, owner, strings.TrimSuffix(repo, ".git"))

	fs, err := git.NewFilesystem(ctx, scm, gitRepoURL,
		append([]containers.Option[git.Filesystem]{
			git.WithPollInterval(5 * time.Second),
			git.WithAuth(&http.BasicAuth{
				Username: "root",
				Password: "password",
			}),
		},
			opts...)...,
	)
	require.NoError(t, err)

	return fs, scm, false
}

func testdata(extra ...[2]string) map[string][]byte {
	res := map[string][]byte{}
	for k, v := range testdataContents {
		res[k] = v
	}

	for _, e := range extra {
		res[e[0]] = []byte(e[1])
	}

	return res
}

var (
	testdataContents = map[string][]byte{
		"default/test.cup.flipt.io-v1alpha1-Resource-bar.json": []byte(`{
    "apiVersion": "test.cup.flipt.io",
    "kind": "Resource",
    "metadata": {
        "namespace": "default",
        "name": "bar",
        "labels": {
            "baz": "bar"
        },
        "annotations": {}
    },
    "spec": {}
}
`),
		"default/test.cup.flipt.io-v1alpha1-Resource-foo.json": []byte(`{
    "apiVersion": "test.cup.flipt.io",
    "kind": "Resource",
    "metadata": {
        "namespace": "default",
        "name": "foo",
        "labels": {
            "bar": "baz"
        },
        "annotations": {}
    },
    "spec": {}
}
`),
	}

	bazPath     = "default/test.cup.flipt.io-v1alpha1-Resource-baz.json"
	bazContents = []byte(`{
    "apiVersion": "test.cup.flipt.io",
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
}
`)
)
