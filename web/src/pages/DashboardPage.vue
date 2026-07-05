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

    <section class="stats-grid">
      <article class="stat-card">
        <span>Applications</span>
        <strong>{{ compliance.total_applications }}</strong>
      </article>
      <article class="stat-card">
        <span>Releases</span>
        <strong>{{ compliance.total_releases }}</strong>
      </article>
      <article class="stat-card">
        <span>Reports</span>
        <strong>{{ compliance.total_reports }}</strong>
      </article>
      <article class="stat-card">
        <span>Success rate</span>
        <strong>{{ formatPercent(compliance.success_rate) }}</strong>
      </article>
    </section>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section class="page-stack">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Compliance</p>
            <h3>Release compliance overview</h3>
          </div>

          <div class="panel-header-actions">
            <RouterLink class="button button-secondary" to="/releases">All releases</RouterLink>
            <RouterLink class="button button-secondary" to="/reports?status=failed">Failed reports</RouterLink>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading compliance analytics...</div>
        <div v-else-if="compliance.by_application.length === 0" class="empty-state">
          <strong>No compliance data</strong>
          <span>Create applications and push CI reports to populate this section.</span>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Application</th>
                <th>Latest version</th>
                <th>Reports</th>
                <th>Success</th>
                <th>Failed</th>
                <th>Rate</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in compliance.by_application" :key="row.application_id">
                <td data-label="Application"><strong>{{ row.application }}</strong></td>
                <td data-label="Latest version">{{ row.latest_version || "—" }}</td>
                <td data-label="Reports">{{ row.total_reports }}</td>
                <td data-label="Success">{{ row.successful_reports }}</td>
                <td data-label="Failed">{{ row.failed_reports }}</td>
                <td data-label="Rate">{{ formatPercent(row.success_rate) }}</td>
                <td data-label="Actions">
                  <RouterLink
                    v-if="row.latest_release_id"
                    class="button button-secondary"
                    :to="{
                      name: 'release-detail',
                      params: { id: row.application_id, releaseId: row.latest_release_id },
                    }"
                  >
                    View
                  </RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

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
                  <td data-label="Rate">{{ formatPercent(row.success_rate) }}</td>
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

            <RouterLink class="button button-secondary" to="/jobs">View jobs</RouterLink>
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
                  <td data-label="Rate">{{ formatPercent(row.success_rate) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>
      </section>
    </section>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import { RouterLink } from "vue-router";
import { getAutomationAnalytics, getComplianceAnalytics } from "../api/analytics";
import AppAlert from "../components/AppAlert.vue";
import { formatPercent } from "../utils/format";

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

const compliance = reactive({
  total_applications: 0,
  total_releases: 0,
  total_reports: 0,
  successful_reports: 0,
  failed_reports: 0,
  started_reports: 0,
  success_rate: 0,
  by_application: [],
});

onMounted(loadAnalytics);

async function loadAnalytics() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [automationResponse, complianceResponse] = await Promise.all([
      getAutomationAnalytics(),
      getComplianceAnalytics(),
    ]);

    Object.assign(analytics, automationResponse.data || {});
    Object.assign(compliance, complianceResponse.data || {});
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}
</script>
