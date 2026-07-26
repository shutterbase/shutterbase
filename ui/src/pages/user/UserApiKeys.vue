<template>
  <main class="py-8">
    <div class="sm:flex sm:items-end sm:justify-between">
      <div class="sm:flex-auto">
        <p class="label-mono text-accent-600 dark:text-accent-400">API keys</p>
        <h1 class="display mt-2 text-2xl text-primary-900 dark:text-white">
          <span v-if="isSelf">Your API keys</span>
          <span v-else>API keys of {{ ownerName }}</span>
        </h1>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
          Use a key to talk to the API from scripts and tools (the downloader, CI) instead of a browser session. Send it as
          <code class="font-data text-xs">Authorization: Bearer &lt;key&gt;</code>. A key acts as you — it has exactly your permissions.
        </p>
      </div>
      <div class="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
        <button
          v-if="canManage"
          id="createApiKey"
          @click="startCreate"
          type="button"
          class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:opacity-50 dark:focus-visible:ring-offset-primary-950"
        >
          Create API key
        </button>
      </div>
    </div>

    <div class="my-8 border-t border-primary-200 dark:border-primary-800"></div>

    <!-- Inline rather than a modal: it is one field, and the minted secret has to
         land on this page anyway. -->
    <form v-if="showCreateForm" class="mb-8 flex flex-wrap items-end gap-3 rounded-lg border border-primary-200 p-4 dark:border-primary-800" @submit.prevent="createKey">
      <div class="min-w-0 flex-1">
        <label for="apiKeyName" class="block text-sm font-medium text-primary-700 dark:text-primary-200">Name</label>
        <p class="mb-1.5 mt-0.5 text-xs text-primary-400">Name it after where it will be used — that is all you will have to recognise it by later.</p>
        <input
          id="apiKeyName"
          v-model="newKeyName"
          type="text"
          placeholder="e.g. downloader on my laptop"
          class="block w-full rounded-md border-0 py-1.5 text-primary-900 shadow-sm ring-1 ring-inset ring-primary-300 placeholder:text-primary-400 focus:ring-2 focus:ring-inset focus:ring-accent-500 dark:bg-primary-900 dark:text-white dark:ring-primary-700 sm:text-sm"
        />
      </div>
      <div class="flex gap-2">
        <button
          id="submitApiKey"
          type="submit"
          :disabled="creating"
          class="inline-flex h-9 cursor-pointer items-center rounded-md bg-accent-600 px-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {{ creating ? "Creating…" : "Create" }}
        </button>
        <button
          type="button"
          class="inline-flex h-9 cursor-pointer items-center rounded-md border border-primary-200 px-3 text-sm font-medium text-primary-700 transition-colors hover:bg-primary-100 dark:border-primary-700 dark:text-primary-200 dark:hover:bg-primary-800"
          @click="showCreateForm = false"
        >
          Cancel
        </button>
      </div>
    </form>

    <!-- The secret exists in exactly one response and is never recoverable, so it
         gets a loud, deliberately dismissable panel rather than a toast. -->
    <div v-if="mintedToken" data-testid="minted-token" class="mb-8 rounded-lg border border-accent-500 bg-accent-500/5 p-4">
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <h2 class="text-sm font-semibold text-primary-900 dark:text-white">Copy this key now</h2>
          <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">It is shown once and stored only as a hash. If you lose it, revoke the key and create a new one.</p>
          <div class="mt-3 flex items-center gap-2">
            <code class="font-data min-w-0 flex-1 truncate rounded-md bg-primary-100 px-3 py-2 text-sm text-primary-900 dark:bg-primary-900 dark:text-primary-100">{{
              mintedToken
            }}</code>
            <Clipboard class="h-5 shrink-0" :text="mintedToken" />
          </div>
        </div>
        <button
          type="button"
          aria-label="Dismiss the new key"
          class="shrink-0 rounded-md p-1 text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800"
          @click="mintedToken = null"
        >
          <XMarkIcon class="h-5 w-5" />
        </button>
      </div>
    </div>

    <!-- The list endpoint scopes to the caller unless they are an admin, so
         rendering it here would show YOUR keys under someone else's name. -->
    <div v-if="!canManage" class="rounded-md border border-dashed border-primary-200 px-4 py-10 text-center text-sm text-primary-400 dark:border-primary-700">
      You can only see your own API keys.
    </div>

    <div v-else-if="keys.length === 0" class="rounded-md border border-dashed border-primary-200 px-4 py-10 text-center text-sm text-primary-400 dark:border-primary-700">
      No API keys yet.
    </div>

    <ul v-else-if="canManage" class="divide-y divide-primary-100 rounded-lg border border-primary-200 dark:divide-primary-800/60 dark:border-primary-800">
      <li v-for="key in keys" :key="key.id" class="flex flex-wrap items-center justify-between gap-3 px-4 py-3" :data-testid="`api-key-${key.keyId}`">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="truncate text-sm font-medium text-primary-900 dark:text-white">{{ key.name }}</span>
            <span :class="['rounded-full px-2 py-0.5 text-[10px] font-medium', badgeClass(key)]">{{ keyStatus(key) }}</span>
          </div>
          <p class="font-data mt-0.5 text-xs text-primary-400">
            {{ key.keyId }} · created {{ dateTimeFromBackend(key.createdAt) }}
            <template v-if="key.lastUsedAt"> · last used {{ dateTimeFromBackend(key.lastUsedAt) }}</template>
          </p>
        </div>
        <button
          v-if="canManage && !key.revoked"
          type="button"
          :aria-label="`Revoke ${key.name}`"
          class="shrink-0 cursor-pointer rounded-md border border-primary-200 px-2.5 py-1 text-xs font-medium text-red-600 transition-colors hover:border-red-300 hover:bg-red-500/10 dark:border-primary-700 dark:text-red-400"
          @click="revokeKey(key)"
        >
          Revoke
        </button>
      </li>
    </ul>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { XMarkIcon } from "@heroicons/vue/24/outline";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "src/api";
