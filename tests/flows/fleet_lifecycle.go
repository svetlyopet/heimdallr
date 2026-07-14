package flows

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// FleetLifecycleSeed creates an orphan agent and a server with an inline agent.
func FleetLifecycleSeed(t *testing.T, c *Client, runID string) FleetLifecycleState {
	t.Helper()

	state := FleetLifecycleState{
		RunID:    runID,
		Hostname: fmt.Sprintf("e2e-server-%s.example.com", runID),
	}

	resp, orphanBody := c.Request(http.MethodPost, "/api/v1/agent", map[string]any{
		"name": "datadog",
		"type": "monitoring",
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	orphanData := DataField(t, orphanBody)
	state.OrphanID = orphanData["id"].(string)
	require.Equal(t, float64(0), orphanData["server_count"])

	resp, serverBody := c.Request(http.MethodPost, "/api/v1/server", map[string]any{
		"hostname":         state.Hostname,
		"operating_system": "linux",
		"hypervisor":       "kvm",
		"location":         "dc1",
		"agent_ids":        []string{state.OrphanID},
		"agents": []map[string]any{{
			"name": "CrowdStrike",
			"type": "security",
		}},
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	state.ServerID = DataField(t, serverBody)["id"].(string)

	return state
}

// FleetLifecycleVerifyAfterSeed asserts the desired state after seeding via GET requests.
func FleetLifecycleVerifyAfterSeed(t *testing.T, c *Client, state FleetLifecycleState) {
	t.Helper()

	resp, serverDetail := c.Request(http.MethodGet, "/api/v1/server/"+state.ServerID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	relations, ok := DataField(t, serverDetail)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), relations["agent_count"])

	resp, unassignedBody := c.Request(http.MethodGet, "/api/v1/agent?unassigned=true", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, ListDataField(t, unassignedBody))
}

// FleetLifecycleRun attaches a second orphan and shares an agent across two servers.
func FleetLifecycleRun(t *testing.T, c *Client, state *FleetLifecycleState) {
	t.Helper()

	resp, secondOrphanBody := c.Request(http.MethodPost, "/api/v1/agent", map[string]any{
		"name": "sentinel",
		"type": "security",
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	state.SecondOrphanID = DataField(t, secondOrphanBody)["id"].(string)

	resp, _ = c.Request(http.MethodPut, "/api/v1/server/"+state.ServerID, map[string]any{
		"agent_ids": []string{state.SecondOrphanID},
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	state.SecondHostname = fmt.Sprintf("e2e-server-2-%s.example.com", state.RunID)
	resp, secondServerBody := c.Request(http.MethodPost, "/api/v1/server", map[string]any{
		"hostname":         state.SecondHostname,
		"operating_system": "linux",
		"agent_ids":        []string{state.OrphanID},
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	state.SecondServerID = DataField(t, secondServerBody)["id"].(string)
}

// FleetLifecycleVerifyAfterRun asserts the desired state after attach operations via GET requests.
func FleetLifecycleVerifyAfterRun(t *testing.T, c *Client, state FleetLifecycleState) {
	t.Helper()

	resp, serverDetail := c.Request(http.MethodGet, "/api/v1/server/"+state.ServerID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	relations, ok := DataField(t, serverDetail)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), relations["agent_count"])

	agentPath := "/api/v1/server/" + state.ServerID + "/agent"
	resp, getAgentBody := c.Request(http.MethodGet, agentPath+"/"+state.OrphanID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "datadog", DataField(t, getAgentBody)["name"])

	resp, globalAgentBody := c.Request(http.MethodGet, "/api/v1/agent/"+state.OrphanID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	globalAgent := DataField(t, globalAgentBody)
	require.Equal(t, "datadog", globalAgent["name"])
	servers, ok := globalAgent["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 2)

	resp, filteredServersBody := c.Request(http.MethodGet, "/api/v1/server?agent_id="+state.OrphanID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, ListDataField(t, filteredServersBody), 2)
}

// FleetLifecycleDetach removes the shared agent from the first server.
func FleetLifecycleDetach(t *testing.T, c *Client, state FleetLifecycleState) {
	t.Helper()

	agentPath := "/api/v1/server/" + state.ServerID + "/agent/" + state.OrphanID
	resp, _ := c.Request(http.MethodDelete, agentPath, nil, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// FleetLifecycleVerifyFinal asserts the desired state after detach via GET requests.
func FleetLifecycleVerifyFinal(t *testing.T, c *Client, state FleetLifecycleState) {
	t.Helper()

	resp, serverAfterDetach := c.Request(http.MethodGet, "/api/v1/server/"+state.ServerID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	relationsAfter, ok := DataField(t, serverAfterDetach)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), relationsAfter["agent_count"])

	resp, globalAfterDetach := c.Request(http.MethodGet, "/api/v1/agent/"+state.OrphanID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	globalAgent := DataField(t, globalAfterDetach)
	require.Equal(t, "datadog", globalAgent["name"])
	servers, ok := globalAgent["servers"].([]any)
	require.True(t, ok)
	require.Len(t, servers, 1)

	resp, secondServerAfterDetach := c.Request(http.MethodGet, "/api/v1/server/"+state.SecondServerID, nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	secondRelations, ok := DataField(t, secondServerAfterDetach)["relations"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), secondRelations["agent_count"])
}

// RunFleetLifecycleFlow executes the full attach/detach integration flow.
func RunFleetLifecycleFlow(t *testing.T, c *Client, runID string) {
	t.Helper()

	state := FleetLifecycleSeed(t, c, runID)
	FleetLifecycleVerifyAfterSeed(t, c, state)
	FleetLifecycleRun(t, c, &state)
	FleetLifecycleVerifyAfterRun(t, c, state)
	FleetLifecycleDetach(t, c, state)
	FleetLifecycleVerifyFinal(t, c, state)
}
