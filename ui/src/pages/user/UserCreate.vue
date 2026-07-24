<template>
  <main class="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
    <p class="label-mono text-accent-600 dark:text-accent-400">Users</p>
    <h1 class="display mt-2 text-2xl text-primary-900 dark:text-white">New user</h1>
    <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
      Creates a local account. The new user signs in with this password and is asked to change it — projects are assigned separately on the project's Members page.
    </p>

    <form class="mt-8 space-y-5" @submit.prevent="submit">
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div v-for="field in textFields" :key="field.key">
          <label :for="field.key" class="label-mono block text-primary-500 dark:text-primary-400">{{ field.label }}</label>
          <input
            :id="field.key"
            v-model="form[field.key]"
            :type="field.type ?? 'text'"
            :autocomplete="field.autocomplete"
            :class="inputClasses"
            class="mt-2"
          />
          <PasswordRequirements v-if="field.key === 'password'" ref="pwReqs" :password="form.password" always-visible />
        </div>
      </div>

      <label class="inline-flex cursor-pointer items-center gap-2">
        <input v-model="form.active" type="checkbox" class="h-4 w-4 rounded border-primary-300 bg-surface text-accent-600 focus:ring-2 focus:ring-accent-500 dark:border-primary-600 dark:bg-surface-dark" />
        <span class="text-sm text-primary-700 dark:text-primary-200">Active — the user can sign in right away</span>
      </label>

      <p v-if="errorMessage" class="text-sm font-medium text-error-600 dark:text-error-400">{{ errorMessage }}</p>

      <div class="flex gap-3 pt-2">
        <button
          type="submit"
          :disabled="!canSubmit"
          class="inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
        >
          Create user
        </button>
        <button
          type="button"
          @click="router.push({ name: 'users' })"
          class="inline-flex items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 cursor-pointer dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
        >
          Cancel
        </button>
      </div>
    </form>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { api } from "src/api";
import PasswordRequirements from "src/components/PasswordRequirements.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { showNotificationToast } from "src/boot/mitt";

const router = useRouter();

const inputClasses =
  "h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600";

const form = reactive({
  firstName: "",
  lastName: "",
  username: "",
  email: "",
  copyrightTag: "",
  password: "",
  active: true,
});

type FormKey = "firstName" | "lastName" | "username" | "email" | "copyrightTag" | "password";
const textFields: { key: FormKey; label: string; type?: string; autocomplete?: string }[] = [
  { key: "firstName", label: "First name" },
  { key: "lastName", label: "Last name" },
  { key: "username", label: "Username", autocomplete: "off" },
  { key: "email", label: "Email", type: "email", autocomplete: "off" },
  { key: "copyrightTag", label: "Copyright tag" },
  { key: "password", label: "Initial password", type: "password", autocomplete: "new-password" },
];

const pwReqs = ref<any>(null);
const errorMessage = ref("");
const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const canSubmit = computed(() => {
  const reqs = Array.isArray(pwReqs.value) ? pwReqs.value[0] : pwReqs.value;
  return !!form.firstName && !!form.lastName && !!form.username && !!form.email && !!reqs?.allMet;
});

async function submit() {
  errorMessage.value = "";
  try {
    const user = await api.users.create({
      username: form.username,
      email: form.email,
      password: form.password,
      firstName: form.firstName,
      lastName: form.lastName,
      copyrightTag: form.copyrightTag || undefined,
      active: form.active,
      // The admin picks a temporary password; the user rotates it on first login.
      forcePasswordChange: true,
    });
    showNotificationToast({ headline: `User ${user.username} created`, type: "success" });
    router.push({ name: "user-general", params: { userid: user.id } });
  } catch (error: any) {
    if (error?.response?.status === 409) {
      errorMessage.value = "That username, email or name is already taken.";
      return;
    }
    if (error?.response?.status === 400) {
      errorMessage.value = error.response.data?.message ?? "The server rejected the input.";
      return;
    }
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}
</script>
