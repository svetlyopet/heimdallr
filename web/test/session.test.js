import assert from "node:assert/strict";
import test from "node:test";

class MemoryStorage {
    #values = new Map();

    getItem(key) {
        return this.#values.get(key) ?? null;
    }

    setItem(key, value) {
        this.#values.set(key, String(value));
    }

    removeItem(key) {
        this.#values.delete(key);
    }

    values() {
        return [...this.#values.values()];
    }
}

test("browser login never stores the session bearer token", async () => {
    globalThis.localStorage = new MemoryStorage();
    globalThis.fetch = async () => ({
        status: 200,
        ok: true,
        json: async () => ({ data: { token: "session-secret" } }),
    });

    localStorage.setItem("heimdallr.auth.token", "legacy-secret");
    const { initSession, loginWithCredentials, logoutSession } = await import("../src/auth/session.js");
    globalThis.document = { cookie: "" };

    initSession();
    await loginWithCredentials("root", "password");

    assert.equal(localStorage.getItem("heimdallr.auth.token"), null);
    assert.equal(localStorage.getItem("heimdallr.auth.username"), "root");
    assert.equal(localStorage.values().includes("session-secret"), false);

    let logoutOptions;
    document.cookie = "heimdallr_csrf=csrf-secret";
    globalThis.fetch = async (_url, options) => {
        logoutOptions = options;
        return {
            status: 204,
            ok: true,
            json: async () => ({}),
        };
    };

    await logoutSession();

    assert.equal(logoutOptions.credentials, "same-origin");
    assert.equal(logoutOptions.headers["X-CSRF-Token"], "csrf-secret");
});
