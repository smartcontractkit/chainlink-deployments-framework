package artifacts

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_httpGet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)

	ctx := t.Context()
	client := srv.Client()
	resp, err := httpGet(ctx, client, srv.URL+"/x", "test op")
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(b))

	missingResp, errMissing := httpGet(ctx, client, srv.URL+"/missing", "test op")
	require.Error(t, errMissing)
	if missingResp != nil {
		defer missingResp.Body.Close()
	}
	require.Nil(t, missingResp)
	require.Contains(t, errMissing.Error(), "404")
}

func Test_githubRESTGETBytes(t *testing.T) {
	t.Parallel()
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		_, _ = io.WriteString(w, `{"x":1}`)
	}))
	t.Cleanup(srv.Close)

	ctx := t.Context()
	client := srv.Client()
	body, err := githubGet(ctx, client, srv.URL+"/api", "test gh")
	require.NoError(t, err)
	require.Equal(t, "application/vnd.github+json", gotAccept)
	require.Equal(t, `{"x":1}`, string(body))
}
