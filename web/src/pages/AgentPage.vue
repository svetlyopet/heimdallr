<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Compliance</p>
        <h2>Agents</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadAll">
          Refresh
        </button>
        <button
          class="button"
          type="button"
          :disabled="!selectedServerId"
          @click="openCreateDialog"
        >
          Register agent
        </button>
      </div>
    </header>

    <StatsGrid :pagination="pagination" />
    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New agent"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitAgent">
        <label>
          Server
          <select v-model="form.serverId" required>
            <option value="" disabled>Select server</option>
            <option v-for="server in servers" :key="server.id" :value="server.id">
              {{ server.hostname }}
            </option>
          </select>
        </label>

        <label>
          Name
          <input v-model.trim="form.name" type="text" required minlength="1" maxlength="255" />
        </label>

        <label>
          Type
          <input v-model.trim="form.type" type="text" maxlength="255" />
        </label>

        <label>
          Version
          <input v-model.trim="form.version" type="text" maxlength="255" />
        </label>

        <button class="button button-full" type="submit" :disabled="submitting">
          Register agent
        </button>
      </form>
    </FormDialog>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>Agent list</h3>
          </div>

          <div class="page-size">
            <label>
              Server
              <select v-model="selectedServerId">
                <option value="" disabled>Select server</option>
                <option v-for="server in servers" :key="server.id" :value="server.id">
                  {{ server.hostname }}
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

        <div v-if="!selectedServerId" class="empty-state">
          <strong>Select a server</strong>
          <span>Agents are scoped by server.</span>
        </div>

        <div v-else-if="loading" class="empty-state">Loading agents...</div>

        <div v-else-if="agents.length === 0" class="empty-state">
          <strong>No agents yet</strong>
          <span>No agents were found for this server.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Version</th>
                <th>Server</th>
                <th>ID</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              <tr v-for="agent in agents" :key="agent.id">
                <td data-label="Name"><strong>{{ agent.name }}</strong></td>
                <td data-label="Type">{{ agent.type || "—" }}</td>
                <td data-label="Version">{{ agent.version || "—" }}</td>
                <td data-label="Server">{{ agent.server || "—" }}</td>
                <td data-label="ID"><code>{{ agent.id }}</code></td>
                <td data-label="Actions">
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{
                        name: 'agent-detail',
                        params: {
                          serverId: selectedServerId,
                          agentId: agent.id,
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
import "../stylesheets/dialog.css";
import { RouterLink } from "vue-router";
import { createAgent, listAgents } from "../api/agents";
import { listServers } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";
import StatsGrid from "../components/StatsGrid.vue";

const servers = ref([]);
const agents = ref([]);
const selectedServerId = ref("");
const loading = ref(false);
const submitting = ref(false);
const errorMessage = ref("");
const showCreateDialog = ref(false);

const form = reactive({
  serverId: "",
  name: "",
  type: "",
  version: "",
});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

onMounted(loadAll);

watch(selectedServerId, async () => {
  pagination.page = 1;
  await loadAgentsForServer();
});

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const serverResponse = await listServers({ page: 1, limit: 100 });
    servers.value = serverResponse.data || [];

    if (!selectedServerId.value && servers.value.length > 0) {
      selectedServerId.value = servers.value[0].id;
    }

    await loadAgentsForServer();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function loadAgentsForServer() {
  if (!selectedServerId.value) {
    agents.value = [];
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
    const response = await listAgents(selectedServerId.value, {
      page: pagination.page,
      limit: pagination.limit,
    });

    agents.value = response.data || [];
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
  await loadAgentsForServer();
}

async function nextPage() {
  if (pagination.page >= pagination.total_pages) return;
  pagination.page += 1;
  await loadAgentsForServer();
}

async function changeLimit() {
  pagination.page = 1;
  await loadAgentsForServer();
}

function openCreateDialog() {
  resetForm();
  form.serverId = selectedServerId.value;
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  resetForm();
  showCreateDialog.value = false;
}

function resetForm() {
  form.serverId = selectedServerId.value;
  form.name = "";
  form.type = "";
  form.version = "";
}

async function submitAgent() {
  if (!form.serverId) {
    errorMessage.value = "Select a server before registering an agent.";
    return;
  }

  submitting.value = true;
  errorMessage.value = "";

  try {
    await createAgent(form.serverId, {
      name: form.name,
      type: form.type,
      version: form.version,
      metadata: {},
    });

    selectedServerId.value = form.serverId;
    closeCreateDialog();
    pagination.page = 1;
    await loadAgentsForServer();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    submitting.value = false;
  }
}
</script>
