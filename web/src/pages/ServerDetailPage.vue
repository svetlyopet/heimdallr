<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Fleet</p>
        <h2>{{ server?.hostname || "Server" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/servers">Back</RouterLink>
        <RouterLink
          class="button button-secondary"
          :to="{ name: 'server-jobs', params: { serverId } }"
        >
          View jobs
        </RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">Refresh</button>
        <button class="button button-secondary" type="button" @click="openAttachDialog">
          Attach agent
        </button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showAttachDialog"
      eyebrow="Attach"
      title="Attach existing agent"
      @close="closeAttachDialog"
    >
      <form class="form" @submit.prevent="submitAttach">
        <label>
          Agent
          <select v-model="attachAgentId" required>
            <option value="" disabled>Select agent</option>
            <option v-for="agent in availableAgents" :key="agent.id" :value="agent.id">
              {{ agent.name }} ({{ agent.server_count }} servers)
            </option>
          </select>
        </label>
        <button class="button button-full" type="submit" :disabled="submitting || !attachAgentId">
          Attach agent
        </button>
      </form>
    </FormDialog>

    <section v-if="server" class="job-detail-grid">
      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Overview</p>
            <h3>{{ server.hostname }}</h3>
          </div>
        </div>

        <dl class="detail-grid">
          <div>
            <dt>Server ID</dt>
            <dd><code>{{ server.id }}</code></dd>
          </div>

          <div>
            <dt>Operating system</dt>
            <dd>{{ server.operating_system || "—" }}</dd>
          </div>

          <div>
            <dt>Hypervisor</dt>
            <dd>{{ server.hypervisor || "—" }}</dd>
          </div>

          <div>
            <dt>Location</dt>
            <dd>{{ server.location || "—" }}</dd>
          </div>

          <div>
            <dt>Agents</dt>
            <dd>{{ server.relations?.agent_count ?? 0 }}</dd>
          </div>

          <div>
            <dt>Jobs</dt>
            <dd>{{ server.relations?.job_count ?? 0 }}</dd>
          </div>

          <div>
            <dt>Releases</dt>
            <dd>{{ server.relations?.release_count ?? 0 }}</dd>
          </div>
        </dl>
      </article>

      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Metadata</p>
            <h3>JSON data</h3>
          </div>
        </div>

        <pre class="json-block">{{ formattedMetadata }}</pre>
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
          <span>Attach an agent to track compliance tooling on this server.</span>
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
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{
                        name: 'agent-detail-global',
                        params: { agentId: agent.id },
                      }"
                    >
                      View
                    </RouterLink>
                    <button
                      class="button button-small button-secondary"
                      type="button"
                      :disabled="detaching"
                      @click="removeAgent(agent.id)"
                    >
                      Detach
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
import { computed, onMounted, reactive, ref } from "vue";
import "../stylesheets/detail.css";
import "../stylesheets/dialog.css";
import { RouterLink, useRoute } from "vue-router";
import {
  attachAgent,
  detachAgent,
  listAgents,
  listAgentsForServer,
} from "../api/agents";
import { getServer } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";
import PaginationControls from "../components/PaginationControls.vue";

const route = useRoute();
const serverId = route.params.serverId;

const server = ref(null);
const agents = ref([]);
const availableAgents = ref([]);
const loading = ref(false);
const submitting = ref(false);
const detaching = ref(false);
const errorMessage = ref("");
const showAttachDialog = ref(false);
const attachAgentId = ref("");

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  total_pages: 0,
});

const formattedMetadata = computed(() => {
  const metadata = server.value?.metadata;
  if (!metadata) return "{}";

  if (typeof metadata === "string") {
    try {
      return JSON.stringify(JSON.parse(metadata), null, 2);
    } catch {
      return metadata;
    }
  }

  return JSON.stringify(metadata, null, 2);
});

onMounted(loadData);

async function loadData() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const [serverResponse, agentResponse] = await Promise.all([
      getServer(serverId),
      listAgentsForServer(serverId, { page: pagination.page, limit: pagination.limit }),
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

async function loadAvailableAgents() {
  try {
    const response = await listAgents({ page: 1, limit: 100 });
    availableAgents.value = response.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function submitAttach() {
  if (!attachAgentId.value) {
    return;
  }

  submitting.value = true;
  errorMessage.value = "";

  try {
    await attachAgent(serverId, attachAgentId.value);
    closeAttachDialog();
    pagination.page = 1;
    await loadData();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    submitting.value = false;
  }
}

async function removeAgent(agentId) {
  if (!window.confirm("Detach this agent from the server?")) {
    return;
  }

  detaching.value = true;
  errorMessage.value = "";

  try {
    await detachAgent(serverId, agentId);
    await loadData();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    detaching.value = false;
  }
}

async function openAttachDialog() {
  attachAgentId.value = "";
  await loadAvailableAgents();
  showAttachDialog.value = true;
}

function closeAttachDialog() {
  attachAgentId.value = "";
  showAttachDialog.value = false;
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
