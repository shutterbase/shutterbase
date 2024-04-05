<template>
  <div class="mx-auto max-w-7xl">
    <Table dense :items="items" :columns="columns" name="Cameras"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import pb from "src/boot/pocketbase";
import { CamerasResponse, UsersResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter, useRoute } from "vue-router";
const router = useRouter();
const route = useRoute();

const userStore = useUserStore();

type ITEM_TYPE = CamerasResponse;
const ITEM_COLLECTION = "cameras";
const ITEM_NAME = "camera";

const userId: string = `${route.params.userid}`;

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<ITEM_TYPE[]> = ref([]);
const columns: TableColumn<ITEM_TYPE>[] = [
  { key: "name", label: "Name" },
  {
    key: "actions",
    label: "Actions",
    actions: [
      { key: "edit", label: "Details", callback: (item) => router.push({ name: ITEM_NAME, params: { userid: userId, cameraid: item.id } }), type: TableRowActionType.EDIT },
    ],
  },
];

async function requestItems() {
  try {
    const resultList = await pb.collection<ITEM_TYPE>(ITEM_COLLECTION).getList(1, 50, {});
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
