//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReaderDeniedApplicationWrite(t *testing.T) {
	ts := startTestServer(t)
	adminHeaders := authHeaders(ts)

	resp, _ := doRequest(t, ts, http.MethodPost, "/api/v1/auth/users", map[string]any{
		"username": "reader-user",
		"email":    "reader-user@example.com",
		"password": "ReaderPassword123!",
		"roles":    []string{"reader"},
	}, adminHeaders)
	if resp.StatusCode != http.StatusCreated {
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	}

	readerHeaders := loginAsUser(t, ts, "reader-user", "ReaderPassword123!")

	resp, _ = doRequest(t, ts, http.MethodPost, "/api/v1/application", map[string]any{
		"name":           "reader-denied-app",
		"description":    "should fail",
		"repository_url": "https://example.com/reader-denied",
	}, readerHeaders)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application?limit=1", nil, readerHeaders)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestScopedTokenBoundaries(t *testing.T) {
	ts := startTestServer(t)
	adminHeaders := authHeaders(ts)

	resp, appBody := doRequest(t, ts, http.MethodPost, "/api/v1/application", map[string]any{
		"name":           "scoped-token-app",
		"description":    "scope test",
		"repository_url": "https://example.com/scoped",
	}, adminHeaders)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	appID := dataField(t, appBody)["id"].(string)

	resp, tokenBody := doRequest(t, ts, http.MethodPost, "/api/v1/auth/tokens", map[string]any{
		"name":   "app-write-only",
		"scopes": []string{"application:write"},
	}, adminHeaders)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	token := dataField(t, tokenBody)["token"].(string)
	scopedHeaders := bearerHeader(token)

	resp, _ = doRequest(t, ts, http.MethodPost, "/api/v1/application/"+appID+"/release?upsert=true", map[string]any{
		"version":    "v1.0.0",
		"commit_sha": "abc123",
	}, scopedHeaders)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodPost, "/api/v1/automation", map[string]any{
		"name": "scoped-denied-automation",
		"url":  "https://example.com/automation",
	}, scopedHeaders)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func loginAsUser(t *testing.T, ts testServer, username, password string) map[string]string {
	t.Helper()

	resp, body := doRequest(t, ts, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	token, ok := dataField(t, body)["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	return bearerHeader(token)
}
