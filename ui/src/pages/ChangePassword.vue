<template>
  <section class="flex min-h-screen items-center justify-center bg-surface px-6 py-12 dark:bg-primary-950">
    <div class="w-full max-w-sm">
      <a href="/" class="mb-12 inline-block">
        <img class="h-9 dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="shutterbase" />
        <img class="hidden h-9 dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
      </a>

      <p class="label-mono text-accent-600 dark:text-accent-400">Account security</p>
      <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white">Change your password</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Choose a new password to continue to your projects.</p>

      <form class="mt-9 space-y-5" action="#">
        <div>
          <label for="current" class="label-mono block text-primary-500 dark:text-primary-400">Current password</label>
          <input
            v-model="currentPassword"
            type="password"
            id="current"
            autocomplete="current-password"
            placeholder="••••••••"
            class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
          />
        </div>
        <div>
          <label for="new" class="label-mono block text-primary-500 dark:text-primary-400">New password</label>
          <input
            v-model="newPassword"
            type="password"
            id="new"
            autocomplete="new-password"
            placeholder="••••••••"
            class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
          />
        </div>
        <div>
          <label for="confirm" class="label-mono block text-primary-500 dark:text-primary-400">Confirm new password</label>
          <input
            v-model="newPasswordConfirm"
            type="password"
            id="confirm"
            autocomplete="new-password"
            placeholder="••••••••"
            class="mt-2 block h-11 w-full rounded-md border border-primary-200 bg-surface-muted px-3.5 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-white dark:placeholder:text-primary-500 dark:hover:border-primary-600"
          />
        </div>
        <p v-if="errorMessage" class="text-sm font-medium text-error-600 dark:text-error-400">{{ errorMessage }}</p>
        <button
          type="submit"
          @click.prevent="submit"
          class="flex h-11 w-full items-center justify-center rounded-md bg-accent-600 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface active:bg-accent-700 dark:focus-visible:ring-offset-primary-950"
        >
          Change password
        </button>
      </form>
    </div>
  </section>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useUserStore } from "src/stores/user-store";
import { showNotificationToast } from "src/boot/mitt";

const router = useRouter();
const userStore = useUserStore();

const currentPassword = ref("");
const newPassword = ref("");
const newPasswordConfirm = ref("");
const errorMessage = ref("");

const CODE_MESSAGES: Record<string, string> = {
  passwords_do_not_match: "The new passwords do not match",
  password_requirements_not_met: "Password must be at least 8 characters and contain upper/lower case letters and a digit",
  incorrect_password: "The current password is incorrect",
};

async function submit() {
  errorMessage.value = "";
  if (newPassword.value !== newPasswordConfirm.value) {
    errorMessage.value = CODE_MESSAGES.passwords_do_not_match;
    return;
  }
  try {
    await userStore.changePassword({
      currentPassword: currentPassword.value,
      newPassword: newPassword.value,
      newPasswordConfirm: newPasswordConfirm.value,
    });
    showNotificationToast({ headline: "Password changed", type: "success" });
    router.push("/");
  } catch (error: any) {
    const code = error.response?.data?.code;
    errorMessage.value = (code && CODE_MESSAGES[code]) || error.response?.data?.message || error.response?.data?.error || "Failed to change password";
  }
}
</script>
