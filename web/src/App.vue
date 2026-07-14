<template>
  <RouterView v-if="!showShell" />

  <div v-else class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-icon">H</div>
        <div>
          <h1>Heimdallr</h1>
          <p>Fleet, Software & Operations Hub</p>
        </div>
      </div>

      <nav class="nav">
        <RouterLink class="nav-item nav-item-top" to="/dashboard">
          Dashboard
        </RouterLink>

        <div class="nav-scroll">
          <SidebarNavGroup
            v-for="group in visibleNavGroups"
            :key="group.id"
            :title="group.title"
            :group-id="`nav-group-${group.id}`"
            :expanded="expandedGroups[group.id]"
            @toggle="toggleGroup(group.id)"
          >
            <RouterLink
              v-for="item in group.items"
              :key="item.to"
              class="nav-item"
              :to="item.to"
            >
              <span>{{ item.icon }}</span>
              {{ item.label }}
            </RouterLink>
          </SidebarNavGroup>
        </div>
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
import { computed, onMounted, reactive, watch } from "vue";
import "./stylesheets/app-shell.css";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
import SidebarNavGroup from "./components/SidebarNavGroup.vue";
import { ensureSessionAccess, logoutSession, sessionState } from "./auth/session";

const route = useRoute();
const router = useRouter();

const navGroups = [
  {
    id: "fleet",
    title: "Fleet",
    routes: ["/servers", "/agents"],
    items: [
      { to: "/servers", label: "Servers", icon: "🖥️" },
      { to: "/agents", label: "Agents", icon: "🤖" },
    ],
  },
  {
    id: "software",
    title: "Software",
    routes: ["/applications", "/releases", "/reports"],
    items: [
      { to: "/applications", label: "Applications", icon: "📦" },
      { to: "/releases", label: "Releases", icon: "🚀" },
      { to: "/reports", label: "Reports", icon: "📋" },
    ],
  },
  {
    id: "operations",
    title: "Operations",
    routes: ["/providers", "/automations", "/jobs"],
    items: [
      { to: "/providers", label: "Providers", icon: "🏢" },
      { to: "/automations", label: "Automations", icon: "⚙️" },
      { to: "/jobs", label: "Jobs", icon: "🧾" },
    ],
  },
  {
    id: "administration",
    title: "Administration",
    adminOnly: true,
    routes: ["/users"],
    items: [{ to: "/users", label: "Users", icon: "👥" }],
  },
];

const isAdmin = computed(() => sessionState.roles.includes("admin"));
const showShell = computed(() => route.name !== "login");

const visibleNavGroups = computed(() =>
  navGroups.filter((group) => !group.adminOnly || isAdmin.value),
);

const expandedGroups = reactive(
  Object.fromEntries(navGroups.map((group) => [group.id, false])),
);

function matchesGroupRoute(path, group) {
  return group.routes.some(
    (groupRoute) => path === groupRoute || path.startsWith(`${groupRoute}/`),
  );
}

function expandGroupForRoute(path) {
  for (const group of navGroups) {
    if (matchesGroupRoute(path, group)) {
      expandedGroups[group.id] = true;
      return;
    }
  }
}

function toggleGroup(groupId) {
  expandedGroups[groupId] = !expandedGroups[groupId];
}

async function logout() {
  await logoutSession();
  await router.push({ name: "login" });
}

watch(
  () => route.path,
  (path) => {
    expandGroupForRoute(path);
  },
  { immediate: true },
);

onMounted(async () => {
  await ensureSessionAccess();
});
</script>
