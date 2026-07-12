//go:build integration

package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/ditest"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/tests/flows"
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
	cfg.Database.DatabaseURL = testutil.PostgresDatabaseURL(t)
	cfg.Auth.BootstrapRootPassword = rootPassword

	injector := ditest.NewServerInjector(t, ditest.WithConfig(cfg))
	srv := ditest.MustInvokeServer(t, injector)

	ts := httptest.NewServer(srv.HTTPHandler())
	t.Cleanup(ts.Close)

	server := testServer{
		Server:   ts,
		RootPass: rootPassword,
	}
	server.Token = flows.LoginToken(t, ts.URL, ts.Client(), "root", rootPassword)

	return server
}

func newFlowsClient(t *testing.T, ts testServer) *flows.Client {
	t.Helper()
	return flows.NewClient(t, ts.Server.URL, ts.Server.Client(), ts.Token)
}

func authHeaders(ts testServer) map[string]string {
	return flows.BearerHeader(ts.Token)
}

func bearerHeader(token string) map[string]string {
	return flows.BearerHeader(token)
}

func doRequest(t *testing.T, ts testServer, method, path string, body any, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()

	client := flows.NewClient(t, ts.Server.URL, ts.Server.Client(), "")
	return client.Request(method, path, body, headers)
}

func dataField(t *testing.T, parsed map[string]any) map[string]any {
	return flows.DataField(t, parsed)
}
