import { apiRequest, buildQuery } from "./client";

export function listProviders({ page = 1, limit = 100 } = {}) {
    return apiRequest(`/v1/provider${buildQuery({ page, limit })}`);
}

export function createProvider(payload) {
    return apiRequest("/v1/provider", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getProvider(id) {
    return apiRequest(`/v1/provider/${id}`);
}