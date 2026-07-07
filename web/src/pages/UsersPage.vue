<template>
  <section>
    <header class="topbar">
      <div>
        <p class="eyebrow">Administration</p>
        <h2>Users</h2>
      </div>

      <div class="topbar-actions">
        <button class="button button-secondary" type="button" @click="loadUsers">
          Refresh
        </button>
        <button class="button" type="button" @click="openCreateDialog">Create user</button>
      </div>
    </header>

    <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

    <FormDialog
      :open="showCreateDialog"
      eyebrow="Create"
      title="New user"
      @close="closeCreateDialog"
    >
      <form class="form" @submit.prevent="submitCreate">
        <label>
          Username
          <input v-model.trim="createForm.username" type="text" required minlength="2" maxlength="64" />
        </label>

        <label>
          Email
          <input v-model.trim="createForm.email" type="email" required />
        </label>

        <label>
          Password
          <input v-model="createForm.password" type="password" required minlength="12" />
        </label>

        <label>
          Roles
          <select v-model="createForm.role">
            <option value="reader">reader</option>
            <option value="admin">admin</option>
          </select>
        </label>

        <button class="button button-full" type="submit" :disabled="loading">Create user</button>
      </form>
    </FormDialog>

    <section class="dashboard-grid">
      <article class="panel table-panel">
        <div class="panel-header">
          <div>
            <p class="eyebrow">Inventory</p>
            <h3>User list</h3>
          </div>
        </div>

        <div v-if="loading" class="empty-state">Loading users...</div>
        <div v-else-if="users.length === 0" class="empty-state">
          <strong>No users found</strong>
          <span>Create your first user.</span>
        </div>

        <div v-else class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>Username</th>
                <th>Email</th>
                <th>Roles</th>
                <th>ID</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id">
                <td><strong>{{ user.username }}</strong></td>
                <td>
                  <input v-model.trim="editById[user.id].email" class="table-input" type="email" />
                </td>
                <td>
                  <select v-model="editById[user.id].role" class="table-input">
                    <option value="reader">reader</option>
                    <option value="admin">admin</option>
                  </select>
                </td>
                <td><code>{{ user.id }}</code></td>
                <td>
                  <div class="table-actions">
                    <button class="button button-secondary" type="button" :disabled="loading" @click="saveUser(user.id)">
                      Save
                    </button>
                    <button class="button button-danger" type="button" :disabled="loading" @click="removeUser(user.id)">
                      Delete
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
import { onMounted, reactive, ref } from "vue";
import "../stylesheets/users-table.css";
import { createUser, deleteUser, listUsers, updateUser } from "../api/users";
import AppAlert from "../components/AppAlert.vue";
import FormDialog from "../components/FormDialog.vue";

const loading = ref(false);
const errorMessage = ref("");
const users = ref([]);
const editById = reactive({});
const showCreateDialog = ref(false);

const createForm = reactive({
  username: "",
  email: "",
  password: "",
  role: "reader",
});

function syncEditState() {
  users.value.forEach((user) => {
    editById[user.id] = {
      email: user.email,
      role: user.roles?.[0] ?? "reader",
    };
  });
}

async function loadUsers() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listUsers();
    users.value = response.data ?? [];
    syncEditState();
  } catch (error) {
    errorMessage.value = error.message || "Failed to load users";
  } finally {
    loading.value = false;
  }
}

async function submitCreate() {
  loading.value = true;
  errorMessage.value = "";

  try {
    await createUser({
      username: createForm.username,
      email: createForm.email,
      password: createForm.password,
      roles: [createForm.role],
    });
    createForm.username = "";
    createForm.email = "";
    createForm.password = "";
    createForm.role = "reader";
    showCreateDialog.value = false;
    await loadUsers();
  } catch (error) {
    errorMessage.value = error.message || "Failed to create user";
    loading.value = false;
  }
}

function openCreateDialog() {
  showCreateDialog.value = true;
}

function closeCreateDialog() {
  showCreateDialog.value = false;
}

async function saveUser(userId) {
  loading.value = true;
  errorMessage.value = "";

  try {
    const edit = editById[userId];
    await updateUser(userId, {
      email: edit.email,
      roles: [edit.role],
    });
    await loadUsers();
  } catch (error) {
    errorMessage.value = error.message || "Failed to update user";
    loading.value = false;
  }
}

async function removeUser(userId) {
  loading.value = true;
  errorMessage.value = "";

  try {
    await deleteUser(userId);
    await loadUsers();
  } catch (error) {
    errorMessage.value = error.message || "Failed to delete user";
    loading.value = false;
  }
}

onMounted(loadUsers);
</script>
