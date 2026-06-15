import { apiRequest, buildQuery } from "./client";

export function listAutomations({ page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/automation${buildQuery({ page, limit })}`);
}

export function createAutomation(payload) {
    return apiRequest("/v1/automation", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getAutomation(id) {
    return apiRequest(`/v1/automation/${id}`);
}

export function updateAutomation(id, payload) {
    return apiRequest(`/v1/automation/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export function deleteAutomation(id) {
    return apiRequest(`/v1/automation/${id}`, {
        method: "DELETE",
    });
}