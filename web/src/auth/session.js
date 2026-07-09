import { reactive } from "vue";
import { apiRequest } from "../api/client";
import { clearStoredToken, clearStoredUsername, getStoredToken, getStoredUsername, setStoredToken, setStoredUsername } from "./headers";

export const sessionState = reactive({
    username: "",
    token: "",
    authenticated: false,
    roles: [],
    checking: false,
    initialized: false,
});

export function initSession() {
    if (sessionState.initialized) {
        return;
    }

    sessionState.token = getStoredToken();
    sessionState.username = getStoredUsername();
    sessionState.initialized = true;
}

export async function loginWithCredentials(username, password) {
    const response = await apiRequest("/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
        skipAuth: true,
    });

    const token = response?.data?.token ?? "";
    if (!token) {
        throw new Error("missing login token");
    }

    sessionState.username = username;
    sessionState.token = token;
    setStoredToken(token);
    setStoredUsername(username);
}

export function clearSession() {
    sessionState.username = "";
    sessionState.token = "";
    sessionState.authenticated = false;
    sessionState.roles = [];
    clearStoredToken();
    clearStoredUsername();
}

export async function ensureSessionAccess() {
    initSession();
    await refreshSessionAccess();
}

export async function refreshSessionAccess() {
    if (!sessionState.token) {
        sessionState.authenticated = false;
        sessionState.roles = [];
        return;
    }

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
