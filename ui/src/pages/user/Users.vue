<template>
  <div class="mx-auto max-w-7xl w-full">
    <Table dense :items="items" :columns="columns" name="Users"></Table>
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { Ref, computed, onMounted, ref, watch } from "vue";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";
import pb from "src/boot/pocketbase";
import { UsersResponse } from "src/types/pocketbase";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";
const router = useRouter();

const userStore = useUserStore();

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

const limit = ref(50);
const offset = ref(0);
const items: Ref<UsersResponse[]> = ref([]);
const columns: TableColumn<UsersResponse>[] = [
  { key: "username", label: "Username" },
  { key: "firstName", label: "First name" },
  { key: "lastName", label: "Last name" },
  { key: "copyrightTag", label: "CopyrightTag" },
  {
    key: "actions",
    label: "Actions",
    actions: [{ key: "edit", label: "Details", callback: (item) => router.push({ name: `user-general`, params: { userid: item.id } }), type: TableRowActionType.EDIT }],
  },
];

async function requestItems() {
  try {
    const resultList = await pb.collection<UsersResponse>("users").getList(1, 50, {});
    items.value = resultList.items;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

onMounted(requestItems);
watch([limit, offset], requestItems);
</script>
