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
      </nav>

      <div class="sidebar-footer">
        <span class="status-dot"></span>
        Connected to API
      </div>
    </aside>

    <main class="main">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { RouterLink, RouterView } from "vue-router";
</script>