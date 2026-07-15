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
              <select v-model="filters.automation_id" @change="resetPageAndLoad">
                <option value="">All automations</option>
                <option v-for="automation in automations" :key="automation.id" :value="automation.id">
                  {{ automation.name }}
                </option>
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

        <div v-if="loading" class="empty-state">Loading jobs...</div>

        <div v-else-if="jobs.length === 0" class="empty-state">
          <strong>No jobs found</strong>
          <span>Adjust filters or run automations to populate this list.</span>
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
                <th>Created</th>
                <th>URL</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="job in jobs" :key="`${job.automation_id}:${job.id}`">
                <td data-label="Job ID"><strong>{{ job.id }}</strong></td>
                <td data-label="Automation">{{ job.automation }}</td>
                <td data-label="Provider"><span class="badge">{{ job.provider }}</span></td>
                <td data-label="Status"><span class="badge" :class="`badge-${job.status}`">{{ job.status }}</span></td>
                <td data-label="Location">{{ job.location }}</td>
                <td data-label="Created">{{ formatDateTime(job.created_at) }}</td>
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
                          automationId: job.automation_id,
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
import { RouterLink, useRoute, useRouter } from "vue-router";
import { listAutomations } from "../api/automations";
import { listAllJobs } from "../api/jobs";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";
import { formatDateTime } from "../utils/format";

const route = useRoute();
const router = useRouter();

const automations = ref([]);
const jobs = ref([]);
const loading = ref(false);
const errorMessage = ref("");

const filters = reactive({
  automation_id: route.query.automation_id || "",
});

const pagination = reactive({
  page: Number(route.query.page) || 1,
  limit: Number(route.query.limit) || 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

watch(
  () => ({ ...filters, page: pagination.page, limit: pagination.limit }),
  (query) => {
    router.replace({
      name: "jobs",
      query: {
        automation_id: query.automation_id || undefined,
        page: query.page > 1 ? String(query.page) : undefined,
        limit: query.limit !== 10 ? String(query.limit) : undefined,
      },
    });
  },
);

async function loadAll() {
  errorMessage.value = "";

  try {
    const automationResponse = await listAutomations({ page: 1, limit: 100 });
    automations.value = automationResponse.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }

  await loadJobs();
}

async function loadJobs() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const params = {
      page: pagination.page,
      limit: pagination.limit,
    };

    if (filters.automation_id) {
      params.automation_id = filters.automation_id;
    }

    const response = await listAllJobs(params);

    jobs.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function resetPageAndLoad() {
  pagination.page = 1;
  await loadJobs();
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
</script>
