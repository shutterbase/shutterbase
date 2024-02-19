<template>
  <div class="mx-auto max-w-7xl">
    <Table :items="items" :columns="columns" name="Project"></Table>
  </div>
</template>

<script setup lang="ts">
import { Ref, onMounted, ref } from "vue";
import Table, { TableColumn, TableRowAction, TableRowActionType } from "src/components/Table.vue";
import pb from "src/boot/pocketbase";
import { ProjectsResponse } from "src/types/pocketbase";

const limit = ref(50);
const offset = ref(0);
const items: Ref<ProjectsResponse[]> = ref([]);
const columns: TableColumn<ProjectsResponse>[] = [
  { key: "name", label: "Name" },
  { key: "description", label: "Description" },
  {
    key: "actions",
    label: "Actions",
    actions: [{ key: "edit", label: "Edit", callback: (item) => console.log(`Edit: ${item.name}`), type: TableRowActionType.EDIT } as TableRowAction<ProjectsResponse>],
  },
];

onMounted(async () => {
  const resultList = await pb.collection<ProjectsResponse>("projects").getList(1, 50, {});
  items.value = resultList.items;
  resultList.items[0].id;
});
</script>
src/types/pocketbase-generated
