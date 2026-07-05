import { apiRequest, buildQuery } from "./client";

export function listReleases(applicationId, params = {}) {
    return apiRequest(`/v1/application/${applicationId}/release${buildQuery(params)}`);
}

export function getRelease(applicationId, releaseId) {
    return apiRequest(`/v1/application/${applicationId}/release/${releaseId}`);
}

export function createRelease(applicationId, payload, upsert = false) {
    const query = upsert ? "?upsert=true" : "";
    return apiRequest(`/v1/application/${applicationId}/release${query}`, {
        method: "POST",
        body: JSON.stringify(payload),
    });
}
