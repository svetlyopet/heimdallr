<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Fleet</p>
        <h2>{{ server?.hostname || "Server" }} jobs</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/servers">All servers</RouterLink>
        <RouterLink
          class="button button-secondary"
          :to="{ name: 'server-detail', params: { serverId } }"
        >
          View server
        </RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">Refresh</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="server" class="stats-grid">
      <article class="stat-card">
        <span>Server ID</span>
        <strong><code>{{ server.id }}</code></strong>
      </article>
      <article class="stat-card">
        <span>Jobs</span>
        <strong>{{ server.relations?.job_count ?? 0 }}</strong>
      </article>
    </section>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Executions</p>
            <h3>Associated jobs</h3>
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

        <div v-if="loading" class="empty-state">Loading jobs...</div>
        <div v-else-if="jobs.length === 0" class="empty-state">
          <strong>No jobs yet</strong>
          <span>No jobs are associated with this server.</span>
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
              <tr v-for="job in jobs" :key="`${job.automation_id}:${job.job_id}`">
                <td><strong>{{ job.job_id }}</strong></td>
                <td>{{ job.automation || "—" }}</td>
                <td><span class="badge">{{ job.provider || "—" }}</span></td>
                <td>
                  <span class="badge" :class="`badge-${job.status}`">{{ job.status || "—" }}</span>
                </td>
                <td>{{ job.location || "—" }}</td>
                <td>
                  <a v-if="job.url" :href="job.url" target="_blank" rel="noreferrer">Open</a>
                  <span v-else>—</span>
                </td>
                <td>
                  <RouterLink
                    class="button button-small button-secondary"
                    :to="{
                      name: 'job-detail',
                      params: {
                        automationId: job.automation_id,
                        jobId: job.job_id,
                      },
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
import { onMounted, reactive, ref } from "vue";
import { RouterLink, useRoute } from "vue-router";
import { getServer, listServerJobs } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";

const route = useRoute();
const serverId = route.params.serverId;

const server = ref(null);
const jobs = ref([]);
const loading = ref(false);
const errorMessage = ref("");

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
    const [serverResponse, jobsResponse] = await Promise.all([
      getServer(serverId),
      listServerJobs(serverId, { page: pagination.page, limit: pagination.limit }),
    ]);

    server.value = serverResponse.data || null;
    jobs.value = jobsResponse.data || [];
    Object.assign(pagination, jobsResponse.pagination || pagination);
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

async function changeLimit() {
  pagination.page = 1;
  await loadData();
}
</script>
