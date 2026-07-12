const API_BASE_URL = "/api";

import { getCSRFToken } from "../auth/headers.js";

export async function apiRequest(path, options = {}) {
    const headers = {
        Accept: "application/json",
        "Content-Type": "application/json",
        ...options.headers,
    };

    const method = (options.method ?? "GET").toUpperCase();
    if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(method)) {
        const csrfToken = getCSRFToken();
        if (csrfToken) {
            headers["X-CSRF-Token"] = csrfToken;
        }
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        headers,
        credentials: "same-origin",
    });

    if (response.status === 204) {
        return null;
    }

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
        throw new Error(data.error || "Request failed");
    }

    return data;
}

export function buildQuery(params = {}) {
    const search = new URLSearchParams();

    Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== "") {
            search.set(key, String(value));
        }
    });

    const query = search.toString();
    return query ? `?${query}` : "";
}