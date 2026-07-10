package flows

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// FleetSeed creates an orphan agent and a server with an inline agent.
func FleetSeed(t *testing.T, c *Client, runID string) FleetState {
	t.Helper()

	state := FleetState{
		RunID:    runID,
		Hostname: fmt.Sprintf("e2e-server-%s.example.com", runID),
	}

	resp, orphanBody := c.Request(http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "datadog",
		"type":    "monitoring",
		"version": "7.0.0",
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
			"name":    "crowdstrike",
			"type":    "security",
			"version": "1.0.0",
		}},
	}, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	state.ServerID = DataField(t, serverBody)["id"].(string)

	return state
}

// FleetVerifyAfterSeed asserts the desired state after seeding via GET requests.
func FleetVerifyAfterSeed(t *testing.T, c *Client, state FleetState) {
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

// FleetRun attaches a second orphan, shares an agent across two servers.
func FleetRun(t *testing.T, c *Client, state *FleetState) {
	t.Helper()

	resp, secondOrphanBody := c.Request(http.MethodPost, "/api/v1/agent", map[string]any{
		"name":    "sentinel",
		"type":    "security",
		"version": "2.0.0",
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

// FleetVerifyAfterRun asserts the desired state after attach operations via GET requests.
func FleetVerifyAfterRun(t *testing.T, c *Client, state FleetState) {
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

// FleetDetach removes the shared agent from the first server.
func FleetDetach(t *testing.T, c *Client, state FleetState) {
	t.Helper()

	agentPath := "/api/v1/server/" + state.ServerID + "/agent/" + state.OrphanID
	resp, _ := c.Request(http.MethodDelete, agentPath, nil, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// FleetVerifyFinal asserts the desired state after detach via GET requests.
func FleetVerifyFinal(t *testing.T, c *Client, state FleetState) {
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

// RunFleetFlow executes the full fleet seed/run/verify flow.
func RunFleetFlow(t *testing.T, c *Client, runID string) {
	t.Helper()

	state := FleetSeed(t, c, runID)
	FleetVerifyAfterSeed(t, c, state)
	FleetRun(t, c, &state)
	FleetVerifyAfterRun(t, c, state)
	FleetDetach(t, c, state)
	FleetVerifyFinal(t, c, state)
}
