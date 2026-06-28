<template>
  <main class="mx-auto w-full max-w-7xl">
    <div class="px-4 sm:px-6 lg:px-8">
      <p class="label-mono text-accent-600 dark:text-accent-400">Project</p>
      <h1 class="display mt-2 text-3xl text-primary-900 dark:text-white">Statistics</h1>
      <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">Tag usage across this project's images.</p>
    </div>
    <Table class="mt-6" dense :items="imageTagStatistics" :columns="imageTagColumns" name="" :allow-add="false"></Table>
  </main>
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>

<script setup lang="ts">
import { Ref, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { ImageTagsResponse } from "src/types/pocketbase";
import Table, { TableColumn, TableRowActionType } from "src/components/Table.vue";

import { api } from "src/api";
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
    const response = (await api.statistics.project(projectId)) as ProjectStatistics;
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
