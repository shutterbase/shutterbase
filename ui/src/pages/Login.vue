<template>
  <div class="flex min-h-screen bg-surface dark:bg-primary-950">
    <!-- brand panel -->
    <aside class="relative hidden w-[46%] flex-col justify-between overflow-hidden bg-primary-950 p-12 lg:flex xl:w-1/2">
      <img src="~assets/img/shutterbase-icon.png" alt="" aria-hidden="true" class="pointer-events-none absolute -bottom-40 -right-40 w-[48rem] select-none opacity-[0.05]" />
      <img src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" class="relative h-7 w-auto self-start" />
      <div class="relative max-w-md">
        <p class="label-mono text-accent-400">Collaborative photography</p>
        <div class="relative mt-5 inline-block px-3 py-2">
          <CornerMarks />
          <h2 class="display text-[2.75rem] leading-[1.04] text-white">Every shot,<br />in sync.</h2>
        </div>
        <p class="mt-6 max-w-sm text-base leading-relaxed text-primary-300">Upload, time-sync across photographers, tag, and find any frame — together, in one shared library.</p>
        <ul class="mt-10 space-y-3.5">
          <li v-for="f in features" :key="f.label" class="flex items-center gap-3.5 text-sm text-primary-200">
            <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md border border-white/10 bg-white/[0.05] text-accent-300">
              <component :is="f.icon" class="h-[18px] w-[18px]" />
            </span>
            {{ f.label }}
          </li>
        </ul>
      </div>
      <div class="relative text-xs text-primary-500">© {{ year }} shutterbase</div>
    </aside>

    <!-- form panel -->
    <main class="flex w-full flex-1 items-center justify-center px-6 py-12 sm:px-12">
      <div class="w-full max-w-sm">
        <router-link to="/" class="mb-12 inline-block lg:hidden">
          <img class="h-9 dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
          <img class="hidden h-9 dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
        </router-link>

        <p class="label-mono text-accent-600 dark:text-accent-400">Welcome back</p>
        <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Sign in</h1>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Sign in to continue to your projects.</p>

        <form class="mt-9 space-y-5" @submit.prevent="login">
          <div>
            <label for="email" class="label-mono block text-primary-500 dark:text-primary-400">Email</label>
            <input
              v-model="email"
              type="email"
              name="email"
              autocomplete="username"
              id="email"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
              placeholder="you@example.com"
            />
            <p v-if="emailErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ emailErrorMessage }}</p>
          </div>

          <div>
            <div class="flex items-center justify-between">
              <label for="password" class="label-mono block text-primary-500 dark:text-primary-400">Password</label>
              <a href="#" class="text-sm font-medium text-accent-600 transition-colors hover:text-accent-500 dark:text-accent-400">Forgot password?</a>
            </div>
            <input
              v-model="password"
              type="password"
              name="password"
              autocomplete="current-password"
              id="password"
              placeholder="••••••••"
              class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
            />
            <p v-if="passwordErrorMessage != ''" class="mt-2 text-sm font-medium text-error-600 dark:text-error-400">{{ passwordErrorMessage }}</p>
          </div>

          <button
            type="submit"
            class="flex h-11 w-full items-center justify-center rounded-md bg-accent-600 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface active:bg-accent-700 dark:focus-visible:ring-offset-primary-950"
          >
            Sign in
          </button>
        </form>

        <p class="mt-8 text-sm text-primary-500 dark:text-primary-400">
          Don't have an account yet?
          <router-link to="/signup" class="font-medium text-accent-600 transition-colors hover:text-accent-500 dark:text-accent-400">Sign up</router-link>
        </p>
      </div>
    </main>

    <UnexpectedErrorMessage
      :show="showUnexpectedErrorMessage"
      :error="unexpectedError"
      :headline="unexpectedErrorHeadline"
      :message="unexpectedErrorMessage"
      @closed="showUnexpectedErrorMessage = false"
    />
  </div>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import * as EmailValidator from "email-validator";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import CornerMarks from "src/components/CornerMarks.vue";
import { ClockIcon, TagIcon, MagnifyingGlassIcon } from "@heroicons/vue/24/outline";

const router = useRouter();
const userStore = useUserStore();

const year = new Date().getFullYear();
const features = [
  { icon: ClockIcon, label: "Time-sync every camera to one timeline" },
  { icon: TagIcon, label: "Tag collaboratively as a team" },
  { icon: MagnifyingGlassIcon, label: "Find any frame in seconds" },
];

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
