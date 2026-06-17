<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Analytics</p>
        <h2>Dashboard</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadAnalytics">
          Refresh
        </button>
      </div>
    </header>

    <section v-if="errorMessage" class="alert">
      <span>{{ errorMessage }}</span>
      <button type="button" @click="errorMessage = ''">Dismiss</button>
    </section>

    <section class="stats-grid">
      <article class="stat-card">
        <span>Total automations</span>
        <strong>{{ analytics.total_automations }}</strong>
      </article>

      <article class="stat-card">
        <span>Total jobs</span>
        <strong>{{ analytics.total_jobs }}</strong>
      </article>

      <article class="stat-card">
        <span>Success rate</span>
        <strong>{{ formatPercent(analytics.success_rate) }}</strong>
      </article>

      <article class="stat-card">
        <span>Failed jobs</span>
        <strong>{{ analytics.failed_jobs }}</strong>
      </article>
    </section>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Locations</p>
            <h3>Success rate by location</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading analytics...</div>

        <div v-else-if="analytics.by_location.length === 0" class="empty-state">
          <strong>No location data</strong>
          <span>Create jobs to populate analytics.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Location</th>
                <th>Total</th>
                <th>Success</th>
                <th>Failed</th>
                <th>Started</th>
                <th>Rate</th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="row in analytics.by_location" :key="row.location">
                <td data-label="Location"><strong>{{ row.location }}</strong></td>
                <td data-label="Total">{{ row.total_jobs }}</td>
                <td data-label="Success">{{ row.successful_jobs }}</td>
                <td data-label="Failed">{{ row.failed_jobs }}</td>
                <td data-label="Started">{{ row.started_jobs }}</td>
                <td data-label="Rate">
                  <div class="rate-cell">
                    <div class="rate-bar">
                      <span :style="{ width: `${Math.min(row.success_rate, 100)}%` }"></span>
                    </div>
                    <strong>{{ formatPercent(row.success_rate) }}</strong>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Automations</p>
            <h3>Success rate by automation</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading analytics...</div>

        <div v-else-if="analytics.by_automation.length === 0" class="empty-state">
          <strong>No automation data</strong>
          <span>Create jobs to populate analytics.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Automation</th>
                <th>Provider</th>
                <th>Total</th>
                <th>Success</th>
                <th>Failed</th>
                <th>Started</th>
                <th>Rate</th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="row in analytics.by_automation" :key="row.automation_id">
                <td data-label="Automation"><strong>{{ row.automation }}</strong></td>
                <td data-label="Provider"><span class="badge">{{ row.provider }}</span></td>
                <td data-label="Total">{{ row.total_jobs }}</td>
                <td data-label="Success">{{ row.successful_jobs }}</td>
                <td data-label="Failed">{{ row.failed_jobs }}</td>
                <td data-label="Started">{{ row.started_jobs }}</td>
                <td data-label="Rate">
                  <div class="rate-cell">
                    <div class="rate-bar">
                      <span :style="{ width: `${Math.min(row.success_rate, 100)}%` }"></span>
                    </div>
                    <strong>{{ formatPercent(row.success_rate) }}</strong>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import { getAutomationAnalytics } from "../api/analytics";

const loading = ref(false);
const errorMessage = ref("");

const analytics = reactive({
  total_automations: 0,
  total_jobs: 0,
  successful_jobs: 0,
  failed_jobs: 0,
  started_jobs: 0,
  success_rate: 0,
  by_location: [],
  by_automation: [],
});

onMounted(loadAnalytics);

async function loadAnalytics() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await getAutomationAnalytics();
    Object.assign(analytics, response.data || {});
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function formatPercent(value) {
  return `${Number(value || 0).toFixed(2)}%`;
}
</script>