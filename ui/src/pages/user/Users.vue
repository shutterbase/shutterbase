<template>
  <div class="mx-auto max-w-7xl w-full">
    <AlertBanner
      v-if="pendingCount > 0 && userStore.isAdmin()"
      :type="AlertBannerType.INFO"
      headline="Accounts waiting for activation"
      :message="`${pendingCount} signed-up ${pendingCount === 1 ? 'account is' : 'accounts are'} inactive and cannot sign in yet.`"
    />
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="relative max-w-md">
        <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-primary-400" />
        <input
          id="user-search"
          v-model="search"
          type="search"
          placeholder="Search by name, username or email"
          aria-label="Search users"
          class="h-10 w-full rounded-md border border-primary-200 bg-surface pl-9 pr-9 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
        />
        <button
          v-if="search"
          type="button"
          @click="search = ''"
          class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-primary-400 hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-200"
        >
          <XMarkIcon class="h-4 w-4" />
          <span class="sr-only">Clear search</span>
        </button>
      </div>
      <p v-if="search" class="label-mono mt-2 text-primary-500 dark:text-primary-400">
        <span class="font-data text-primary-700 dark:text-primary-200">{{ total }}</span> {{ total === 1 ? "match" : "matches" }}
      </p>
    </div>
    <Table
      dense
      :items="items"
      :columns="columns"
      name="User"
      :loading="loading"
      :allow-add="userStore.isAdmin()"
      :add-callback="() => router.push({ name: 'user-create' })"
    ></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <ModalMessage
      :show="showDeactivateDialog"
      :type="MessageType.CONFIRM_WARNING"
      @closed="showDeactivateDialog = false"
      headline="Deactivate user"
      :message="`Deactivate '${deactivateCandidate?.username}'? They lose access immediately — running sessions and API keys stop working.`"
      confirmText="Deactivate"
      @confirmed="confirmDeactivate"
    />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import AlertBanner, { AlertBannerType } from "src/components/AlertBanner.vue";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { api } from "src/api";
import { UsersResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import { useDebounceFn } from "@vueuse/core";
import { MagnifyingGlassIcon, XMarkIcon } from "@heroicons/vue/24/outline";
const router = useRouter();

const userStore = useUserStore();

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const showDeactivateDialog = ref(false);
const deactivateCandidate: Ref<UsersResponse | null> = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<UsersResponse[]> = ref([]);
const total = ref(0);
const loading = ref(false);

// Server-side search (§4.12 matches username, email, first and last name), so a
// hit is never limited to the page that happens to be loaded.
const search = ref("");

const pendingCount = computed(() => items.value.filter((u) => !u.active).length);

const columns: TableColumn<UsersResponse>[] = [
  { key: "username", label: "Username" },
  { key: "firstName", label: "First name" },
  { key: "lastName", label: "Last name" },
  { key: "copyrightTag", label: "CopyrightTag" },
  { key: "active", label: "Status", formatter: (active) => (active ? "Active" : "Inactive") },
  { key: ["role", "key"], label: "Role" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Details", callback: (item) => router.push({ name: `user-general`, params: { userid: item.id } }), type: TableRowActionType.EDIT },
      {
        key: "activate",
        label: "Activate",
        showCallback: (item) => userStore.isAdmin() && !item.active,
        callback: (item) => setActive(item, true),
        type: TableRowActionType.CUSTOM,
      },
      {
        key: "deactivate",
        label: "Deactivate",
        // Never offer self-deactivation: an admin locking themselves out needs
        // another admin (or a DB fix) to get back in.
        showCallback: (item) => userStore.isAdmin() && item.active && item.id !== userStore.user?.id,
        callback: (item) => {
          deactivateCandidate.value = item;
          showDeactivateDialog.value = true;
        },
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

// latest-wins guard: a fast typist outruns the network, so only the newest
// request may write the list.
let requestId = 0;

async function requestItems() {
  const myRequestId = ++requestId;
  loading.value = true;
  try {
    const resultList = await api.users.list({ limit: limit.value, offset: offset.value, search: search.value || undefined });
    if (myRequestId !== requestId) return;
    items.value = resultList.items;
    total.value = resultList.total;
  } catch (error: any) {
    if (myRequestId !== requestId) return;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  } finally {
    if (myRequestId === requestId) loading.value = false;
  }
}

async function setActive(item: UsersResponse, active: boolean) {
  try {
    const updated = await api.users.update(item.id, { active });
    Object.assign(item, updated);
    showNotificationToast({ headline: `User ${active ? "activated" : "deactivated"}`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function confirmDeactivate() {
  showDeactivateDialog.value = false;
  if (deactivateCandidate.value) {
    await setActive(deactivateCandidate.value, false);
    deactivateCandidate.value = null;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
watch(
  search,
  useDebounceFn(() => {
    offset.value = 0;
    requestItems();
  }, 300),
);
</script>
