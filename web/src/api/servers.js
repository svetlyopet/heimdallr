import { apiRequest, buildQuery } from "./client";

export function listServers({ page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/server${buildQuery({ page, limit })}`);
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
