import { apiRequest, buildQuery } from "./client";

export function listApplications(params = {}) {
    return apiRequest(`/v1/application${buildQuery(params)}`);
}

export function getApplication(applicationId) {
    return apiRequest(`/v1/application/${applicationId}`);
}

export function createApplication(payload) {
    return apiRequest("/v1/application", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}
