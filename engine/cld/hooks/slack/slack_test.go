package slack

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func okResponse(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(chatPostMessageResponse{OK: true})
	require.NoError(t, err)

	return b
}

func errResponse(t *testing.T, msg string) []byte {
	t.Helper()
	b, err := json.Marshal(chatPostMessageResponse{OK: false, Error: msg})
	require.NoError(t, err)

	return b
}

func withTestURL(t *testing.T, url string) {
	t.Helper()
	orig := slackAPIPostURL
	slackAPIPostURL = url
	t.Cleanup(func() { slackAPIPostURL = orig })
}

func capturePayload(t *testing.T) (*chatPostMessagePayload, *string) {
	t.Helper()
	var received chatPostMessagePayload
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		_, _ = w.Write(okResponse(t))
	}))
	t.Cleanup(srv.Close)
	withTestURL(t, srv.URL)

	return &received, &authHeader
}

//nolint:paralleltest // mutates package-level slackAPIPostURL
func TestNotify_PostsBlockKit(t *testing.T) {
	received, authHeader := capturePayload(t)

	hook := Notify("xoxb-test-token", "#deploys", "deploying tokens")
	err := hook.Func(t.Context(), changeset.PreHookParams{
		ChangesetKey: "0001_deploy",
	})

	require.NoError(t, err)
	assert.Equal(t, "#deploys", received.Channel)
	assert.Equal(t, "Bearer xoxb-test-token", *authHeader)
	assert.Equal(t, "Changeset 0001_deploy: deploying tokens", received.Text)

	require.Len(t, received.Blocks, 1)
	assert.Equal(t, "header", received.Blocks[0].Type)
	assert.Contains(t, received.Blocks[0].Text.Text, "Starting")

	require.Len(t, received.Attachments, 1)
	att := received.Attachments[0]
	assert.Equal(t, colorBlue, att.Color)
	require.Len(t, att.Blocks, 2)
	assert.Equal(t, "section", att.Blocks[0].Type)
	require.Len(t, att.Blocks[0].Fields, 2)
	assert.Contains(t, att.Blocks[0].Fields[0].Text, "0001_deploy")
	assert.Contains(t, att.Blocks[0].Fields[1].Text, "deploying tokens")
	assert.Equal(t, "context", att.Blocks[1].Type)
}

func TestNotify_EmptyToken_Noop(t *testing.T) {
	t.Parallel()

	hook := Notify("", "#deploys", "should not send")
	err := hook.Func(t.Context(), changeset.PreHookParams{
		Env: changeset.HookEnv{Logger: logger.Test(t)},
	})

	require.NoError(t, err)
}

//nolint:paralleltest // mutates package-level slackAPIPostURL
func TestNotify_APIError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(errResponse(t, "channel_not_found"))
	}))
	t.Cleanup(srv.Close)
	withTestURL(t, srv.URL)

	hook := Notify("xoxb-token", "#bad", "test")
	err := hook.Func(t.Context(), changeset.PreHookParams{})

	require.ErrorContains(t, err, "channel_not_found")
}

//nolint:paralleltest // mutates package-level slackAPIPostURL
func TestNotify_HTTPError_ReturnsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	withTestURL(t, srv.URL)

	hook := Notify("xoxb-token", "#ch", "test")
	err := hook.Func(t.Context(), changeset.PreHookParams{})

	require.ErrorContains(t, err, "unexpected status 429")
}

func TestNotify_Metadata(t *testing.T) {
	t.Parallel()

	hook := Notify("token", "#ch", "msg")
	assert.Equal(t, "slack-notify", hook.Name)
	assert.Equal(t, changeset.Warn, hook.FailurePolicy)
	assert.Equal(t, 10*time.Second, hook.Timeout)
}

//nolint:paralleltest // mutates package-level slackAPIPostURL
func TestResult_Success(t *testing.T) {
	received, _ := capturePayload(t)

	hook := Result("xoxb-token", "#deploys")
	err := hook.Func(t.Context(), changeset.PostHookParams{
		ChangesetKey: "0001_deploy",
		Err:          nil,
	})

	require.NoError(t, err)
	assert.Equal(t, "#deploys", received.Channel)
	assert.Equal(t, "Changeset 0001_deploy succeeded", received.Text)

	require.Len(t, received.Blocks, 1)
	assert.Equal(t, "header", received.Blocks[0].Type)
	assert.Contains(t, received.Blocks[0].Text.Text, "Succeeded")

	require.Len(t, received.Attachments, 1)
	att := received.Attachments[0]
	assert.Equal(t, colorGreen, att.Color)
	require.Len(t, att.Blocks, 2)
	require.Len(t, att.Blocks[0].Fields, 2)
	assert.Contains(t, att.Blocks[0].Fields[0].Text, "0001_deploy")
	assert.Contains(t, att.Blocks[0].Fields[1].Text, "Succeeded")
	assert.Equal(t, "context", att.Blocks[1].Type)
}

//nolint:paralleltest // mutates package-level slackAPIPostURL
func TestResult_Failure(t *testing.T) {
	received, _ := capturePayload(t)

	hook := Result("xoxb-token", "#deploys")
	err := hook.Func(t.Context(), changeset.PostHookParams{
		ChangesetKey: "0002_migrate",
		Err:          errors.New("tx reverted"),
	})

	require.NoError(t, err)
	assert.Equal(t, "Changeset 0002_migrate failed: tx reverted", received.Text)

	require.Len(t, received.Blocks, 1)
	assert.Equal(t, "header", received.Blocks[0].Type)
	assert.Contains(t, received.Blocks[0].Text.Text, "Failed")

	require.Len(t, received.Attachments, 1)
	att := received.Attachments[0]
	assert.Equal(t, colorRed, att.Color)
	require.Len(t, att.Blocks, 3)
	assert.Contains(t, att.Blocks[0].Fields[0].Text, "0002_migrate")
	assert.Contains(t, att.Blocks[1].Text.Text, "tx reverted")
	assert.Equal(t, "context", att.Blocks[2].Type)
}

func TestResult_EmptyToken_Noop(t *testing.T) {
	t.Parallel()

	hook := Result("", "#deploys")
	err := hook.Func(t.Context(), changeset.PostHookParams{
		Env:          changeset.HookEnv{Logger: logger.Test(t)},
		ChangesetKey: "0001_deploy",
	})

	require.NoError(t, err)
}

func TestResult_Metadata(t *testing.T) {
	t.Parallel()

	hook := Result("token", "#ch")
	assert.Equal(t, "slack-result", hook.Name)
	assert.Equal(t, changeset.Warn, hook.FailurePolicy)
	assert.Equal(t, 10*time.Second, hook.Timeout)
}
