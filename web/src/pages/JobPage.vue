<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Executions</p>
        <h2>Jobs</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadAll">
          Refresh
        </button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Automation jobs</p>
            <h3>Job list</h3>
          </div>

          <div class="page-size">
            <label>
              Automation
              <select v-model="selectedAutomationId">
                <option value="" disabled>Select automation</option>
                <option v-for="automation in automations" :key="automation.id" :value="automation.id">
                  {{ automation.name }}
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

        <div v-if="!selectedAutomationId" class="empty-state">
          <strong>Select an automation</strong>
          <span>Jobs are scoped by automation.</span>
        </div>

        <div v-else-if="loading" class="empty-state">Loading jobs...</div>

        <div v-else-if="jobs.length === 0" class="empty-state">
          <strong>No jobs yet</strong>
          <span>No jobs were found for this automation.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Job ID</th>
                <th>Automation</th>
                <th>Provider</th>
                <th>Status</th>
                <th>Location</th>
                <th>URL</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="job in jobs" :key="`${selectedAutomationId}:${job.id}`">
                <td data-label="Job ID"><strong>{{ job.id }}</strong></td>
                <td data-label="Automation">{{ job.automation }}</td>
                <td data-label="Provider"><span class="badge">{{ job.provider }}</span></td>
                <td data-label="Status"><span class="badge" :class="`badge-${job.status}`">{{ job.status }}</span></td>
                <td data-label="Location">{{ job.location }}</td>
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
                          automationId: selectedAutomationId,
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
import { listAutomations } from "../api/automations";
import { listJobs } from "../api/jobs";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";

const automations = ref([]);
const jobs = ref([]);
const selectedAutomationId = ref("");
const loading = ref(false);
const errorMessage = ref("");

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

watch(selectedAutomationId, async () => {
  pagination.page = 1;
  await loadJobsForAutomation();
});

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const automationResponse = await listAutomations({ page: 1, limit: 100 });
    automations.value = automationResponse.data || [];

    if (!selectedAutomationId.value && automations.value.length > 0) {
      selectedAutomationId.value = automations.value[0].id;
    }

    await loadJobsForAutomation();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function loadJobsForAutomation() {
  if (!selectedAutomationId.value) {
    jobs.value = [];
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
    const response = await listJobs(selectedAutomationId.value, {
      page: pagination.page,
      limit: pagination.limit,
    });

    jobs.value = response.data || [];
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
  await loadJobsForAutomation();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadJobsForAutomation();
}

async function changeLimit() {
  pagination.page = 1;
  await loadJobsForAutomation();
}
</script>