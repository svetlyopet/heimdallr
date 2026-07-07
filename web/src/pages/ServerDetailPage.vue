<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Compliance</p>
        <h2>{{ server?.hostname || "Server" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/servers">Back</RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">Refresh</button>
        <button class="button" type="button" @click="openCreateDialog">Register agent</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New agent"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitAgent">
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

    <section v-if="server" class="stats-grid">
      <article class="stat-card">
        <span>Server ID</span>
        <strong><code>{{ server.id }}</code></strong>
      </article>
      <article class="stat-card">
        <span>Agents</span>
        <strong>{{ server.relations?.agent_count ?? 0 }}</strong>
      </article>
      <article class="stat-card">
        <span>Jobs</span>
        <strong>{{ server.relations?.job_count ?? 0 }}</strong>
      </article>
      <article class="stat-card">
        <span>Releases</span>
        <strong>{{ server.relations?.release_count ?? 0 }}</strong>
      </article>
    </section>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Agents</p>
            <h3>Registered agents</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading agents...</div>
        <div v-else-if="agents.length === 0" class="empty-state">
          <strong>No agents yet</strong>
          <span>Register an agent to track compliance tooling on this server.</span>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Version</th>
                <th>ID</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="agent in agents" :key="agent.id">
                <td><strong>{{ agent.name }}</strong></td>
                <td>{{ agent.type || "—" }}</td>
                <td>{{ agent.version || "—" }}</td>
                <td><code>{{ agent.id }}</code></td>
                <td>
                  <RouterLink
                    class="button button-secondary"
                    :to="{
                      name: 'agent-detail',
                      params: { serverId, agentId: agent.id },
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
import "../stylesheets/dialog.css";
import { RouterLink, useRoute } from "vue-router";
import { createAgent, listAgents } from "../api/agents";
import { getServer } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";

const route = useRoute();
const serverId = route.params.serverId;

const server = ref(null);
const agents = ref([]);
const loading = ref(false);
const submitting = ref(false);
const errorMessage = ref("");
const showCreateDialog = ref(false);

const form = reactive({
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

onMounted(loadData);

async function loadData() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [serverResponse, agentResponse] = await Promise.all([
      getServer(serverId),
      listAgents(serverId, { page: pagination.page, limit: pagination.limit }),
    ]);

    server.value = serverResponse.data || null;
    agents.value = agentResponse.data || [];
    Object.assign(pagination, agentResponse.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function submitAgent() {
  submitting.value = true;
  errorMessage.value = "";

  try {
    await createAgent(serverId, {
      name: form.name,
      type: form.type,
      version: form.version,
      metadata: {},
    });

    closeCreateDialog();
    pagination.page = 1;
    await loadData();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    submitting.value = false;
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
  form.type = "";
  form.version = "";
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
</script>
