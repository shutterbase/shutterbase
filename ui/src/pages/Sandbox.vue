<template>
  <main class="mx-auto w-full max-w-3xl space-y-6 px-4 py-12 sm:px-6 lg:px-8">
    <p class="label-mono text-primary-500 dark:text-primary-400">Dev sandbox</p>
    <div class="flex flex-wrap items-center gap-3">
      <input
        type="button"
        value="Sync Tags"
        @click="syncTags"
        class="inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white"
      />
      <input
        type="text"
        v-model="search"
        @change="doRequest"
        placeholder="Search…"
        class="h-10 w-full max-w-xs rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
      />
      <span class="label-mono font-data text-primary-500 dark:text-primary-400">{{ images.length }}</span>
    </div>
    <p class="flex flex-wrap items-center gap-2 text-sm text-primary-700 dark:text-primary-300" v-for="image in images" :key="image.id">
      <span class="font-data text-primary-500 dark:text-primary-400">{{ image.capturedAtCorrected }} - {{ image.computedFileName }}</span>
      <span
        class="inline-flex items-center gap-1 rounded-md border border-primary-200 bg-surface px-2 py-0.5 text-xs font-medium text-primary-700 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200"
        v-for="assignment in image.tags"
        :key="assignment.id"
        >{{ assignment.tag.name }}</span
      >
    </p>
  </main>
</template>

<script setup lang="ts">
import { Ref, ref } from "vue";
import { api } from "src/api";
import { useUserStore } from "src/stores/user-store";
import { ImageWithTagsType } from "src/types/custom";

const userStore = useUserStore();
const search: Ref<string> = ref("");
const images: Ref<ImageWithTagsType[]> = ref([]);

async function doRequest() {
  if (!userStore.activeProjectId) return;
  const result = await api.images.list({
    projectId: userStore.activeProjectId,
    search: search.value || undefined,
    sort: "capturedAtCorrected",
    order: "desc",
    limit: 500,
  });
  images.value = result.items;
}

async function syncTags() {
  await api.statistics.syncImageTags();
}
</script>
