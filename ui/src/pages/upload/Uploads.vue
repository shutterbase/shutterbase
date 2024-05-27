<template>
  <div class="mx-auto max-w-7xl">
    <Table dense :items="items" :columns="columns" name="Upload" subtitle="" :add-callback="() => router.push('/uploads/create')"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import pb from "src/boot/pocketbase";
import { UploadsResponse, ProjectsResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
const router = useRouter();

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<UploadsResponse[]> = ref([]);
const columns: TableColumn<UploadsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "user", label: "User" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Edit", callback: (item) => router.push({ name: `upload-edit`, params: { id: item.id } }), type: TableRowActionType.EDIT },
      { key: "tagging", label: "Tagging", callback: (item) => router.push({ name: `upload-tagging`, params: { id: item.id } }), type: TableRowActionType.CUSTOM },
    ],
  },
];

async function requestItems() {
  try {
    const resultList = await pb.collection<UploadsResponse>("uploads").getList(1, 500, {});
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
