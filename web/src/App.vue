<template>
  <RouterView v-if="!showShell" />

  <div v-else class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-icon">H</div>
        <div>
          <h1>Heimdallr</h1>
          <p>Automation Console</p>
        </div>
      </div>

      <nav class="nav">
        <RouterLink class="nav-item" to="/dashboard">
          <span>📊</span>
          Dashboard
        </RouterLink>

        <RouterLink class="nav-item" to="/providers">
          <span>🏢</span>
          Providers
        </RouterLink>

        <RouterLink class="nav-item" to="/automations">
          <span>⚙️</span>
          Automations
        </RouterLink>

        <RouterLink class="nav-item" to="/jobs">
          <span>🧾</span>
          Jobs
        </RouterLink>

        <RouterLink v-if="isAdmin" class="nav-item" to="/users">
          <span>👥</span>
          Users
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <span class="status-dot"></span>
        Connected to API
      </div>
    </aside>

    <main class="main">
      <header class="app-topbar">
        <div class="user-card">
          <div>
            <p class="eyebrow">User</p>
            <strong>{{ sessionState.username || "Not configured" }}</strong>
          </div>
          <span class="role-pill" :class="isAdmin ? 'role-pill-admin' : 'role-pill-reader'">
            {{ isAdmin ? "admin" : "reader" }}
          </span>
        </div>

        <button class="button button-secondary" type="button" @click="logout">
          Logout
        </button>
      </header>

      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
import { clearSession, ensureSessionAccess, sessionState } from "./auth/session";

const route = useRoute();
const router = useRouter();

const isAdmin = computed(() => sessionState.roles.includes("admin"));
const showShell = computed(() => route.name !== "login");

function logout() {
  clearSession();
  router.push({ name: "login" });
}

onMounted(async () => {
  await ensureSessionAccess();
});
</script>