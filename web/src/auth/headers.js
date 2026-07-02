const USERNAME_STORAGE_KEY = "heimdallr.auth.username";
const PASSWORD_STORAGE_KEY = "heimdallr.auth.password";

export function getStoredCredentials() {
    return {
        username: localStorage.getItem(USERNAME_STORAGE_KEY) ?? "",
        password: localStorage.getItem(PASSWORD_STORAGE_KEY) ?? "",
    };
}

export function setStoredCredentials(username, password) {
    localStorage.setItem(USERNAME_STORAGE_KEY, username);
    localStorage.setItem(PASSWORD_STORAGE_KEY, password);
}

export function clearStoredCredentials() {
    localStorage.removeItem(USERNAME_STORAGE_KEY);
    localStorage.removeItem(PASSWORD_STORAGE_KEY);
}

export function getAuthHeaders() {
    const { username, password } = getStoredCredentials();
    if (!username || !password) {
        return {};
    }

    return {
        "X-Auth-Username": username,
        "X-Auth-Password": password,
    };
}
