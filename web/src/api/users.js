import { apiRequest } from "./client";

export async function listUsers() {
    return apiRequest("/v1/auth/users");
}

export async function createUser(payload) {
    return apiRequest("/v1/auth/users", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updateUser(id, payload) {
    return apiRequest(`/v1/auth/users/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function deleteUser(id) {
    return apiRequest(`/v1/auth/users/${id}`, {
        method: "DELETE",
    });
}
