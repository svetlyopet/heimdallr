//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/ditest"
)

type testServer struct {
	Server   *httptest.Server
	RootPass string
	Token    string
}

func startTestServer(t *testing.T) testServer {
	t.Helper()

	rootPassword := "IntegrationTestPassword12!"
	cfg := config.DefaultTestConfig(bytes.NewBuffer(nil))
	cfg.Database.DatabasePath = filepath.Join(t.TempDir(), "heimdallr.db")
	cfg.Auth.BootstrapRootPassword = rootPassword

	injector := ditest.NewServerInjector(t, ditest.WithConfig(cfg))
	srv := ditest.MustInvokeServer(t, injector)

	ts := httptest.NewServer(srv.HTTPHandler())
	t.Cleanup(ts.Close)

	server := testServer{
		Server:   ts,
		RootPass: rootPassword,
	}
	server.Token = loginToken(t, server, "root", rootPassword)

	return server
}

func doRequest(t *testing.T, ts testServer, method, path string, body any, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, reader)
	require.NoError(t, err)
	req.Host = "localhost"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := ts.Server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	var parsed map[string]any
	if resp.Body != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		require.NoError(t, readErr)
		if len(bodyBytes) > 0 {
			require.NoError(t, json.Unmarshal(bodyBytes, &parsed))
		}
	}

	return resp, parsed
}

func loginToken(t *testing.T, ts testServer, username, password string) string {
	t.Helper()

	resp, parsed := doRequest(t, ts, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data := dataField(t, parsed)
	token, ok := data["token"].(string)
	require.True(t, ok, fmt.Sprintf("expected login token, got %#v", parsed))
	return token
}

func authHeaders(ts testServer) map[string]string {
	return bearerHeader(ts.Token)
}

func bearerHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

func dataField(t *testing.T, parsed map[string]any) map[string]any {
	t.Helper()

	data, ok := parsed["data"].(map[string]any)
	require.True(t, ok, fmt.Sprintf("expected data object, got %#v", parsed))
	return data
}
