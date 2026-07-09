import { apiRequest, buildQuery } from "./client";

export function listServers({ page = 1, limit = 10, agentId } = {}) {
    return apiRequest(
        `/v1/server${buildQuery({
            page,
            limit,
            agent_id: agentId || undefined,
        })}`,
    );
}

export function createServer(payload) {
    return apiRequest("/v1/server", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getServer(serverId) {
    return apiRequest(`/v1/server/${serverId}`);
}

export function updateServer(serverId, payload) {
    return apiRequest(`/v1/server/${serverId}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export function listServerJobs(serverId, { page = 1, limit = 10 } = {}) {
    return apiRequest(
        `/v1/server/${serverId}/job${buildQuery({
            page,
            limit,
        })}`,
    );
}
