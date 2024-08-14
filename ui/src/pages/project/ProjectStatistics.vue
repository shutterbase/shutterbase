<template>
  <main class="px-4 sm:px-6 lg:flex-auto lg:px-0 py-4">
    <div class="mx-auto max-w-2xl lg:mx-0 lg:max-w-none">
      <div class="pb-2">
        <h2 class="text-2xl font-semibold leading-7 text-gray-900 dark:text-primary-200">Project Statistics</h2>
        <Table dense :items="imageTagStatistics" :columns="imageTagColumns" name="" :allow-add="false"></Table>
      </div>
    </div>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ImageTagsResponse } from "src/types/pocketbase";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";

import pb from "src/boot/pocketbase";
const route = useRoute();

type ProjectStatistics = {
  tags: ImageTagWithCount[];
};

type ImageTagWithCount = ImageTagsResponse & { count: number };

const imageTagStatistics: Ref<ImageTagWithCount[]> = ref([]);

const showUnexpectedErrorMessage = ref(false);
const unexpectedError = ref(null);

async function loadData() {
  const projectId: string = `${route.params.id}`;
  if (!projectId || projectId === "") {
    console.log("No project ID provided");
    return;
  }

  try {
    console.log(`Loading project statistics for  ${projectId}`);
    const response = await pb.send<ProjectStatistics>(`/api/statistics/${projectId}`, {});
    console.log(response);
    response.tags.sort((a, b) => b.count - a.count);
    response.tags = response.tags.map(trimTagDescription);
    imageTagStatistics.value = response.tags;
  } catch (error: any) {
    unexpectedError.value = error;
    showUnexpectedErrorMessage.value = true;
  }
}

function trimTagDescription(tag: ImageTagWithCount): ImageTagWithCount {
  let description = tag.description;
  if (description && description.length > 50) {
    description = `${description.substring(0, 47)}...`;
  }
  return {
    ...tag,
    description: description,
  };
}

const imageTagColumns: TableColumn<ImageTagWithCount>[] = [
  { key: "name", label: "Name" },
  { key: "description", label: "Description" },
  { key: "type", label: "Type" },
  { key: "count", label: "Count" },
];

watch(route, loadData);
onMounted(loadData);
</script>
