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

    <section class="content-grid">
      <article class="panel form-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">{{ editingJob ? "Update" : "Create" }}</p>
            <h3>{{ editingJob ? "Update job" : "Create job" }}</h3>
          </div>
        </div>

        <form class="form" @submit.prevent="submitForm">
          <label>
            Automation
            <select v-model="selectedAutomationId" required :disabled="Boolean(editingJob)">
              <option value="" disabled>Select automation</option>
              <option v-for="automation in automations" :key="automation.id" :value="automation.id">
                {{ automation.name }}
              </option>
            </select>
          </label>

          <label>
            Job ID
            <input
              v-model.trim="form.id"
              type="text"
              required
              :disabled="Boolean(editingJob)"
            />
          </label>

          <label>
            Status
            <select v-model="form.status" required>
              <option value="started">started</option>
              <option value="success">success</option>
              <option value="failed">failed</option>
            </select>
          </label>

          <template v-if="!editingJob">
            <label>
              Location
              <input v-model.trim="form.location" type="text" required />
            </label>

            <label>
              URL
              <input v-model.trim="form.url" type="url" required />
            </label>
          </template>

          <button class="button button-full" type="submit" :disabled="loading || !selectedAutomationId">
            {{ editingJob ? "Save status" : "Create job" }}
          </button>
        </form>
      </article>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Automation jobs</p>
            <h3>Job list</h3>
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

        <div v-if="!selectedAutomationId" class="empty-state">
          <strong>Select an automation</strong>
          <span>Jobs are scoped by automation.</span>
        </div>

        <div v-else-if="loading" class="empty-state">Loading jobs...</div>

        <div v-else-if="jobs.length === 0" class="empty-state">
          <strong>No jobs yet</strong>
          <span>Create the first job for this automation.</span>
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
                  <button
                    class="button button-small button-secondary"
                    type="button"
                    @click="editJob(job)"
                  >
                    Update status
                  </button>
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
import { listAutomations } from "../api/automations";
import { createJob, listJobs, updateJob } from "../api/jobs";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";

const automations = ref([]);
const jobs = ref([]);
const selectedAutomationId = ref("");
const editingJob = ref(null);
const loading = ref(false);
const errorMessage = ref("");

const form = reactive({
  id: "",
  status: "started",
  location: "",
  url: "",
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

watch(selectedAutomationId, async () => {
  pagination.page = 1;
  resetJobForm();
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

async function submitForm() {
  if (editingJob.value) {
    await saveJob();
    return;
  }

  await addJob();
}

async function addJob() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createJob(selectedAutomationId.value, {
      id: form.id,
      status: form.status,
      location: form.location,
      url: form.url,
    });

    resetJobForm();
    pagination.page = 1;
    await loadJobsForAutomation();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function editJob(job) {
  editingJob.value = job;
  form.id = job.id;
  form.status = job.status || "started";
  form.location = job.location || "";
  form.url = job.url || "";
  window.scrollTo({ top: 0, behavior: "smooth" });
}

async function saveJob() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await updateJob(selectedAutomationId.value, editingJob.value.id, {
      status: form.status,
    });

    resetJobForm();
    await loadJobsForAutomation();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function resetJobForm() {
  editingJob.value = null;
  form.id = "";
  form.status = "started";
  form.location = "";
  form.url = "";
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