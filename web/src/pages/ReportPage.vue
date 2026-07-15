<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Software</p>
        <h2>Reports</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadReports">
          Refresh
        </button>
      </div>
    </header>

    <section class="stats-grid">
      <article class="stat-card">
        <span>Total reports</span>
        <strong>{{ pagination.total }}</strong>
      </article>
      <article class="stat-card">
        <span>Current page</span>
        <strong>{{ pagination.page }}</strong>
      </article>
      <article class="stat-card">
        <span>Total pages</span>
        <strong>{{ pagination.total_pages }}</strong>
      </article>
      <article class="stat-card">
        <span>Page size</span>
        <strong>{{ pagination.limit }}</strong>
      </article>
    </section>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">CI reports</p>
            <h3>Report inbox</h3>
          </div>

          <div class="page-size">
            <label>
              Application
              <select v-model="filters.application_id" @change="onApplicationChange">
                <option value="">All applications</option>
                <option v-for="application in applications" :key="application.id" :value="application.id">
                  {{ application.name }}
                </option>
              </select>
            </label>

            <label>
              Release
              <select v-model="filters.release_id" @change="resetPageAndLoad">
                <option value="">All releases</option>
                <option v-for="release in releases" :key="release.id" :value="release.id">
                  {{ release.version }}
                </option>
              </select>
            </label>

            <label>
              Status
              <select v-model="filters.status" @change="resetPageAndLoad">
                <option value="">All statuses</option>
                <option value="failed">Failed</option>
                <option value="success">Success</option>
              </select>
            </label>

            <label>
              Type
              <select v-model="filters.type" @change="resetPageAndLoad">
                <option value="">All types</option>
                <option value="sast">SAST</option>
                <option value="dast">DAST</option>
                <option value="sbom">SBOM</option>
                <option value="code_coverage">Code coverage</option>
                <option value="custom">Custom</option>
              </select>
            </label>

            <label>
              Limit
              <select v-model.number="pagination.limit" @change="resetPageAndLoad">
                <option :value="5">5</option>
                <option :value="10">10</option>
                <option :value="20">20</option>
              </select>
            </label>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading reports...</div>

        <div v-else-if="reports.length === 0" class="empty-state">
          <strong>No reports found</strong>
          <span>Adjust filters or push CI reports to populate this inbox.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Application</th>
                <th>Version</th>
                <th>Type</th>
                <th>Status</th>
                <th>Location</th>
                <th>Created</th>
                <th>URL</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="report in reports" :key="`${report.application_id}:${report.release_id}:${report.id}`">
                <td data-label="ID"><code>{{ report.id }}</code></td>
                <td data-label="Application">{{ report.application }}</td>
                <td data-label="Version">{{ report.version }}</td>
                <td data-label="Type">{{ report.type }}</td>
                <td data-label="Status">
                  <span class="badge" :class="`badge-${report.status}`">{{ report.status }}</span>
                </td>
                <td data-label="Location">{{ report.location || "—" }}</td>
                <td data-label="Created">{{ formatDateTime(report.created_at) }}</td>
                <td data-label="URL">
                  <a v-if="report.url" :href="report.url" target="_blank" rel="noreferrer">Open</a>
                  <span v-else>—</span>
                </td>
                <td data-label="Actions">
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{
                        name: 'report-detail',
                        params: {
                          id: report.application_id,
                          releaseId: report.release_id,
                          reportId: report.id,
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
          :loading="loading"
          @previous="previousPage"
          @next="nextPage"
        />
      </article>
    </section>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { listApplications } from "../api/applications";
import { listAllReports } from "../api/reports";
import { listReleases } from "../api/releases";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import { formatDateTime } from "../utils/format";

const route = useRoute();
const router = useRouter();

const applications = ref([]);
const releases = ref([]);
const reports = ref([]);
const loading = ref(false);
const errorMessage = ref("");

const filters = reactive({
  application_id: route.query.application_id || "",
  release_id: route.query.release_id || "",
  status: route.query.status || "",
  type: route.query.type || "",
});

const pagination = reactive({
  page: Number(route.query.page) || 1,
  limit: Number(route.query.limit) || 10,
  total: 0,
  total_pages: 0,
});

onMounted(async () => {
  await loadApplications();
  if (filters.application_id) {
    await loadReleasesForApplication();
  }
  await loadReports();
});

watch(
  () => ({ ...filters, page: pagination.page, limit: pagination.limit }),
  (query) => {
    router.replace({
      name: "reports",
      query: {
        application_id: query.application_id || undefined,
        release_id: query.release_id || undefined,
        status: query.status || undefined,
        type: query.type || undefined,
        page: query.page > 1 ? String(query.page) : undefined,
        limit: query.limit !== 10 ? String(query.limit) : undefined,
      },
    });
  },
);

async function loadApplications() {
  try {
    const response = await listApplications({ page: 1, limit: 100 });
    applications.value = response.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function loadReleasesForApplication() {
  if (!filters.application_id) {
    releases.value = [];
    return;
  }

  try {
    const response = await listReleases(filters.application_id, { page: 1, limit: 100 });
    releases.value = response.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function onApplicationChange() {
  filters.release_id = "";
  pagination.page = 1;
  await loadReleasesForApplication();
  await loadReports();
}

async function resetPageAndLoad() {
  pagination.page = 1;
  await loadReports();
}

async function loadReports() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const params = {
      page: pagination.page,
      limit: pagination.limit,
    };

    if (filters.application_id) params.application_id = filters.application_id;
    if (filters.release_id) params.release_id = filters.release_id;
    if (filters.status) params.status = filters.status;
    if (filters.type) params.type = filters.type;

    const response = await listAllReports(params);
    reports.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadReports();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadReports();
}
</script>
