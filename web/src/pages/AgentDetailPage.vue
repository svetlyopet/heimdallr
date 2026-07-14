<template>
  <section>
    <BreadcrumbNav :items="breadcrumbItems" />

    <header class="topbar">
      <div>
        <p class="eyebrow">Fleet</p>
        <h2>{{ agent?.name || "Agent" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/agents">
          All agents
        </RouterLink>
        <button class="button button-secondary" type="button" @click="openAttachDialog">
          Attach to server
        </button>
        <button class="button button-secondary" type="button" @click="loadAgent">
          Refresh
        </button>
        <button
          class="button"
          type="button"
          :disabled="loading || deleting"
          @click="removeAgent"
        >
          Delete agent
        </button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showAttachDialog"
      eyebrow="Attach"
      title="Attach to server"
      @close="closeAttachDialog"
    >
      <form class="form" @submit.prevent="submitAttach">
        <label>
          Server
          <select v-model="attachServerId" required>
            <option value="" disabled>Select server</option>
            <option v-for="server in servers" :key="server.id" :value="server.id">
              {{ server.hostname }}
            </option>
          </select>
        </label>
        <button class="button button-full" type="submit" :disabled="attaching || !attachServerId">
          Attach to server
        </button>
      </form>
    </FormDialog>

    <section v-if="loading" class="empty-state">Loading agent...</section>

    <section v-else-if="!agent" class="empty-state">
      <strong>Agent not found</strong>
      <span>The selected agent could not be loaded.</span>
    </section>

    <section v-else class="job-detail-grid">
      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Overview</p>
            <h3>{{ agent.name }}</h3>
          </div>
        </div>

        <dl class="detail-grid">
          <div>
            <dt>Name</dt>
            <dd>{{ agent.name || "—" }}</dd>
          </div>

          <div>
            <dt>Type</dt>
            <dd>{{ agent.type || "—" }}</dd>
          </div>

          <div>
            <dt>Servers</dt>
            <dd>{{ agent.server_count ?? 0 }}</dd>
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

      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Installations</p>
            <h3>Linked servers</h3>
          </div>
        </div>

        <div v-if="linkedServers.length === 0" class="empty-state">
          <strong>No linked servers</strong>
          <span>This agent is not installed on any server yet.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Hostname</th>
                <th>ID</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="server in linkedServers" :key="server.id">
                <td><strong>{{ server.hostname }}</strong></td>
                <td><code>{{ server.id }}</code></td>
                <td>
                  <div class="row-actions">
                    <RouterLink
                      class="button button-small button-secondary"
                      :to="{ name: 'server-detail', params: { serverId: server.id } }"
                    >
                      View server
                    </RouterLink>
                    <button
                      class="button button-small button-secondary"
                      type="button"
                      :disabled="detaching"
                      @click="detachFromServer(server.id)"
                    >
                      Detach
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import "../stylesheets/detail.css";
import "../stylesheets/breadcrumb.css";
import "../stylesheets/dialog.css";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { attachAgent, deleteAgent, detachAgent, getAgent } from "../api/agents";
import { listServers } from "../api/servers";
import AppAlert from "../components/AppAlert.vue";
import BreadcrumbNav from "../components/BreadcrumbNav.vue";
import FormDialog from "../components/FormDialog.vue";

const route = useRoute();
const router = useRouter();
const agentId = route.params.agentId;

const agent = ref(null);
const linkedServers = ref([]);
const servers = ref([]);
const loading = ref(false);
const deleting = ref(false);
const detaching = ref(false);
const attaching = ref(false);
const errorMessage = ref("");
const showAttachDialog = ref(false);
const attachServerId = ref("");

const breadcrumbItems = computed(() => [
  { label: "Agents", to: { name: "agents" } },
  { label: agent.value?.name || agentId },
]);

const formattedMetadata = computed(() => {
  const metadata = agent.value?.metadata;
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

onMounted(loadAgent);

async function loadAgent() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await getAgent(agentId);
    agent.value = response.data || null;
    linkedServers.value = response.data?.servers || [];
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function loadServers() {
  try {
    const response = await listServers({ page: 1, limit: 100 });
    servers.value = response.data || [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function openAttachDialog() {
  attachServerId.value = "";
  await loadServers();
  showAttachDialog.value = true;
}

function closeAttachDialog() {
  attachServerId.value = "";
  showAttachDialog.value = false;
}

async function submitAttach() {
  if (!attachServerId.value) {
    return;
  }

  attaching.value = true;
  errorMessage.value = "";

  try {
    await attachAgent(attachServerId.value, agentId);
    closeAttachDialog();
    await loadAgent();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    attaching.value = false;
  }
}

async function detachFromServer(serverId) {
  if (!window.confirm("Detach this agent from the server?")) {
    return;
  }

  detaching.value = true;
  errorMessage.value = "";

  try {
    await detachAgent(serverId, agentId);
    await loadAgent();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    detaching.value = false;
  }
}

async function removeAgent() {
  if (!window.confirm("Delete this agent and remove all server links?")) {
    return;
  }

  deleting.value = true;
  errorMessage.value = "";

  try {
    await deleteAgent(agentId);
    await router.push({ name: "agents" });
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    deleting.value = false;
  }
}
</script>
