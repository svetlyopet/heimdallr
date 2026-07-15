package flows

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type requiredAgentDef struct {
	Name string
	Type string
}

var fleetRequiredAgents = []requiredAgentDef{
	{Name: "Flexera", Type: "inventory"},
	{Name: "CrowdStrike", Type: "security"},
	{Name: "Tenable", Type: "vulnerability"},
	{Name: "Nagios", Type: "monitoring"},
	{Name: "Tanium", Type: "endpoint"},
}

type fleetServerDef struct {
	Suffix   string
	Location string
	AgentIDs func(map[string]string) []string
}

// FleetComplianceSeed creates required-agent policies, catalog agents, and 20 fleet servers for E2E.
func FleetComplianceSeed(t *testing.T, c *Client, runID string) FleetComplianceState {
	t.Helper()

	state := FleetComplianceState{
		RunID:    runID,
		AgentIDs: make(map[string]string, len(fleetRequiredAgents)),
	}

	for _, agent := range fleetRequiredAgents {
		resp, _ := c.Request(http.MethodPost, "/api/v1/required-agent", map[string]any{
			"agent_name": agent.Name,
			"agent_type": agent.Type,
		}, nil)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	for _, agent := range fleetRequiredAgents {
		resp, body := c.Request(http.MethodPost, "/api/v1/agent", map[string]any{
			"name": agent.Name,
			"type": agent.Type,
		}, nil)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		state.AgentIDs[agent.Name] = DataField(t, body)["id"].(string)
	}

	allAgentIDs := allFleetAgentIDs(state.AgentIDs)
	servers := []fleetServerDef{
		{Suffix: "01", Location: "dc1", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "02", Location: "dc1", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "03", Location: "dc1", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "04", Location: "dc2", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "05", Location: "dc2", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "06", Location: "dc2", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "07", Location: "dc3", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "08", Location: "dc3", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "09", Location: "dc3", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "10", Location: "dc3", AgentIDs: func(map[string]string) []string { return allAgentIDs }},
		{Suffix: "11", Location: "dc1", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Flexera")
		}},
		{Suffix: "12", Location: "dc1", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "CrowdStrike")
		}},
		{Suffix: "13", Location: "dc2", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Tenable")
		}},
		{Suffix: "14", Location: "dc2", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Nagios")
		}},
		{Suffix: "15", Location: "dc3", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Tanium")
		}},
		{Suffix: "16", Location: "dc1", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Flexera", "CrowdStrike")
		}},
		{Suffix: "17", Location: "dc2", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Tenable", "Nagios")
		}},
		{Suffix: "18", Location: "dc3", AgentIDs: func(ids map[string]string) []string {
			return []string{ids["Tanium"]}
		}},
		{Suffix: "19", Location: "dc3", AgentIDs: func(map[string]string) []string { return []string{} }},
		{Suffix: "20", Location: "dc1", AgentIDs: func(ids map[string]string) []string {
			return excludeFleetAgentIDs(ids, "Nagios")
		}},
	}

	for _, server := range servers {
		resp, _ := c.Request(http.MethodPost, "/api/v1/server", map[string]any{
			"hostname":         fmt.Sprintf("fleet-server-%s-%s.example.com", server.Suffix, runID),
			"operating_system": "linux",
			"hypervisor":       "kvm",
			"location":         server.Location,
			"agent_ids":        server.AgentIDs(state.AgentIDs),
		}, nil)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	return state
}

// FleetComplianceVerify asserts fleet compliance analytics match the seeded E2E scenario.
func FleetComplianceVerify(t *testing.T, c *Client) {
	t.Helper()

	resp, body := c.Request(http.MethodGet, "/api/v1/analytics/fleet", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data := DataField(t, body)
	require.Equal(t, float64(20), data["total_servers"])
	require.Equal(t, float64(10), data["compliant_servers"])
	require.Equal(t, float64(10), data["non_compliant_servers"])
	require.Equal(t, float64(50), data["compliance_rate"])
	require.Equal(t, float64(5), data["total_required_agents"])

	coverage, ok := data["required_agent_coverage"].([]any)
	require.True(t, ok)
	require.Len(t, coverage, 5)

	detailsResp, detailsBody := c.Request(http.MethodGet, "/api/v1/analytics/fleet/non-compliant-servers?page=1&limit=10", nil, nil)
	require.Equal(t, http.StatusOK, detailsResp.StatusCode)

	details := ListDataField(t, detailsBody)
	require.Len(t, details, 10)

	pagination, ok := detailsBody["pagination"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(10), pagination["total"])
}

// RunFleetComplianceFlow executes the full fleet compliance E2E seed and verify flow.
func RunFleetComplianceFlow(t *testing.T, c *Client, runID string) {
	t.Helper()

	FleetComplianceSeed(t, c, runID)
	FleetComplianceVerify(t, c)
}

func allFleetAgentIDs(agentIDs map[string]string) []string {
	return []string{
		agentIDs["Flexera"],
		agentIDs["CrowdStrike"],
		agentIDs["Tenable"],
		agentIDs["Nagios"],
		agentIDs["Tanium"],
	}
}

func excludeFleetAgentIDs(agentIDs map[string]string, exclude ...string) []string {
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		excludeSet[name] = struct{}{}
	}

	result := make([]string, 0, len(agentIDs))
	for _, agent := range fleetRequiredAgents {
		if _, skip := excludeSet[agent.Name]; skip {
			continue
		}

		result = append(result, agentIDs[agent.Name])
	}

	return result
}
