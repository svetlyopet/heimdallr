<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Operations</p>
        <h2>{{ automation?.name || "Automation" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/automations">
          Back to automations
        </RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">
          Refresh
        </button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="loading" class="empty-state">Loading automation...</section>

    <section v-else-if="!automation" class="empty-state">
      <strong>Automation not found</strong>
      <span>The selected automation could not be loaded.</span>
    </section>

    <template v-else>
      <section class="stats-grid">
        <article class="stat-card">
          <span>Total jobs</span>
          <strong>{{ analytics.total_jobs }}</strong>
        </article>
        <article class="stat-card">
          <span>Success rate</span>
          <strong>{{ formatPercent(analytics.success_rate) }}</strong>
        </article>
        <article class="stat-card">
          <span>Successful jobs</span>
          <strong>{{ analytics.successful_jobs }}</strong>
        </article>
        <article class="stat-card">
          <span>Failed jobs</span>
          <strong>{{ analytics.failed_jobs }}</strong>
        </article>
      </section>

      <section class="dashboard-grid">
        <article class="panel detail-panel panel-span-full">
          <div class="panel-header">
            <div>
              <p class="eyebrow">Overview</p>
              <h3>{{ automation.name }}</h3>
            </div>
          </div>

          <dl class="detail-grid">
            <div>
              <dt>Automation ID</dt>
              <dd><code>{{ automation.id }}</code></dd>
            </div>

            <div>
              <dt>Provider</dt>
              <dd><span class="badge">{{ automation.provider || "—" }}</span></dd>
            </div>

            <div>
              <dt>URL</dt>
              <dd>
                <a v-if="automation.url" :href="automation.url" target="_blank" rel="noreferrer">
                  {{ automation.url }}
                </a>
                <span v-else>—</span>
              </dd>
            </div>

            <div>
              <dt>Cost savings</dt>
              <dd>{{ formatNumber(automation.cost_savings) }}</dd>
            </div>
          </dl>
        </article>

        <article class="panel table-panel">
          <div class="panel-header">
            <div>
              <p class="eyebrow">Locations</p>
              <h3>Success rate by location</h3>
            </div>
          </div>

          <div v-if="analytics.by_location.length === 0" class="empty-state">
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
                  <th>Rate</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in analytics.by_location" :key="row.location">
                  <td data-label="Location"><strong>{{ row.location }}</strong></td>
                  <td data-label="Total">{{ row.total_jobs }}</td>
                  <td data-label="Success">{{ row.successful_jobs }}</td>
                  <td data-label="Failed">{{ row.failed_jobs }}</td>
                  <td data-label="Rate">{{ formatPercent(row.success_rate) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>

        <article class="panel table-panel">
          <div class="panel-header">
            <div>
              <p class="eyebrow">Executions</p>
              <h3>Jobs</h3>
            </div>

            <div class="page-size">
              <label>
                Limit
                <select v-model.number="pagination.limit" @change="changeLimit">
                  <option :value="5">5</option>
                  <option :value="10">10</option>
                  <option :value="20">20</option>
                </select>
              </label>
            </div>
          </div>

          <div v-if="jobsLoading" class="empty-state">Loading jobs...</div>
          <div v-else-if="jobs.length === 0" class="empty-state">
            <strong>No jobs yet</strong>
            <span>No jobs were found for this automation.</span>
          </div>
          <div v-else class="table-wrapper">
            <table>
              <thead>
                <tr>
                  <th>Job ID</th>
                  <th>Status</th>
                  <th>Location</th>
                  <th>URL</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="job in jobs" :key="job.id">
                  <td data-label="Job ID"><strong>{{ job.id }}</strong></td>
                  <td data-label="Status">
                    <span class="badge" :class="`badge-${job.status}`">{{ job.status }}</span>
                  </td>
                  <td data-label="Location">{{ job.location || "—" }}</td>
                  <td data-label="URL">
                    <a v-if="job.url" :href="job.url" target="_blank" rel="noreferrer">Open</a>
                    <span v-else>—</span>
                  </td>
                  <td data-label="Actions">
                    <div class="row-actions">
                      <RouterLink
                        class="button button-small button-secondary"
                        :to="{
                          name: 'job-detail',
                          params: {
                            automationId,
                            jobId: job.id,
                          },
                        }"
                      >
                        View
                      </RouterLink>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <PaginationControls
            :page="pagination.page"
            :total-pages="pagination.total_pages"
            :loading="jobsLoading"
            @previous="previousPage"
            @next="nextPage"
          />
        </article>
      </section>
    </template>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import "../stylesheets/detail.css";
import { RouterLink, useRoute } from "vue-router";
import { getAutomationAnalyticsById } from "../api/analytics";
import { getAutomation } from "../api/automations";
import { listJobs } from "../api/jobs";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import { formatNumber, formatPercent } from "../utils/format";

const route = useRoute();
const automationId = route.params.automationId;

const automation = ref(null);
const jobs = ref([]);
const loading = ref(false);
const jobsLoading = ref(false);
const errorMessage = ref("");

const analytics = reactive({
  total_jobs: 0,
  successful_jobs: 0,
  failed_jobs: 0,
  success_rate: 0,
  by_location: [],
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadData);

async function loadData() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [automationResponse, analyticsResponse, jobsResponse] = await Promise.all([
      getAutomation(automationId),
      getAutomationAnalyticsById(automationId),
      listJobs(automationId, { page: pagination.page, limit: pagination.limit }),
    ]);

    automation.value = automationResponse.data || null;
    Object.assign(analytics, analyticsResponse.data || {});
    jobs.value = jobsResponse.data || [];
    Object.assign(pagination, jobsResponse.pagination || pagination);
  } catch (error) {
    automation.value = null;
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function loadJobs() {
  jobsLoading.value = true;
  errorMessage.value = "";

  try {
    const response = await listJobs(automationId, {
      page: pagination.page,
      limit: pagination.limit,
    });

    jobs.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    jobsLoading.value = false;
  }
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadJobs();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadJobs();
}

async function changeLimit() {
  pagination.page = 1;
  await loadJobs();
}
</script>
