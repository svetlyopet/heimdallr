<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Software</p>
        <h2>Releases</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadAll">
          Refresh
        </button>
      </div>
    </header>

    <section class="stats-grid">
      <article class="stat-card">
        <span>Total releases</span>
        <strong>{{ pagination.total }}</strong>
      </article>
      <article class="stat-card">
        <span>Applications</span>
        <strong>{{ applications.length }}</strong>
      </article>
      <article class="stat-card">
        <span>Failed reports</span>
        <strong>{{ compliance.failed_reports }}</strong>
      </article>
      <article class="stat-card">
        <span>Success rate</span>
        <strong>{{ formatPercent(compliance.success_rate) }}</strong>
      </article>
    </section>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Version history</p>
            <h3>Release list</h3>
          </div>

          <div class="page-size">
            <label>
              Application
              <select v-model="selectedApplicationId">
                <option value="" disabled>Select application</option>
                <option v-for="application in applications" :key="application.id" :value="application.id">
                  {{ application.name }}
                </option>
              </select>
            </label>

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

        <div v-if="!selectedApplicationId" class="empty-state">
          <strong>Select an application</strong>
          <span>Releases are scoped by application.</span>
        </div>

        <div v-else-if="loading" class="empty-state">Loading releases...</div>

        <div v-else-if="releases.length === 0" class="empty-state">
          <strong>No releases yet</strong>
          <span>Create a release manually or push from CI.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Version</th>
                <th>Branch</th>
                <th>Commit</th>
                <th>Reports</th>
                <th>Failed</th>
                <th>Rate</th>
                <th>Created</th>
                <th>Pipeline</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="release in releases" :key="release.id">
                <td data-label="Version"><strong>{{ release.version }}</strong></td>
                <td data-label="Branch">{{ release.branch || "—" }}</td>
                <td data-label="Commit">
                  <code v-if="release.commit_sha">{{ release.commit_sha.slice(0, 8) }}</code>
                  <span v-else>—</span>
                </td>
                <td data-label="Reports">{{ release.compliance?.total_reports || 0 }}</td>
                <td data-label="Failed">{{ release.compliance?.failed_reports || 0 }}</td>
                <td data-label="Rate">{{ formatPercent(release.compliance?.success_rate) }}</td>
                <td data-label="Created">{{ formatDateTime(release.created_at) }}</td>
                <td data-label="Pipeline">
                  <a v-if="release.pipeline_url" :href="release.pipeline_url" target="_blank" rel="noreferrer">Open</a>
                  <span v-else>—</span>
                </td>
                <td data-label="Actions">
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{
                        name: 'release-detail',
                        params: { id: selectedApplicationId, releaseId: release.id },
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
import { RouterLink } from "vue-router";
import { getComplianceAnalytics } from "../api/analytics";
import { listApplications } from "../api/applications";
import { listReleases } from "../api/releases";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import { formatDateTime, formatPercent } from "../utils/format";

const applications = ref([]);
const releases = ref([]);
const selectedApplicationId = ref("");
const loading = ref(false);
const errorMessage = ref("");

const compliance = reactive({
  failed_reports: 0,
  success_rate: 0,
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

watch(selectedApplicationId, async () => {
  pagination.page = 1;
  await loadReleasesForApplication();
});

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [applicationResponse, complianceResponse] = await Promise.all([
      listApplications({ page: 1, limit: 100 }),
      getComplianceAnalytics(),
    ]);

    applications.value = applicationResponse.data || [];
    Object.assign(compliance, complianceResponse.data || {});

    if (!selectedApplicationId.value && applications.value.length > 0) {
      selectedApplicationId.value = applications.value[0].id;
    }

    await loadReleasesForApplication();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function loadReleasesForApplication() {
  if (!selectedApplicationId.value) {
    releases.value = [];
    Object.assign(pagination, {
      page: 1,
      limit: pagination.limit,
      total: 0,
      total_pages: 0,
    });
    return;
  }

  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listReleases(selectedApplicationId.value, {
      page: pagination.page,
      limit: pagination.limit,
    });

    releases.value = response.data || [];
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
  await loadReleasesForApplication();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadReleasesForApplication();
}

async function changeLimit() {
  pagination.page = 1;
  await loadReleasesForApplication();
}
</script>
