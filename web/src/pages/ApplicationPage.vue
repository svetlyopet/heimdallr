<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Software</p>
        <h2>Applications</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadApplications">
          Refresh
        </button>
        <button class="button" type="button" @click="openCreateDialog">
          Create application
        </button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New application"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitApplication">
        <label>
          Name
          <input v-model.trim="form.name" type="text" required minlength="2" maxlength="100" />
        </label>

        <label>
          Description
          <textarea v-model.trim="form.description" rows="3" maxlength="1000"></textarea>
        </label>

        <label>
          Repository URL
          <input v-model.trim="form.repository_url" type="url" />
        </label>

        <button class="button button-full" type="submit" :disabled="loading">
          Create application
        </button>
      </form>
    </FormDialog>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>Application list</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading applications...</div>

        <div v-else-if="applications.length === 0" class="empty-state">
          <strong>No applications yet</strong>
          <span>Create your first application to track software releases and reports.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>ID</th>
                <th>Repository</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="application in applications" :key="application.id">
                <td>
                  <div class="name-cell">
                    <span class="avatar">{{ getInitial(application.name) }}</span>
                    <strong>{{ application.name }}</strong>
                  </div>
                </td>
                <td><code>{{ application.id }}</code></td>
                <td>
                  <a
                    v-if="application.repository_url"
                    :href="application.repository_url"
                    target="_blank"
                    rel="noreferrer"
                  >
                    Open
                  </a>
                  <span v-else>—</span>
                </td>
                <td>
                  <RouterLink
                    class="button button-secondary"
                    :to="{ name: 'application-detail', params: { id: application.id } }"
                  >
                    View releases
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
import { RouterLink } from "vue-router";
import { createApplication, listApplications } from "../api/applications";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";
import { getInitial } from "../utils/format";

const applications = ref([]);
const loading = ref(false);
const errorMessage = ref("");
const showCreateDialog = ref(false);

const form = reactive({
  name: "",
  description: "",
  repository_url: "",
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadApplications);

async function loadApplications() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listApplications({
      page: pagination.page,
      limit: pagination.limit,
    });

    applications.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function submitApplication() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createApplication({
      name: form.name,
      description: form.description,
      repository_url: form.repository_url,
    });

    resetForm();
    showCreateDialog.value = false;
    pagination.page = 1;
    await loadApplications();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function openCreateDialog() {
  resetForm();
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  resetForm();
  showCreateDialog.value = false;
}

function resetForm() {
  form.name = "";
  form.description = "";
  form.repository_url = "";
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadApplications();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadApplications();
}
</script>
