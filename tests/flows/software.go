package flows

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// SoftwareSeed creates an application and a scoped API token.
func SoftwareSeed(t *testing.T, c *Client, runID string) SoftwareState {
	t.Helper()

	appName := fmt.Sprintf("e2e-software-app-%s", runID)
	tokenName := fmt.Sprintf("e2e-software-token-%s", runID)

	resp, appBody := c.Request(http.MethodPost, "/api/v1/application", map[string]any{
		"name":           appName,
		"description":    "e2e",
		"repository_url": "https://example.com/e2e",
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	appID := DataField(t, appBody)["id"].(string)

	resp, tokenBody := c.Request(http.MethodPost, "/api/v1/auth/tokens", map[string]any{
		"name":   tokenName,
		"scopes": []string{"application:write"},
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	token := DataField(t, tokenBody)["token"].(string)

	resp, _ = c.Request(http.MethodGet, "/api/v1/application/"+appID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	return SoftwareState{
		ApplicationID:  appID,
		Token:          token,
		ReleaseVersion: "v1.0.0-e2e",
		ReportID:       fmt.Sprintf("sast-e2e-%s", runID),
		CommitSHA:      "abc123",
	}
}

// SoftwareRun pushes a release and report using the application token.
func SoftwareRun(t *testing.T, c *Client, state *SoftwareState) {
	t.Helper()

	tokenClient := c.WithToken(state.Token)

	resp, releaseBody := tokenClient.Request(http.MethodPost,
		"/api/v1/application/"+state.ApplicationID+"/release?upsert=true",
		map[string]any{
			"version":      state.ReleaseVersion,
			"commit_sha":   state.CommitSHA,
			"branch":       "main",
			"pipeline_url": "https://example.com/pipeline/e2e",
		}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	state.ReleaseID = DataField(t, releaseBody)["id"].(string)

	reportPath := fmt.Sprintf("/api/v1/application/%s/release/%s/report", state.ApplicationID, state.ReleaseID)
	resp, _ = tokenClient.Request(http.MethodPost, reportPath, map[string]any{
		"id":       state.ReportID,
		"type":     "sast",
		"status":   "started",
		"location": "ci",
		"url":      "https://example.com/run/e2e",
		"metadata": map[string]string{"tool": "e2e"},
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	output := base64.StdEncoding.EncodeToString([]byte("<h1>E2E SAST report</h1>"))
	resp, _ = tokenClient.Request(http.MethodPatch, reportPath+"/"+state.ReportID, map[string]any{
		"status":   "success",
		"metadata": map[string]any{"findings": 0, "tool": "e2e"},
		"output":   output,
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// SoftwareVerify asserts the desired software catalog state via GET requests.
func SoftwareVerify(t *testing.T, c *Client, state SoftwareState) {
	t.Helper()

	releasePath := fmt.Sprintf("/api/v1/application/%s/release/%s", state.ApplicationID, state.ReleaseID)
	resp, releaseJSON := c.Request(http.MethodGet, releasePath, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	releaseData := DataField(t, releaseJSON)
	require.Equal(t, state.ReleaseVersion, releaseData["version"])
	require.Equal(t, state.CommitSHA, releaseData["commit_sha"])

	reportPath := releasePath + "/report/" + state.ReportID
	resp, reportJSON := c.Request(http.MethodGet, reportPath, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "success", DataField(t, reportJSON)["status"])

	resp, listJSON := c.Request(http.MethodGet, "/api/v1/report?status=success", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	found := false
	for _, item := range ListDataField(t, listJSON) {
		entry, ok := item.(map[string]any)
		require.True(t, ok)
		if entry["id"] == state.ReportID {
			found = true
			break
		}
	}
	require.True(t, found, "report %s not found in success list", state.ReportID)
}

// RunSoftwareFlow executes the full software seed/run/verify flow.
func RunSoftwareFlow(t *testing.T, c *Client, runID string) {
	t.Helper()

	state := SoftwareSeed(t, c, runID)
	SoftwareRun(t, c, &state)
	SoftwareVerify(t, c, state)
}
