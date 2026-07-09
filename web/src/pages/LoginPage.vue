<template>
  <section class="auth-page">
    <article class="panel auth-panel">
      <div class="panel-header">
        <div>
          <p class="eyebrow">Authentication</p>
          <h2>Sign in</h2>
        </div>
      </div>

      <AppAlert :message="errorMessage" @dismiss="errorMessage = ''" />

      <form class="form" @submit.prevent="submitLogin">
        <label>
          Username
          <input v-model.trim="form.username" type="text" required />
        </label>

        <label>
          Password
          <input v-model="form.password" type="password" required />
        </label>

        <button class="button button-full" type="submit" :disabled="sessionState.checking">
          {{ sessionState.checking ? "Signing in..." : "Sign in" }}
        </button>
      </form>
    </article>
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import "../stylesheets/auth.css";
import { useRouter } from "vue-router";
import AppAlert from "../components/AppAlert.vue";
import { clearSession, ensureSessionAccess, loginWithCredentials, sessionState } from "../auth/session";

const router = useRouter();

const form = reactive({
  username: "",
  password: "",
});

const errorMessage = ref("");

async function submitLogin() {
  errorMessage.value = "";
  sessionState.checking = true;

  try {
    await loginWithCredentials(form.username, form.password);
    await ensureSessionAccess();

    if (sessionState.authenticated) {
      await router.push({ name: "dashboard" });
      return;
    }
  } catch {
    clearSession();
    errorMessage.value = "Invalid username or password";
    return;
  } finally {
    sessionState.checking = false;
  }

  clearSession();
  errorMessage.value = "Invalid username or password";
}

onMounted(async () => {
  await ensureSessionAccess();
  if (sessionState.authenticated) {
    await router.replace({ name: "dashboard" });
  }
});
</script>