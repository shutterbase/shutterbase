<template>
  <div class="mx-auto max-w-7xl w-full">
    <!-- One toolbar owns the title and every action, so the board and the list
         get the same controls and nothing stacks up against the table's own
         header (Table renders with :hide-header). -->
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="min-w-0">
          <h1 class="display text-3xl text-primary-900 dark:text-white">Uploads</h1>
          <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
            {{ reviewEnabled ? "Tag an upload, then hand it to a project admin for review." : "Every upload in this project." }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <!-- Photographer filter. Only reviewers ever need it: the API forces a
               photographer's list to their own uploads, so for them it would be
               a control over a set of one. -->
          <SearchSelect
            v-if="canSeeOthers"
            id="photographer-filter"
            v-model="photographerFilter"
            aria-label="Filter by photographer"
            placeholder="All photographers"
            empty-text="No photographer matches"
            :options="photographerOptions"
          />

          <!-- The kanban only exists where the state flow does; without upload
               reviews every upload sits in "open" and the board would be one column. -->
          <div v-if="reviewEnabled" class="flex rounded-lg border border-primary-200 bg-surface p-0.5 dark:border-primary-700 dark:bg-surface-dark">
            <button
              v-for="option in viewOptions"
              :key="option.value"
              type="button"
              @click="view = option.value"
              :class="[
                'inline-flex h-8 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors cursor-pointer',
                view === option.value
                  ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
                  : 'text-primary-500 hover:bg-primary-100 hover:text-primary-700 dark:text-primary-400 dark:hover:bg-primary-800 dark:hover:text-primary-200',
              ]"
            >
              <component :is="option.icon" class="h-4 w-4" />
              {{ option.label }}
            </button>
          </div>

          <button
            v-if="canCreate"
            type="button"
            @click="createUpload"
            class="inline-flex h-9 cursor-pointer items-center gap-1.5 rounded-md bg-accent-600 px-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950"
          >
            <PlusIcon class="h-4 w-4" />
            Add Upload
          </button>
        </div>
      </div>
    </div>

    <UploadKanban
      v-if="reviewEnabled && view === 'kanban'"
      :uploads="visibleItems"
      :is-reviewer="userStore.isProjectAdminOrHigher()"
      :current-user-id="userStore.user?.id"
      :can-create="canCreate"
      @create="createUpload"
      @move="moveUpload"
      @open="(item) => router.push({ name: `upload-edit`, params: { id: item.id } })"
    />

    <Table v-else dense hide-header :items="visibleItems" :columns="columns" name="Upload"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
    <ModalMessage
      :show="showDeleteDialog"
      :type="MessageType.CONFIRM_WARNING"
      @closed="showDeleteDialog = false"
      headline="Delete Upload"
      :message="`Are you sure you want to delete upload '${deleteCandidate?.name}'?`"
      @confirmed="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import UploadKanban from "src/components/upload/UploadKanban.vue";
import SearchSelect, { SearchSelectOption } from "src/components/SearchSelect.vue";
import { api } from "src/api";
import { UploadsResponse } from "src/types/pocketbase";
import { UploadState } from "src/types/api";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import { useStorage } from "@vueuse/core";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { Bars3Icon, PlusIcon, ViewColumnsIcon } from "@heroicons/vue/24/outline";
import { isUploadReadOnly, showUploadEdit } from "./uploadUtil";
import { UPLOAD_STATE_LABEL } from "src/util/uploadReview";

const router = useRouter();

const userStore = useUserStore();
const { activeProjectId, activeProject } = storeToRefs(userStore);

const reviewEnabled = computed(() => !!activeProject.value?.uploadReviewEnabled);

// The API scopes a photographer's list to their own uploads (§4.9), so only a
// reviewer ever has more than one photographer on screen.
const canSeeOthers = computed(() => userStore.isProjectAdminOrHigher());
const canCreate = computed(() => userStore.isProjectEditorOrHigher());

function createUpload() {
  router.push("/uploads/create");
}

const view = useStorage<"table" | "kanban">("upload-view", "kanban");
const viewOptions = [
  { value: "kanban" as const, label: "Board", icon: ViewColumnsIcon },
  { value: "table" as const, label: "List", icon: Bars3Icon },
];

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const showDeleteDialog = ref(false);
const deleteCandidate: Ref<UploadsResponse | null> = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<UploadsResponse[]> = ref([]);

const photographerFilter = ref("");

// Derived from what actually loaded, so the picker never offers an empty result.
// The empty value is a real option ("All photographers") — picking it clears.
const photographerOptions = computed<SearchSelectOption[]>(() => {
  const seen = new Map<string, SearchSelectOption>();
  for (const item of items.value) {
    const u = item.user;
    if (!u?.id || seen.has(u.id)) continue;
    seen.set(u.id, {
      value: u.id,
      label: `${u.firstName} ${u.lastName}${u.id === userStore.user?.id ? " (you)" : ""}`,
      hint: u.copyrightTag || undefined,
    });
  }
  const people = [...seen.values()].sort((a, b) => a.label.localeCompare(b.label));
  return [{ value: "", label: "All photographers" }, ...people];
});

const visibleItems = computed(() => (photographerFilter.value ? items.value.filter((i) => i.user?.id === photographerFilter.value) : items.value));
const columns: TableColumn<UploadsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "user", label: "User", formatter: (user) => `${user.firstName} ${user.lastName}` },
  { key: "state", label: "State", formatter: (state) => UPLOAD_STATE_LABEL[(state ?? "open") as UploadState] },
  {
    key: "actions",
    label: "Actions",
    actions: [
      {
        key: "show",
        label: "Show",
        showCallback: isUploadReadOnly,
        callback: (item) => router.push({ name: `upload-edit`, params: { id: item.id } }),
        type: TableRowActionType.CUSTOM,
      },
      {
        key: "edit",
        label: "Edit",
        showCallback: showUploadEdit,
        callback: (item) => router.push({ name: `upload-edit`, params: { id: item.id } }),
        type: TableRowActionType.EDIT,
      },
      {
        key: "delete",
        label: "Delete",
        showCallback: showUploadEdit,
        callback: deleteItem,
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

// Most recently edited first — the sort also decides WHICH uploads make the
// 500-row window on a long-running project, so it belongs on the query too and
// not only in the board's own ordering.
async function requestItems() {
  try {
    const resultList = await api.uploads.list({ projectId: activeProjectId.value, limit: 500, sort: "updatedAt", order: "desc" });
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

async function moveUpload(item: UploadsResponse, state: UploadState) {
  const previous = item.state;
  item.state = state; // optimistic: the board should not lag the drop
  try {
    const updated = await api.uploads.update(item.id, { state });
    Object.assign(item, updated);
    showNotificationToast({ headline: `Upload moved to '${UPLOAD_STATE_LABEL[state]}'`, type: "success" });
  } catch (error: any) {
    item.state = previous;
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function deleteItem(item: UploadsResponse) {
  deleteCandidate.value = item;
  showDeleteDialog.value = true;
}

async function confirmDelete() {
  if (!deleteCandidate.value) {
    return;
  }
  const item = deleteCandidate.value;
  try {
    await api.uploads.remove(item.id);
    items.value = items.value.filter((i) => i.id !== item.id);
    deleteCandidate.value = null;
    showDeleteDialog.value = false;
    showNotificationToast({ headline: `Upload deleted`, type: "success" });
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
