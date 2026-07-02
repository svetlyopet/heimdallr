import { reactive } from "vue";
import { apiRequest } from "../api/client";
import { clearStoredCredentials, getStoredCredentials, setStoredCredentials } from "./headers";

export const sessionState = reactive({
    username: "",
    password: "",
    authenticated: false,
    roles: [],
    checking: false,
    initialized: false,
});

export function initSession() {
    if (sessionState.initialized) {
        return;
    }

    const { username, password } = getStoredCredentials();
    sessionState.username = username;
    sessionState.password = password;
    sessionState.initialized = true;
}

export function setSessionCredentials(username, password) {
    sessionState.username = username;
    sessionState.password = password;
    setStoredCredentials(username, password);
}

export function clearSession() {
    sessionState.username = "";
    sessionState.password = "";
    sessionState.authenticated = false;
    sessionState.roles = [];
    clearStoredCredentials();
}

export async function ensureSessionAccess() {
    initSession();
    await refreshSessionAccess();
}

export async function refreshSessionAccess() {
    if (!sessionState.username || !sessionState.password) {
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
