import { apiRequest } from "./client";

export function getAutomationAnalytics() {
    return apiRequest("/v1/analytics/automation");
}

export function getAutomationAnalyticsById(automationId) {
    return apiRequest(`/v1/analytics/automation/${automationId}`);
}

export function getComplianceAnalytics() {
    return apiRequest("/v1/analytics/compliance");
}

export function getFleetComplianceAnalytics() {
    return apiRequest("/v1/analytics/fleet");
}

export function listNonCompliantServers({ page = 1, limit = 10 } = {}) {
    return apiRequest(`/v1/analytics/fleet/non-compliant-servers?page=${page}&limit=${limit}`);
}