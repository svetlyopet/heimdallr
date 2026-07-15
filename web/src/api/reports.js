import { apiRequest, buildQuery } from "./client";

export function listAllReports(params = {}) {
    return apiRequest(`/v1/report${buildQuery(params)}`);
}

export function listReports(applicationId, releaseId, params = {}) {
    return apiRequest(
        `/v1/application/${applicationId}/release/${releaseId}/report${buildQuery(params)}`,
    );
}

export function getReport(applicationId, releaseId, reportId) {
    return apiRequest(
        `/v1/application/${applicationId}/release/${releaseId}/report/${reportId}`,
    );
}

export function createReport(applicationId, releaseId, payload) {
    return apiRequest(
        `/v1/application/${applicationId}/release/${releaseId}/report`,
        {
            method: "POST",
            body: JSON.stringify(payload),
        },
    );
}
