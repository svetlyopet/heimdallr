const API_BASE_URL = "/api";

import { getAuthHeaders } from "../auth/headers";

export async function apiRequest(path, options = {}) {
    const headers = {
        Accept: "application/json",
        "Content-Type": "application/json",
        ...options.headers,
    };

    if (!options.skipAuth) {
        Object.assign(headers, getAuthHeaders());
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
        headers,
        ...options,
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