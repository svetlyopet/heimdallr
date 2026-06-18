<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Execution</p>
        <h2>Job details</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/jobs">
          Back to jobs
        </RouterLink>

        <button class="button button-secondary" type="button" @click="loadJob">
          Refresh
        </button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="loading" class="empty-state">Loading job...</section>

    <section v-else-if="!job" class="empty-state">
      <strong>Job not found</strong>
      <span>The selected job could not be loaded.</span>
    </section>

    <section v-else class="job-detail-grid">
      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Overview</p>
            <h3>{{ job.id }}</h3>
          </div>

          <span class="badge" :class="`badge-${job.status}`">
            {{ job.status }}
          </span>
        </div>

        <dl class="detail-grid">
          <div>
            <dt>Automation</dt>
            <dd>{{ job.automation || "—" }}</dd>
          </div>

          <div>
            <dt>Provider</dt>
            <dd>{{ job.provider || "—" }}</dd>
          </div>

          <div>
            <dt>Location</dt>
            <dd>{{ job.location || "—" }}</dd>
          </div>

          <div>
            <dt>URL</dt>
            <dd>
              <a v-if="job.url" :href="job.url" target="_blank" rel="noreferrer">
                {{ job.url }}
              </a>
              <span v-else>—</span>
            </dd>
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

      <article class="panel detail-panel job-output-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Output</p>
            <h3>Rendered output</h3>
          </div>
        </div>

        <div v-if="!decodedOutput" class="empty-state">
          <strong>No output</strong>
          <span>This job does not have output data.</span>
        </div>

        <iframe
          v-else
          class="output-frame"
          title="Job output"
          sandbox=""
          :srcdoc="outputDocument"
        ></iframe>
      </article>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { RouterLink, useRoute } from "vue-router";
import { getJob } from "../api/jobs";
import AppAlert from "../components/AppAlert.vue";

const route = useRoute();

const job = ref(null);
const loading = ref(false);
const errorMessage = ref("");

const decodedOutput = computed(() => decodeBase64(job.value?.output || ""));

const formattedMetadata = computed(() => {
  const metadata = job.value?.metadata ?? job.value?.data;

  if (!metadata) {
    return "{}";
  }

  if (typeof metadata === "string") {
    try {
      return JSON.stringify(JSON.parse(metadata), null, 2);
    } catch {
      return metadata;
    }
  }

  return JSON.stringify(metadata, null, 2);
});

const outputDocument = computed(() => {
  const output = decodedOutput.value;

  if (!output) {
    return "";
  }

  if (looksLikeHtml(output)) {
    return output;
  }

  return `<!doctype html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        padding: 16px;
        color: #0f172a;
        background: #ffffff;
        font: 14px/1.6 ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
      }
    </style>
  </head>
  <body>${escapeHtml(output)}</body>
</html>`;
});

onMounted(loadJob);

async function loadJob() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await getJob(route.params.automationId, route.params.jobId);
    job.value = response.data || null;
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function decodeBase64(value) {
  if (!value) {
    return "";
  }

  try {
    const binary = atob(value);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));

    return new TextDecoder().decode(bytes);
  } catch {
    return "";
  }
}

function looksLikeHtml(value) {
  return /<\/?[a-z][\s\S]*>/i.test(value);
}

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}
</script>