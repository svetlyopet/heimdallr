<template>
  <section>
    <BreadcrumbNav :items="breadcrumbItems" />

    <header class="topbar">
      <div>
        <p class="eyebrow">Compliance</p>
        <h2>Report {{ report?.id }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink
          class="button button-secondary"
          :to="{
            name: 'reports',
            query: { application_id: applicationId, release_id: releaseId },
          }"
        >
          All reports
        </RouterLink>
        <RouterLink
          class="button button-secondary"
          :to="{ name: 'release-detail', params: { id: applicationId, releaseId } }"
        >
          Back to release
        </RouterLink>
        <button class="button button-secondary" type="button" @click="loadReport">Refresh</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="loading" class="empty-state">Loading report...</section>

    <section v-else-if="!report" class="empty-state">
      <strong>Report not found</strong>
    </section>

    <section v-else class="job-detail-grid">
      <article class="panel detail-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Overview</p>
            <h3>{{ report.type }}</h3>
          </div>
          <span class="badge" :class="`badge-${report.status}`">{{ report.status }}</span>
        </div>

        <dl class="detail-grid">
          <div>
            <dt>Application</dt>
            <dd>{{ report.application || "—" }}</dd>
          </div>
          <div>
            <dt>Version</dt>
            <dd>{{ report.version || "—" }}</dd>
          </div>
          <div>
            <dt>Location</dt>
            <dd>{{ report.location || "—" }}</dd>
          </div>
          <div>
            <dt>Created</dt>
            <dd>{{ formatDateTime(report.created_at) }}</dd>
          </div>
          <div>
            <dt>URL</dt>
            <dd>
              <a v-if="report.url" :href="report.url" target="_blank" rel="noreferrer">{{ report.url }}</a>
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
          <span>This report does not have output data.</span>
        </div>

        <iframe v-else class="output-frame" title="Report output" sandbox="" :srcdoc="outputDocument"></iframe>
      </article>
    </section>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import "../stylesheets/detail.css";
import { RouterLink, useRoute } from "vue-router";
import { getReport } from "../api/reports";
import AppAlert from "../components/AppAlert.vue";
import BreadcrumbNav from "../components/BreadcrumbNav.vue";
import { formatDateTime } from "../utils/format";

const route = useRoute();
const applicationId = route.params.id;
const releaseId = route.params.releaseId;
const reportId = route.params.reportId;

const report = ref(null);
const loading = ref(false);
const errorMessage = ref("");

const breadcrumbItems = computed(() => [
  { label: "Applications", to: { name: "applications" } },
  {
    label: report.value?.application || "Application",
    to: { name: "application-detail", params: { id: applicationId } },
  },
  { label: "Releases", to: { name: "releases" } },
  {
    label: report.value?.version || "Release",
    to: { name: "release-detail", params: { id: applicationId, releaseId } },
  },
  { label: "Reports", to: { name: "reports", query: { application_id: applicationId, release_id: releaseId } } },
  { label: report.value?.id || reportId },
]);

const decodedOutput = computed(() => decodeBase64(report.value?.output || ""));

const formattedMetadata = computed(() => {
  const metadata = report.value?.metadata;
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

const outputDocument = computed(() => {
  const output = decodedOutput.value;
  if (!output) return "";

  if (looksLikeHtml(output)) {
    return output;
  }

  return `<!doctype html><html><head><style>body{margin:0;padding:16px;font:14px/1.6 ui-monospace,monospace;white-space:pre-wrap;}</style></head><body>${escapeHtml(output)}</body></html>`;
});

onMounted(loadReport);

async function loadReport() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await getReport(applicationId, releaseId, reportId);
    report.value = response.data || null;
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function decodeBase64(value) {
  if (!value) return "";
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
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}
</script>
