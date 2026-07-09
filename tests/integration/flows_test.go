//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComplianceFlow(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders(ts)

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

func TestServerAgentComplianceFlow(t *testing.T) {
	ts := startTestServer(t)
	headers := authHeaders(ts)

	resp, orphanBody := doRequest(t, ts, http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "datadog",
		"type":    "monitoring",
		"version": "7.0.0",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orphanData := dataField(t, orphanBody)
	orphanID := orphanData["id"].(string)
	require.Equal(t, float64(0), orphanData["server_count"])

	resp, serverBody := doRequest(t, ts, http.MethodPost, "/api/v1/server", map[string]any{
		"hostname":         "compliance-host.example.com",
		"operating_system": "linux",
		"hypervisor":       "kvm",
		"location":         "dc1",
		"agent_ids":        []string{orphanID},
		"agents": []map[string]any{{
			"name":    "crowdstrike",
			"type":    "security",
			"version": "1.0.0",
		}},
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	serverID := dataField(t, serverBody)["id"].(string)

	resp, serverDetailBody := doRequest(t, ts, http.MethodGet, "/api/v1/server/"+serverID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	relations, ok := dataField(t, serverDetailBody)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), relations["agent_count"])

	resp, unassignedBody := doRequest(t, ts, http.MethodGet, "/api/v1/agent?unassigned=true", nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	unassigned, ok := unassignedBody["data"].([]any)
	require.True(t, ok)
	require.Empty(t, unassigned)

	resp, secondOrphanBody := doRequest(t, ts, http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "sentinel",
		"type":    "security",
		"version": "2.0.0",
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	secondOrphanID := dataField(t, secondOrphanBody)["id"].(string)

	resp, updateBody := doRequest(t, ts, http.MethodPut, "/api/v1/server/"+serverID, map[string]any{
		"agent_ids": []string{secondOrphanID},
	}, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	updatedRelations, ok := dataField(t, updateBody)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), updatedRelations["agent_count"])

	agentPath := "/api/v1/server/" + serverID + "/agent"
	resp, getAgentBody := doRequest(t, ts, http.MethodGet, agentPath+"/"+orphanID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "datadog", dataField(t, getAgentBody)["name"])

	resp, globalAgentBody := doRequest(t, ts, http.MethodGet, "/api/v1/agent/"+orphanID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	globalAgent := dataField(t, globalAgentBody)
	require.Equal(t, "datadog", globalAgent["name"])
	servers, ok := globalAgent["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 1)

	resp, secondServerBody := doRequest(t, ts, http.MethodPost, "/api/v1/server", map[string]any{
		"hostname":         "compliance-host-2.example.com",
		"operating_system": "linux",
		"agent_ids":        []string{orphanID},
	}, headers)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	secondServerID := dataField(t, secondServerBody)["id"].(string)

	resp, globalAgentBody = doRequest(t, ts, http.MethodGet, "/api/v1/agent/"+orphanID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	servers, ok = dataField(t, globalAgentBody)["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 2)

	resp, filteredServersBody := doRequest(t, ts, http.MethodGet, "/api/v1/server?agent_id="+orphanID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	filteredServers, ok := filteredServersBody["data"].([]any)
	require.True(t, ok)
	require.Len(t, filteredServers, 2)

	resp, _ = doRequest(t, ts, http.MethodDelete, agentPath+"/"+orphanID, nil, headers)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, serverAfterDetach := doRequest(t, ts, http.MethodGet, "/api/v1/server/"+serverID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	relationsAfter, ok := dataField(t, serverAfterDetach)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), relationsAfter["agent_count"])

	resp, globalAfterDetach := doRequest(t, ts, http.MethodGet, "/api/v1/agent/"+orphanID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "datadog", dataField(t, globalAfterDetach)["name"])

	resp, secondServerAfterDetach := doRequest(t, ts, http.MethodGet, "/api/v1/server/"+secondServerID, nil, headers)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	secondRelations, ok := dataField(t, secondServerAfterDetach)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), secondRelations["agent_count"])
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
