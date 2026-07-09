const TOKEN_STORAGE_KEY = "heimdallr.auth.token";
const USERNAME_STORAGE_KEY = "heimdallr.auth.username";

export function getStoredToken() {
    return localStorage.getItem(TOKEN_STORAGE_KEY) ?? "";
}

export function setStoredToken(token) {
    localStorage.setItem(TOKEN_STORAGE_KEY, token);
}

export function clearStoredToken() {
    localStorage.removeItem(TOKEN_STORAGE_KEY);
}

export function getStoredUsername() {
    return localStorage.getItem(USERNAME_STORAGE_KEY) ?? "";
}

export function setStoredUsername(username) {
    localStorage.setItem(USERNAME_STORAGE_KEY, username);
}

export function clearStoredUsername() {
    localStorage.removeItem(USERNAME_STORAGE_KEY);
}

export function getAuthHeaders() {
    const token = getStoredToken();
    if (!token) {
        return {};
    }

    return {
        Authorization: `Bearer ${token}`,
    };
}