import { ApiKey, User } from "src/types/api";
import { useUserStore } from "src/stores/user-store";
import { showNotificationToast } from "src/boot/mitt";
import { dateTimeFromBackend } from "src/util/dateTimeUtil";
import { canManageApiKeys, keyStatus, sortApiKeys } from "src/util/apiKeys";
import Clipboard from "src/components/Clipboard.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";

const route = useRoute();
const userStore = useUserStore();

const userId = computed(() => `${route.params.userid}`);
const isSelf = computed(() => userId.value === userStore.user?.id);
const canManage = computed(() => canManageApiKeys(userStore.user ? { id: userStore.user.id, isAdmin: userStore.isAdmin() } : null, userId.value));

const keys = ref<ApiKey[]>([]);
const owner = ref<User | null>(null);
const ownerName = computed(() => (owner.value ? `${owner.value.firstName} ${owner.value.lastName}` : "this user"));

const mintedToken = ref<string | null>(null);
const showCreateForm = ref(false);
const creating = ref(false);
const newKeyName = ref("");

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

function badgeClass(key: ApiKey): string {
  if (key.revoked) return "bg-primary-100 text-primary-500 dark:bg-primary-800 dark:text-primary-300";
  if (key.lastUsedAt) return "bg-green-500/15 text-green-700 dark:text-green-300";
  return "bg-amber-500/15 text-amber-700 dark:text-amber-300";
}

async function loadData() {
  // Nothing to fetch when the viewer may not manage these keys: the list would
  // be their own, and the owner lookup would 403 for a non-admin anyway.
  if (!canManage.value) return;
  try {
    const res = await api.apiKeys.list({ userId: userId.value, limit: 200 });
    keys.value = sortApiKeys(res.items);
    if (!isSelf.value) owner.value = await api.users.get(userId.value);
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function startCreate() {
  newKeyName.value = "";
  showCreateForm.value = true;
}

async function createKey() {
  const name = newKeyName.value.trim();
  if (name === "") {
    showNotificationToast({ headline: "A name is required", type: "error" });
    return;
  }
  creating.value = true;
  try {
    const created = await api.apiKeys.create({ name, userId: userId.value });
    showCreateForm.value = false;
    // Keep the secret from the create response — no later call ever returns it.
    mintedToken.value = created.token;
    keys.value = sortApiKeys([created, ...keys.value]);
    showNotificationToast({ headline: "API key created", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    creating.value = false;
  }
}

async function revokeKey(key: ApiKey) {
  try {
    await api.apiKeys.revoke(key.id);
    // Revoked keys stay listed (the server keeps the row) — reflect the new state
    // rather than dropping the row, so the list matches a reload.
    keys.value = sortApiKeys(keys.value.map((k) => (k.id === key.id ? { ...k, revoked: true } : k)));
    showNotificationToast({ headline: "API key revoked", type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

watch(userId, loadData);
onMounted(loadData);
</script>
