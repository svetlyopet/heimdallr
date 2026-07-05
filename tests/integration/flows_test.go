//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComplianceFlow(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders("root", ts.RootPass)

	resp, appBody := doRequest(t, ts, http.MethodPost, "/api/v1/application", map[string]any{
		"name":           "integration-app",
		"description":    "integration test app",
		"repository_url": "https://example.com/integration-app",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	appID := dataField(t, appBody)["id"].(string)

	resp, releaseBody := doRequest(t, ts, http.MethodPost, "/api/v1/application/"+appID+"/release?upsert=true", map[string]any{
		"version":      "v1.0.0",
		"commit_sha":   "abc123",
		"branch":       "main",
		"pipeline_url": "https://example.com/pipeline/1",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	releaseID := dataField(t, releaseBody)["id"].(string)

	reportPath := "/api/v1/application/" + appID + "/release/" + releaseID + "/report"
	resp, _ = doRequest(t, ts, http.MethodPost, reportPath, map[string]any{
		"id":       "sast-integration",
		"type":     "sast",
		"status":   "started",
		"location": "ci",
		"url":      "https://example.com/run/1",
		"metadata": map[string]string{"tool": "integration"},
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, reportBody := doRequest(t, ts, http.MethodPatch, reportPath+"/sast-integration", map[string]any{
		"status":   "success",
		"metadata": map[string]int{"findings": 0},
		"output":   "dGVzdA==",
	}, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "success", dataField(t, reportBody)["status"])

	resp, listBody := doRequest(t, ts, http.MethodGet, "/api/v1/report?status=success", nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	data, ok := listBody["data"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, data)
}

func TestOperationsFlow(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders("root", ts.RootPass)

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

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, authHeaders("root", "wrong-password"))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, bearerHeader("invalid-token"))
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	resp, _ = doRequest(t, ts, http.MethodGet, "/api/v1/application", nil, authHeaders("root", ts.RootPass))
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHealthEndpoint(t *testing.T) {
	ts := startTestServer(t)

	resp, body := doRequest(t, ts, http.MethodGet, "/api/health", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "ok", body["status"])
}
