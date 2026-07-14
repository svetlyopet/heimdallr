<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Fleet</p>
        <h2>Servers</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadServers">
          Refresh
        </button>
        <button class="button" type="button" @click="openCreateDialog">
          Create server
        </button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New server"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitServer">
        <label>
          Hostname
          <input v-model.trim="form.hostname" type="text" required minlength="1" maxlength="255" />
        </label>

        <label>
          Operating system
          <input v-model.trim="form.operating_system" type="text" maxlength="255" />
        </label>

        <label>
          Hypervisor
          <input v-model.trim="form.hypervisor" type="text" maxlength="255" />
        </label>

        <label>
          Location
          <input v-model.trim="form.location" type="text" maxlength="255" />
        </label>

        <label>
          Metadata
          <textarea
            v-model="form.metadata"
            rows="5"
            spellcheck="false"
            placeholder="{}"
          ></textarea>
        </label>
        <p v-if="metadataError" class="form-error">{{ metadataError }}</p>

        <button class="button button-full" type="submit" :disabled="submitting">
          Create server
        </button>
      </form>
    </FormDialog>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>Server list</h3>
          </div>

          <div class="page-size">
            <label>
              Agent
              <select v-model="selectedAgentId" @change="onFilterChange">
                <option value="">All agents</option>
                <option v-for="agent in agentOptions" :key="agent.id" :value="agent.id">
                  {{ agent.name }}
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

        <div v-if="loading" class="empty-state">Loading servers...</div>

        <div v-else-if="servers.length === 0" class="empty-state">
          <strong>No servers yet</strong>
          <span>Create your first server to track security agents.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>ID</th>
                <th>Location</th>
                <th>OS</th>
                <th>Hypervisor</th>
                <th>Agents</th>
                <th>Jobs</th>
                <th>Releases</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="server in servers" :key="server.id">
                <td>
                  <div class="name-cell">
                    <span class="avatar">{{ getInitial(server.hostname) }}</span>
                    <strong>{{ server.hostname }}</strong>
                  </div>
                </td>
                <td><code>{{ server.id }}</code></td>
                <td>{{ server.location || "—" }}</td>
                <td>{{ server.operating_system || "—" }}</td>
                <td>{{ server.hypervisor || "—" }}</td>
                <td>{{ server.relations?.agent_count ?? 0 }}</td>
                <td>{{ server.relations?.job_count ?? 0 }}</td>
                <td>{{ server.relations?.release_count ?? 0 }}</td>
                <td data-label="Actions">
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{ name: 'server-detail', params: { serverId: server.id } }"
                    >
                      View server
                    </RouterLink>
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{ name: 'server-jobs', params: { serverId: server.id } }"
                    >
                      View jobs
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
import { onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import "../stylesheets/dialog.css";
import { RouterLink } from "vue-router";
import { listAgents } from "../api/agents";
import { createServer, listServers } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";
import { getInitial } from "../utils/format";

const route = useRoute();
const router = useRouter();

const servers = ref([]);
const agentOptions = ref([]);
const selectedAgentId = ref("");
const loading = ref(false);
const submitting = ref(false);
const errorMessage = ref("");
const metadataError = ref("");
const showCreateDialog = ref(false);

const form = reactive({
  hostname: "",
  operating_system: "",
  hypervisor: "",
  location: "",
  metadata: "{}",
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(async () => {
  selectedAgentId.value = route.query.agentId || "";
  await loadAgentOptions();
  await loadServers();
});

async function loadAgentOptions() {
  try {
    const response = await listAgents({ page: 1, limit: 100 });
    agentOptions.value = response.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function loadServers() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listServers({
      page: pagination.page,
      limit: pagination.limit,
      agentId: selectedAgentId.value || undefined,
    });

    servers.value = response.data || [];
    Object.assign(pagination, response.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function onFilterChange() {
  pagination.page = 1;
  await router.replace({
    query: {
      agentId: selectedAgentId.value || undefined,
    },
  });
  await loadServers();
}

async function changeLimit() {
  pagination.page = 1;
  await loadServers();
}

async function submitServer() {
  metadataError.value = "";
  const parsedMetadata = parseMetadata(form.metadata);

  if (parsedMetadata.error) {
    metadataError.value = parsedMetadata.error;
    return;
  }

  submitting.value = true;
  errorMessage.value = "";

  try {
    await createServer({
      hostname: form.hostname,
      operating_system: form.operating_system,
      hypervisor: form.hypervisor,
      location: form.location,
      metadata: parsedMetadata.value,
    });

    resetForm();
    showCreateDialog.value = false;
    pagination.page = 1;
    await loadServers();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    submitting.value = false;
  }
}

function parseMetadata(raw) {
  const trimmed = String(raw ?? "").trim();

  if (!trimmed) {
    return { value: {} };
  }

  try {
    return { value: JSON.parse(trimmed) };
  } catch {
    return { error: "Metadata must be valid JSON." };
  }
}

function openCreateDialog() {
  resetForm();
  metadataError.value = "";
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  resetForm();
  metadataError.value = "";
  showCreateDialog.value = false;
}

function resetForm() {
  form.hostname = "";
  form.operating_system = "";
  form.hypervisor = "";
  form.location = "";
  form.metadata = "{}";
}

async function previousPage() {
  if (pagination.page <= 1) return;
  pagination.page -= 1;
  await loadServers();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadServers();
}
</script>
