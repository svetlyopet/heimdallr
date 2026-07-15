import { apiRequest, buildQuery } from "./client";

export function listJobs(automationId, { page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/automation/${automationId}/job${buildQuery({ page, limit })}`);
}

export function createJob(automationId, payload) {
    return apiRequest(`/v1/automation/${automationId}/job`, {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export function getJob(automationId, jobId) {
    return apiRequest(`/v1/automation/${automationId}/job/${jobId}`);
}
