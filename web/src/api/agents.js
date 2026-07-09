import { apiRequest, buildQuery } from "./client";

export function listAgents({ page = 1, limit = 10, unassigned, serverId, agentId } = {}) {
    return apiRequest(
        `/v1/agent${buildQuery({
            page,
            limit,
            unassigned: unassigned === true ? true : undefined,
            server_id: serverId || undefined,
            agent_id: agentId || undefined,
        })}`,
    );
}

export function createAgent(payload) {
    return apiRequest("/v1/agent", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getAgent(agentId) {
    return apiRequest(`/v1/agent/${agentId}`);
}

export function deleteAgent(agentId) {
    return apiRequest(`/v1/agent/${agentId}`, {
        method: "DELETE",
    });
}

export function listAgentServers(agentId, { page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/agent/${agentId}/server${buildQuery({ page, limit })}`);
}

export function listAgentsForServer(serverId, { page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/server/${serverId}/agent${buildQuery({ page, limit })}`);
}

export function attachAgent(serverId, agentId) {
    return apiRequest(`/v1/server/${serverId}/agent`, {
        method: "POST",
        body: JSON.stringify({ agent_id: agentId }),
    });
}

export function createAgentOnServer(serverId, payload) {
    return apiRequest(`/v1/server/${serverId}/agent`, {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getAgentOnServer(serverId, agentId) {
    return apiRequest(`/v1/server/${serverId}/agent/${agentId}`);
}

export function detachAgent(serverId, agentId) {
    return apiRequest(`/v1/server/${serverId}/agent/${agentId}`, {
        method: "DELETE",
    });
}
