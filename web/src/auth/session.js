import { reactive } from "vue";
import { apiRequest } from "../api/client.js";
import { clearStoredUsername, getStoredUsername, setStoredUsername } from "./headers.js";

const LEGACY_TOKEN_STORAGE_KEY = "heimdallr.auth.token";

export const sessionState = reactive({
    username: "",
    authenticated: false,
    roles: [],
    checking: false,
    initialized: false,
});

export function initSession() {
    if (sessionState.initialized) {
        return;
    }

    localStorage.removeItem(LEGACY_TOKEN_STORAGE_KEY);
    sessionState.username = getStoredUsername();
    sessionState.initialized = true;
}

export async function loginWithCredentials(username, password) {
    await apiRequest("/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
    });

    sessionState.username = username;
    setStoredUsername(username);
}

export function clearSession() {
    sessionState.username = "";
    sessionState.authenticated = false;
    sessionState.roles = [];
    localStorage.removeItem(LEGACY_TOKEN_STORAGE_KEY);
    clearStoredUsername();
}

export async function logoutSession() {
    try {
        await apiRequest("/v1/auth/logout", { method: "POST" });
    } finally {
        clearSession();
    }
}

export async function ensureSessionAccess() {
    initSession();
    await refreshSessionAccess();
}

export async function refreshSessionAccess() {
    sessionState.checking = true;
    try {
        await apiRequest("/v1/provider?limit=1");
        sessionState.authenticated = true;

        try {
            await apiRequest("/v1/auth/users");
            sessionState.roles = ["admin"];
        } catch {
            sessionState.roles = ["reader"];
        }
    } catch {
        sessionState.authenticated = false;
        sessionState.roles = [];
    } finally {
        sessionState.checking = false;
    }
}
