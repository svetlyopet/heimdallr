const USERNAME_STORAGE_KEY = "heimdallr.auth.username";
const CSRF_COOKIE_NAME = import.meta.env?.VITE_CSRF_COOKIE_NAME || "heimdallr_csrf";

export function getStoredUsername() {
    return localStorage.getItem(USERNAME_STORAGE_KEY) ?? "";
}

export function setStoredUsername(username) {
    localStorage.setItem(USERNAME_STORAGE_KEY, username);
}

export function clearStoredUsername() {
    localStorage.removeItem(USERNAME_STORAGE_KEY);
}

export function getCSRFToken() {
    const prefix = `${encodeURIComponent(CSRF_COOKIE_NAME)}=`;
    const cookie = document.cookie
        .split(";")
        .map((part) => part.trim())
        .find((part) => part.startsWith(prefix));

    return cookie ? decodeURIComponent(cookie.slice(prefix.length)) : "";
}
