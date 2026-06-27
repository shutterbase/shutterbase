<template>
  <section class="bg-gray-50 dark:bg-gray-900">
    <div class="flex flex-col items-center justify-center px-6 py-8 mx-auto md:h-screen lg:py-0">
      <a href="/" class="flex items-center mb-6 text-2xl font-semibold text-gray-900 dark:text-white">
        <img class="h-12 mr-2 block dark:!hidden" src="~assets/img/shutterbase-header-logo-light.png" alt="logo" />
        <img class="h-12 mr-2 hidden dark:!block" src="~assets/img/shutterbase-header-logo-dark.png" alt="logo" />
      </a>
      <div class="w-full bg-white rounded-lg shadow dark:border md:mt-0 sm:max-w-md xl:p-0 dark:!bg-gray-800 dark:border-gray-700">
        <div class="p-6 space-y-4 md:space-y-6 sm:p-8">
          <h1 class="text-xl font-bold leading-tight tracking-tight text-gray-900 md:text-2xl dark:text-white">Change your password</h1>
          <form class="space-y-4 md:space-y-6" action="#">
            <div>
              <label for="current" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Current password</label>
              <input
                v-model="currentPassword"
                type="password"
                id="current"
                autocomplete="current-password"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
              />
            </div>
            <div>
              <label for="new" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">New password</label>
              <input
                v-model="newPassword"
                type="password"
                id="new"
                autocomplete="new-password"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
              />
            </div>
            <div>
              <label for="confirm" class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Confirm new password</label>
              <input
                v-model="newPasswordConfirm"
                type="password"
                id="confirm"
                autocomplete="new-password"
                placeholder="••••••••"
                class="bg-gray-50 border border-gray-300 text-gray-900 sm:text-sm rounded-lg focus:ring-primary-600 focus:border-primary-600 block w-full p-2.5 dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 dark:text-white"
              />
            </div>
            <p v-if="errorMessage" class="mt-2 text-sm text-red-600 dark:text-red-500">
              <span class="font-medium">{{ errorMessage }}</span>
            </p>
            <button
              type="submit"
              @click.prevent="submit"
              class="w-full text-white bg-primary-600 hover:bg-primary-700 focus:ring-4 focus:outline-none focus:ring-primary-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center dark:bg-primary-600 dark:hover:bg-primary-700"
            >
              Change password
            </button>
          </form>
        </div>
      </div>
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
