<template>
  <section>
    <BreadcrumbNav :items="breadcrumbItems" />

    <header class="topbar">
      <div>
        <p class="eyebrow">Software</p>
        <h2>{{ release?.application }} · {{ release?.version }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/releases">All releases</RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">Refresh</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="release" class="stats-grid">
      <article class="stat-card">
        <span>Total reports</span>
        <strong>{{ release.compliance?.total_reports || 0 }}</strong>
      </article>
      <article class="stat-card">
        <span>Success rate</span>
        <strong>{{ formatPercent(release.compliance?.success_rate) }}</strong>
      </article>
      <article class="stat-card">
        <span>Failed</span>
        <strong>{{ release.compliance?.failed_reports || 0 }}</strong>
      </article>
      <article class="stat-card">
        <span>In progress</span>
        <strong>{{ release.compliance?.started_reports || 0 }}</strong>
      </article>
    </section>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Matrix</p>
            <h3>Compliance by report type</h3>
          </div>
        </div>

        <div v-if="matrixRows.length === 0" class="empty-state">
          <strong>No reports yet</strong>
          <span>CI pipelines push SAST, DAST, SBOM, and coverage reports here.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Type</th>
                <th v-for="status in reportStatuses" :key="status">{{ status }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in matrixRows" :key="row.type">
                <td><strong>{{ row.type }}</strong></td>
                <td v-for="status in reportStatuses" :key="`${row.type}-${status}`">
                  <span v-if="row.counts[status]" class="badge" :class="`badge-${status}`">
                    {{ row.counts[status] }}
                  </span>
                  <span v-else>—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Reports</p>
            <h3>All reports for this release</h3>
          </div>

          <RouterLink
            class="button button-secondary"
            :to="{
              name: 'reports',
              query: { application_id: applicationId, release_id: releaseId },
            }"
          >
            Open inbox
          </RouterLink>
        </div>

        <div v-if="loading" class="empty-state">Loading reports...</div>
        <div v-else-if="reports.length === 0" class="empty-state">
          <strong>No reports yet</strong>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Type</th>
                <th>Status</th>
                <th>Location</th>
                <th>Created</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="report in reports" :key="report.id">
                <td data-label="ID"><code>{{ report.id }}</code></td>
                <td data-label="Type">{{ report.type }}</td>
                <td data-label="Status">
                  <span class="badge" :class="`badge-${report.status}`">{{ report.status }}</span>
                </td>
                <td data-label="Location">{{ report.location || "—" }}</td>
                <td data-label="Created">{{ formatDateTime(report.created_at) }}</td>
                <td data-label="Actions">
                  <RouterLink
                    class="button button-secondary"
                    :to="{
                      name: 'report-detail',
                      params: { id: applicationId, releaseId, reportId: report.id },
                    }"
                  >
                    View
                  </RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <PaginationControls
          :page="pagination.page"
          :total-pages="pagination.total_pages"
          :loading="loading"
          @previous="previousPage"
          @next="nextPage"
        />
      </article>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { RouterLink, useRoute } from "vue-router";
import { listReports } from "../api/reports";
import { getRelease } from "../api/releases";
import AppAlert from "../components/AppAlert.vue";
import BreadcrumbNav from "../components/BreadcrumbNav.vue";
import PaginationControls from "../components/PaginationControls.vue";
import { formatDateTime, formatPercent } from "../utils/format";

const route = useRoute();
const applicationId = route.params.id;
const releaseId = route.params.releaseId;

const release = ref(null);
const reports = ref([]);
const loading = ref(false);
const errorMessage = ref("");
const reportStatuses = ["success", "failed"];

const pagination = reactive({
  page: 1,
  limit: 20,
  total: 0,
  total_pages: 0,
});

const breadcrumbItems = computed(() => [
  { label: "Applications", to: { name: "applications" } },
  {
    label: release.value?.application || "Application",
    to: { name: "application-detail", params: { id: applicationId } },
  },
  { label: "Releases", to: { name: "releases" } },
  { label: release.value?.version || "Release" },
]);

const matrixRows = computed(() => {
  const byType = {};

  for (const entry of release.value?.compliance?.by_type || []) {
    if (!byType[entry.type]) {
      byType[entry.type] = { type: entry.type, counts: {} };
    }

    byType[entry.type].counts[entry.status] = entry.count;
  }

  return Object.values(byType);
});

onMounted(loadData);

async function loadData() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [releaseResponse, reportResponse] = await Promise.all([
      getRelease(applicationId, releaseId),
      listReports(applicationId, releaseId, { page: pagination.page, limit: pagination.limit }),
    ]);

    release.value = releaseResponse.data || null;
    reports.value = reportResponse.data || [];
    Object.assign(pagination, reportResponse.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadData();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadData();
}
</script>
