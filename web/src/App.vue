<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-icon">H</div>
        <div>
          <h1>Heimdallr</h1>
          <p>Automation Console</p>
        </div>
      </div>

      <nav class="nav">
        <button
          class="nav-item"
          :class="{ active: activeView === 'dashboard' }"
          type="button"
          @click="activeView = 'dashboard'"
        >
          <span>📊</span>
          Dashboard
        </button>

        <button
          class="nav-item"
          :class="{ active: activeView === 'providers' }"
          type="button"
          @click="activeView = 'providers'"
        >
          <span>🏢</span>
          Providers
        </button>

        <button
          class="nav-item"
          :class="{ active: activeView === 'automations' }"
          type="button"
          @click="activeView = 'automations'"
        >
          <span>⚙️</span>
          Automations
        </button>

        <button
          class="nav-item"
          :class="{ active: activeView === 'jobs' }"
          type="button"
          @click="activeView = 'jobs'"
        >
          <span>🧾</span>
          Jobs
        </button>
      </nav>

      <div class="sidebar-footer">
        <span class="status-dot"></span>
        Connected to API
      </div>
    </aside>

    <main class="main">
      <DashboardPage v-if="activeView === 'dashboard'" />
      <ProviderPage v-else-if="activeView === 'providers'" />
      <AutomationPage v-else-if="activeView === 'automations'" />
      <JobPage v-else />
    </main>
  </div>
</template>

<script setup>
import { ref } from "vue";
import AutomationPage from "./pages/AutomationPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import JobPage from "./pages/JobPage.vue";
import ProviderPage from "./pages/ProviderPage.vue";

const activeView = ref("dashboard");
</script>