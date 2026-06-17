<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Dashboard</p>
        <h2>Automations</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadAll">
          Refresh
        </button>
        <button class="button" type="button" @click="resetForm">New automation</button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section class="content-grid">
      <article class="panel form-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">{{ editingAutomation ? "Update" : "Create" }}</p>
            <h3>{{ editingAutomation ? "Edit automation" : "Create automation" }}</h3>
          </div>
        </div>

        <form class="form" @submit.prevent="submitForm">
          <label>
            Name
            <input
              v-model.trim="form.name"
              type="text"
              required
              minlength="2"
              maxlength="100"
              :disabled="Boolean(editingAutomation)"
            />
          </label>

          <label>
            Provider
            <select v-model="form.provider_id" required :disabled="Boolean(editingAutomation)">
              <option value="" disabled>Select provider</option>
              <option v-for="provider in providers" :key="provider.id" :value="provider.id">
                {{ provider.name }}
              </option>
            </select>
          </label>

          <label>
            URL
            <input v-model.trim="form.url" type="url" />
          </label>

          <label>
            Cost savings
            <input v-model.number="form.cost_savings" type="number" min="0" step="0.01" />
          </label>

          <button class="button button-full" type="submit" :disabled="loading">
            {{ editingAutomation ? "Save changes" : "Create automation" }}
          </button>
        </form>
      </article>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>Automation list</h3>
          </div>

          <div class="page-size">
            <label>
              Limit
              <select v-model.number="pagination.limit" @change="changeLimit">
                <option :value="5">5</option>
                <option :value="10">10</option>
                <option :value="20">20</option>
                <option :value="50">50</option>
              </select>
            </label>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading automations...</div>

        <div v-else-if="automations.length === 0" class="empty-state">
          <strong>No automations yet</strong>
          <span>Create your first automation.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Provider</th>
                <th>URL</th>
                <th>Cost savings</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="automation in automations" :key="automation.id">
                <td data-label="Name">
                  <div class="name-cell">
                    <span class="avatar">{{ getInitial(automation.name) }}</span>
                    <strong>{{ automation.name }}</strong>
                  </div>
                </td>

                <td data-label="Provider">
                  <span class="badge">{{ automation.provider }}</span>
                </td>

                <td data-label="URL">
                  <a v-if="automation.url" :href="automation.url" target="_blank" rel="noreferrer">
                    Open
                  </a>
                  <span v-else>—</span>
                </td>

                <td data-label="Cost savings">{{ formatNumber(automation.cost_savings) }}</td>

                <td data-label="Actions">
                  <div class="row-actions">
                    <button
                      class="button button-small button-secondary"
                      type="button"
                      @click="editAutomation(automation)"
                    >
                      Edit
                    </button>

                    <button
                      class="button button-small button-danger"
                      type="button"
                      @click="removeAutomation(automation)"
                    >
                      Delete
                    </button>
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
import { onMounted, reactive, ref } from "vue";
import {
  createAutomation,
  deleteAutomation,
  listAutomations,
  updateAutomation,
} from "../api/automations";
import { listProviders } from "../api/providers";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";
import { formatNumber, getInitial } from "../utils/format";

const automations = ref([]);
const providers = ref([]);
const loading = ref(false);
const errorMessage = ref("");
const editingAutomation = ref(null);

const form = reactive({
  name: "",
  provider_id: "",
  url: "",
  cost_savings: 0,
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [providerResponse, automationResponse] = await Promise.all([
      listProviders({ page: 1, limit: 100 }),
      listAutomations({ page: pagination.page, limit: pagination.limit }),
    ]);

    providers.value = providerResponse.data || [];
    automations.value = automationResponse.data || [];
    Object.assign(pagination, automationResponse.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function submitForm() {
  if (editingAutomation.value) {
    await saveAutomation();
    return;
  }

  await addAutomation();
}

async function addAutomation() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createAutomation({
      name: form.name,
      provider_id: form.provider_id,
      url: form.url,
      cost_savings: Number(form.cost_savings || 0),
    });

    resetForm();
    pagination.page = 1;
    await loadAll();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function editAutomation(automation) {
  editingAutomation.value = automation;
  form.name = automation.name || "";
  form.provider_id = automation.provider_id || "";
  form.url = automation.url || "";
  form.cost_savings = Number(automation.cost_savings || 0);

  window.scrollTo({ top: 0, behavior: "smooth" });
}

async function saveAutomation() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await updateAutomation(editingAutomation.value.id, {
      url: form.url,
      cost_savings: Number(form.cost_savings || 0),
    });

    resetForm();
    await loadAll();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function removeAutomation(automation) {
  if (!window.confirm(`Delete "${automation.name}"?`)) return;

  loading.value = true;
  errorMessage.value = "";

  try {
    await deleteAutomation(automation.id);

    if (automations.value.length === 1 && pagination.page > 1) {
      pagination.page -= 1;
    }

    await loadAll();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function resetForm() {
  editingAutomation.value = null;
  form.name = "";
  form.provider_id = "";
  form.url = "";
  form.cost_savings = 0;
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadAll();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadAll();
}

async function changeLimit() {
  pagination.page = 1;
  await loadAll();
}
</script>