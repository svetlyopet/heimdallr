<template>
  <section>
    <BreadcrumbNav :items="breadcrumbItems" />

    <header class="topbar">
      <div>
        <p class="eyebrow">Compliance</p>
        <h2>{{ agent?.name || "Agent" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/agents">
          All agents
        </RouterLink>
        <RouterLink
          class="button button-secondary"
          :to="{ name: 'server-detail', params: { serverId } }"
        >
          Back to server
        </RouterLink>
        <button class="button button-secondary" type="button" @click="loadAgent">
          Refresh
        </button>
        <button
          class="button"
          type="button"
          :disabled="loading || deleting"
          @click="removeAgent"
        >
          Remove agent
        </button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

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
            <dt>Version</dt>
            <dd>{{ agent.version || "—" }}</dd>
          </div>

          <div>
            <dt>Server</dt>
            <dd>{{ agent.server || "—" }}</dd>
          </div>

          <div>
            <dt>Server ID</dt>
            <dd><code>{{ agent.server_id || "—" }}</code></dd>
          </div>

          <div>
            <dt>Created</dt>
            <dd>{{ formatDateTime(agent.created_at) }}</dd>
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
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import "../stylesheets/detail.css";
import "../stylesheets/breadcrumb.css";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { deleteAgent, getAgent } from "../api/agents";
import AppAlert from "../components/AppAlert.vue";
import BreadcrumbNav from "../components/BreadcrumbNav.vue";
import { formatDateTime } from "../utils/format";

const route = useRoute();
const router = useRouter();
const serverId = route.params.serverId;
const agentId = route.params.agentId;

const agent = ref(null);
const loading = ref(false);
const deleting = ref(false);
const errorMessage = ref("");

const breadcrumbItems = computed(() => [
  { label: "Servers", to: { name: "servers" } },
  {
    label: agent.value?.server || "Server",
    to: { name: "server-detail", params: { serverId } },
  },
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
    const response = await getAgent(serverId, agentId);
    agent.value = response.data || null;
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function removeAgent() {
  if (!window.confirm("Remove this agent from the server?")) {
    return;
  }

  deleting.value = true;
  errorMessage.value = "";

  try {
    await deleteAgent(serverId, agentId);
    await router.push({ name: "server-detail", params: { serverId } });
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    deleting.value = false;
  }
}
</script>
