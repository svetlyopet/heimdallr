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
          <span>Total servers</span>
          <strong>{{ fleet.total_servers }}</strong>
        </article>
        <article class="stat-card">
          <span>Compliant servers</span>
          <strong>{{ fleet.compliant_servers }}</strong>
        </article>
        <article class="stat-card">
          <span>Non-compliant servers</span>
          <strong>{{ fleet.non_compliant_servers }}</strong>
        </article>
        <article class="stat-card">
          <span>Fleet compliance rate</span>
          <strong>{{ formatPercent(fleet.compliance_rate) }}</strong>
        </article>
      </section>

      <section class="dashboard-grid">
        <article class="panel table-panel">
          <div class="panel-header">
            <div>
              <p class="eyebrow">Fleet</p>
              <h3>Required agent coverage</h3>
            </div>
            <RouterLink class="button button-secondary" to="/servers">All servers</RouterLink>
          </div>

          <div v-if="loading" class="empty-state">Loading fleet analytics...</div>
          <div v-else-if="fleet.required_agent_coverage.length === 0" class="empty-state">
            <strong>No required agents configured</strong>
            <span>Add required agent policies to track fleet compliance.</span>
          </div>
          <div v-else class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Agent</th>
                  <th>Installed on</th>
                  <th>Missing from</th>
                  <th>Rate</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in fleet.required_agent_coverage" :key="row.agent_name">
                  <td data-label="Agent"><strong>{{ row.agent_name }}</strong></td>
                  <td data-label="Installed on">{{ row.servers_with }}</td>
                  <td data-label="Missing from">{{ row.servers_missing }}</td>
                  <td data-label="Rate">{{ formatPercent(row.coverage_rate) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>

        <article class="panel table-panel">
          <div class="panel-header">
            <div>
              <p class="eyebrow">Fleet</p>
              <h3>Compliance by location</h3>
            </div>
          </div>

          <div v-if="loading" class="empty-state">Loading fleet analytics...</div>
          <div v-else-if="fleet.by_location.length === 0" class="empty-state">
            <strong>No fleet location data</strong>
            <span>Create servers to populate this section.</span>
          </div>
          <div v-else class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Location</th>
                  <th>Total</th>
                  <th>Compliant</th>
                  <th>Non-compliant</th>
                  <th>Rate</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in fleet.by_location" :key="row.location">
                  <td data-label="Location"><strong>{{ row.location }}</strong></td>
                  <td data-label="Total">{{ row.total_servers }}</td>
                  <td data-label="Compliant">{{ row.compliant_servers }}</td>
                  <td data-label="Non-compliant">{{ row.non_compliant_servers }}</td>
                  <td data-label="Rate">{{ formatPercent(row.compliance_rate) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>
      </section>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Fleet</p>
            <h3>Non-compliant servers</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading fleet analytics...</div>
        <div v-else-if="fleet.non_compliant_server_details.length === 0" class="empty-state">
          <strong>All servers compliant</strong>
          <span>Every tracked server has the required agents installed.</span>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>Location</th>
                <th>Missing agents</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in fleet.non_compliant_server_details" :key="row.server_id">
                <td data-label="Hostname"><strong>{{ row.hostname }}</strong></td>
                <td data-label="Location">{{ row.location }}</td>
                <td data-label="Missing agents">{{ row.missing_agents.join(", ") }}</td>
                <td data-label="Actions">
                  <RouterLink
                    class="button button-secondary"
                    :to="{ name: 'server-detail', params: { id: row.server_id } }"
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
                  <td data-label="Automation">
                    <RouterLink
                      v-if="row.automation_id"
                      :to="{
                        name: 'automation-detail',
                        params: { automationId: row.automation_id },
                      }"
                    >
                      <strong>{{ row.automation }}</strong>
                    </RouterLink>
                    <strong v-else>{{ row.automation }}</strong>
                  </td>
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
import { getAutomationAnalytics, getComplianceAnalytics, getFleetComplianceAnalytics } from "../api/analytics";
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

const fleet = reactive({
  total_servers: 0,
  compliant_servers: 0,
  non_compliant_servers: 0,
  compliance_rate: 0,
  total_required_agents: 0,
  required_agent_coverage: [],
  by_location: [],
  non_compliant_server_details: [],
});

onMounted(loadAnalytics);

async function loadAnalytics() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [automationResponse, complianceResponse, fleetResponse] = await Promise.all([
      getAutomationAnalytics(),
      getComplianceAnalytics(),
      getFleetComplianceAnalytics(),
    ]);

    Object.assign(analytics, automationResponse.data || {});
    Object.assign(compliance, complianceResponse.data || {});
    Object.assign(fleet, fleetResponse.data || {});
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}
</script>
