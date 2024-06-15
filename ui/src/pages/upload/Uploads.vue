<template>
  <div class="mx-auto max-w-7xl">
    <Table dense :items="items" :columns="columns" name="Upload" subtitle="" :add-callback="() => router.push('/uploads/create')"></Table>
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
import pb from "src/boot/pocketbase";
import { UploadsResponse, ProjectsResponse, UsersResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
import ModalMessage, { MessageType } from "src/components/ModalMessage.vue";
import { showNotificationToast } from "src/boot/mitt";
import { U } from "app/dist/spa/assets/UnexpectedErrorMessage-BfdH_7q6";
const router = useRouter();

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);
const showDeleteDialog = ref(false);
const deleteCandidate: Ref<UploadsResponse | null> = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<UploadsResponse[]> = ref([]);
const columns: TableColumn<UploadsResponse>[] = [
  { key: "name", label: "Name" },
  { key: ["expand", "user"], label: "User", formatter: (user) => `${user.firstName} ${user.lastName}` },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Edit", callback: (item) => router.push({ name: `upload-edit`, params: { id: item.id } }), type: TableRowActionType.EDIT },
      { key: "tagging", label: "Tagging", callback: (item) => router.push({ name: `upload-tagging`, params: { id: item.id } }), type: TableRowActionType.CUSTOM },
      {
        key: "delete",
        label: "Delete",
        callback: deleteItem,
        type: TableRowActionType.DELETE,
      },
    ],
  },
];

async function requestItems() {
  try {
    const resultList = await pb.collection<UploadsResponse>("uploads").getList(1, 500, {
      filter: `(project='${activeProjectId.value}')`,
      expand: "user",
    });
    items.value = resultList.items;
  } catch (error: any) {
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
    await pb.collection<UploadsResponse>("uploads").delete(item.id);
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
