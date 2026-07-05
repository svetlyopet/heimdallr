<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Compliance</p>
        <h2>{{ application?.name || "Application" }}</h2>
      </div>

      <div class="topbar-actions">
        <RouterLink class="button button-secondary" to="/applications">Back</RouterLink>
        <button class="button button-secondary" type="button" @click="loadData">Refresh</button>
        <button class="button" type="button" @click="showCreatePanel = true">Create release</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <section v-if="application" class="stats-grid">
      <article class="stat-card">
        <span>Application ID</span>
        <strong><code>{{ application.id }}</code></strong>
      </article>
      <article class="stat-card">
        <span>Releases</span>
        <strong>{{ pagination.total }}</strong>
      </article>
    </section>

    <section class="dashboard-grid">
      <article v-if="showCreatePanel" class="panel form-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Create</p>
            <h3>New release</h3>
          </div>
          <button class="icon-button" type="button" @click="showCreatePanel = false">×</button>
        </div>

        <form class="form" @submit.prevent="submitRelease">
          <label>
            Version
            <input v-model.trim="form.version" type="text" required />
          </label>
          <label>
            Commit SHA
            <input v-model.trim="form.commit_sha" type="text" />
          </label>
          <label>
            Branch
            <input v-model.trim="form.branch" type="text" />
          </label>
          <label>
            Pipeline URL
            <input v-model.trim="form.pipeline_url" type="url" />
          </label>
          <button class="button button-full" type="submit" :disabled="loading">Create release</button>
        </form>
      </article>

      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Releases</p>
            <h3>Version history</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading releases...</div>
        <div v-else-if="releases.length === 0" class="empty-state">
          <strong>No releases yet</strong>
          <span>Create a release or push from CI with upsert.</span>
        </div>
        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Version</th>
                <th>Branch</th>
                <th>Commit</th>
                <th>Pipeline</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="release in releases" :key="release.id">
                <td><strong>{{ release.version }}</strong></td>
                <td>{{ release.branch || "—" }}</td>
                <td><code>{{ release.commit_sha || "—" }}</code></td>
                <td>
                  <a v-if="release.pipeline_url" :href="release.pipeline_url" target="_blank" rel="noreferrer">
                    Open
                  </a>
                  <span v-else>—</span>
                </td>
                <td>
                  <RouterLink
                    class="button button-secondary"
                    :to="{
                      name: 'release-detail',
                      params: { id: applicationId, releaseId: release.id },
                    }"
                  >
                    View reports
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
import { RouterLink, useRoute } from "vue-router";
import { getApplication } from "../api/applications";
import { createRelease, listReleases } from "../api/releases";
import AppAlert from "../components/AppAlert.vue";
import PaginationControls from "../components/PaginationControls.vue";

const route = useRoute();
const applicationId = route.params.id;

const application = ref(null);
const releases = ref([]);
const loading = ref(false);
const errorMessage = ref("");
const showCreatePanel = ref(false);

const form = reactive({
  version: "",
  commit_sha: "",
  branch: "",
  pipeline_url: "",
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
    const [appResponse, releaseResponse] = await Promise.all([
      getApplication(applicationId),
      listReleases(applicationId, { page: pagination.page, limit: pagination.limit }),
    ]);

    application.value = appResponse.data || null;
    releases.value = releaseResponse.data || [];
    Object.assign(pagination, releaseResponse.pagination || pagination);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

async function submitRelease() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createRelease(applicationId, { ...form });
    showCreatePanel.value = false;
    form.version = "";
    form.commit_sha = "";
    form.branch = "";
    form.pipeline_url = "";
    pagination.page = 1;
    await loadData();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
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
