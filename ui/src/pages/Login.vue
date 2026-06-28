<template>
  <section class="min-h-screen flex flex-col items-center justify-center bg-primary-50 dark:bg-primary-950 px-6 py-12">
    <router-link to="/" class="mb-9">
      <img class="h-11 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
      <img class="h-11 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
    </router-link>

    <div class="w-full max-w-md rounded-xl border border-primary-200 dark:border-primary-800 bg-surface dark:bg-surface-dark shadow-panel dark:shadow-panel-dark">
      <div class="p-8">
        <h1 class="text-2xl font-semibold tracking-tight text-primary-900 dark:text-white">Sign in</h1>
        <p class="mt-1.5 text-sm text-primary-500 dark:text-primary-400">Welcome back — sign in to continue.</p>

        <form class="mt-7 space-y-5" @submit.prevent="login">
          <div>
            <label for="email" class="block text-sm font-medium text-primary-700 dark:text-primary-200">Email</label>
            <input
              v-model="email"
              type="email"
              name="email"
              autocomplete="username"
              id="email"
              class="mt-2 block w-full rounded-md border border-primary-300 dark:border-primary-700 bg-primary-50 dark:bg-primary-950 px-3.5 py-2.5 text-sm text-primary-900 dark:text-white placeholder:text-primary-400 dark:placeholder:text-primary-500 focus:border-accent-500 focus:ring-1 focus:ring-accent-500 transition-colors"
              placeholder="you@example.com"
            />
            <p v-if="emailErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ emailErrorMessage }}</p>
          </div>

          <div>
            <div class="flex items-center justify-between">
              <label for="password" class="block text-sm font-medium text-primary-700 dark:text-primary-200">Password</label>
              <a href="#" class="text-sm font-medium text-accent-600 dark:text-accent-400 hover:text-accent-500">Forgot password?</a>
            </div>
            <input
              v-model="password"
              type="password"
              name="password"
              autocomplete="current-password"
              id="password"
              placeholder="••••••••"
              class="mt-2 block w-full rounded-md border border-primary-300 dark:border-primary-700 bg-primary-50 dark:bg-primary-950 px-3.5 py-2.5 text-sm text-primary-900 dark:text-white placeholder:text-primary-400 dark:placeholder:text-primary-500 focus:border-accent-500 focus:ring-1 focus:ring-accent-500 transition-colors"
            />
            <p v-if="passwordErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ passwordErrorMessage }}</p>
          </div>

          <button
            type="submit"
            class="w-full rounded-md bg-accent-600 hover:bg-accent-500 active:bg-accent-700 px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors"
          >
            Sign in
          </button>
        </form>
      </div>

      <div class="border-t border-primary-200 dark:border-primary-800 px-8 py-4">
        <p class="text-sm text-primary-500 dark:text-primary-400">
          Don't have an account yet?
          <router-link to="/signup" class="font-medium text-accent-600 dark:text-accent-400 hover:text-accent-500">Sign up</router-link>
        </p>
      </div>
    </div>

    <UnexpectedErrorMessage
      :show="showUnexpectedErrorMessage"
      :error="unexpectedError"
      :headline="unexpectedErrorHeadline"
      :message="unexpectedErrorMessage"
      @closed="showUnexpectedErrorMessage = false"
    />
  </section>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import * as EmailValidator from "email-validator";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";

const router = useRouter();
const userStore = useUserStore();

const email = ref("");
const emailErrorMessage = ref("");
function validateUsername() {
  if (email.value === "") {
    emailErrorMessage.value = "Please enter a username";
    return false;
  } else {
    if (!EmailValidator.validate(email.value)) {
      emailErrorMessage.value = "Please enter a valid email";
      return false;
    }
  }
  return true;
}

const password = ref("");
const passwordErrorMessage = ref("");
function validatePassword() {
  if (password.value === "") {
    passwordErrorMessage.value = "Please enter a password";
    return false;
  }
  return true;
}

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const unexpectedErrorHeadline = ref("");
const unexpectedErrorMessage = ref("");

function clearErrorMessages() {
  emailErrorMessage.value = "";
  passwordErrorMessage.value = "";
}

async function login() {
  clearErrorMessages();

  if (!validateUsername() || !validatePassword()) {
    return;
  }

  try {
    const user = await userStore.login(email.value, password.value);
    if (user.forcePasswordChange) {
      router.push({ name: "change-password" });
      return;
    }
    router.push("/");
  } catch (error: any) {
    const status = error.response?.status;
    if (status === 400 || status === 401) {
      passwordErrorMessage.value = "Invalid username or password";
      return;
    }

    const message = error.response?.data?.message || error.response?.data?.error;
    if (message) {
      unexpectedErrorHeadline.value = "Error";
      unexpectedErrorMessage.value = message;
    }

    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
