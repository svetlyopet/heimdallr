//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/tests/flows"
)

func TestComplianceFlow(t *testing.T) {
	ts := startTestServer(t)
	flows.RunComplianceFlow(t, newFlowsClient(t, ts), "integration")
}

func TestOperationsFlow(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders(ts)

	resp, providerBody := doRequest(t, ts, http.MethodPost, "/api/v1/provider", map[string]any{
		"name": "awx",
		"url":  "https://awx.example.com",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	providerID := dataField(t, providerBody)["id"].(string)

	resp, automationBody := doRequest(t, ts, http.MethodPost, "/api/v1/automation", map[string]any{
		"name":        "deploy",
		"url":         "https://awx.example.com/#/templates/job_template/1",
		"provider_id": providerID,
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	automationID := dataField(t, automationBody)["id"].(string)

	jobPath := "/api/v1/automation/" + automationID + "/job"
	resp, _ = doRequest(t, ts, http.MethodPost, jobPath, map[string]any{
		"id":       "1000",
		"status":   "started",
		"location": "global",
		"url":      "https://example.com/#/jobs/playbook/200",
		"metadata": map[string]string{"inventory": "true"},
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, jobBody := doRequest(t, ts, http.MethodPatch, jobPath+"/1000", map[string]any{
		"status":   "success",
		"metadata": map[string]string{"result": "ok"},
		"output":   "dGVzdA==",
	}, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "success", dataField(t, jobBody)["status"])
	require.Equal(t, "dGVzdA==", dataField(t, jobBody)["output"])
}

func TestAuthEnforcement(t *testing.T) {
	ts := startTestServer(t)

	resp, _ := doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	loginResp, _ := doRequest(t, ts, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": "root",
		"password": "wrong-password",
	}, nil)
	require.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, bearerHeader("invalid-token"))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, authHeaders(ts))
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealthEndpoint(t *testing.T) {
	ts := startTestServer(t)

	resp, body := doRequest(t, ts, http.MethodGet, "/api/health", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "ok", body["status"])
}

func TestFleetFlow(t *testing.T) {
	ts := startTestServer(t)
	flows.RunFleetFlow(t, newFlowsClient(t, ts), "integration")
}

func TestAgentNameMustBeUnique(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders(ts)

	resp, _ := doRequest(t, ts, http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "unique-agent",
		"type":    "monitoring",
		"version": "1.0.0",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "unique-agent",
		"type":    "monitoring",
		"version": "2.0.0",
	}, headers)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}
