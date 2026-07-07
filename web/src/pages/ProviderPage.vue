<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Registry</p>
        <h2>Providers</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadProviders">
          Refresh
        </button>
        <button class="button" type="button" @click="openCreateDialog">
          Create provider
        </button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New provider"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitProvider">
        <label>
          Name
          <input v-model.trim="form.name" type="text" required minlength="2" maxlength="100" />
        </label>

        <label>
          URL
          <input v-model.trim="form.url" type="url" required />
        </label>

        <button class="button button-full" type="submit" :disabled="loading">
          Create provider
        </button>
      </form>
    </FormDialog>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>Provider list</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading providers...</div>

        <div v-else-if="providers.length === 0" class="empty-state">
          <strong>No providers yet</strong>
          <span>Create your first provider.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>ID</th>
                <th>URL</th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="provider in providers" :key="provider.id">
                <td>
                  <div class="name-cell">
                    <span class="avatar">{{ getInitial(provider.name) }}</span>
                    <strong>{{ provider.name }}</strong>
                  </div>
                </td>
                <td><code>{{ provider.id }}</code></td>
                <td>
                  <a :href="provider.url" target="_blank" rel="noreferrer">Open</a>
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
import { createProvider, listProviders } from "../api/providers";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";
import { getInitial } from "../utils/format";

const providers = ref([]);
const loading = ref(false);
const errorMessage = ref("");
const showCreateDialog = ref(false);

const form = reactive({
  name: "",
  url: "",
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadProviders);

async function loadProviders() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listProviders({
      page: pagination.page,
      limit: pagination.limit,
    });

    providers.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function submitProvider() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createProvider({
      name: form.name,
      url: form.url,
    });

    resetProviderForm();
    showCreateDialog.value = false;
    pagination.page = 1;
    await loadProviders();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function openCreateDialog() {
  resetProviderForm();
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  resetProviderForm();
  showCreateDialog.value = false;
}

function resetProviderForm() {
  form.name = "";
  form.url = "";
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadProviders();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadProviders();
}
</script>