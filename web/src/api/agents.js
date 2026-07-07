import { apiRequest, buildQuery } from "./client";

export function listAgents(serverId, { page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/server/${serverId}/agent${buildQuery({ page, limit })}`);
}

export function createAgent(serverId, payload) {
    return apiRequest(`/v1/server/${serverId}/agent`, {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getAgent(serverId, agentId) {
    return apiRequest(`/v1/server/${serverId}/agent/${agentId}`);
}

export function deleteAgent(serverId, agentId) {
    return apiRequest(`/v1/server/${serverId}/agent/${agentId}`, {
        method: "DELETE",
    });
}
