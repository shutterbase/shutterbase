<template>
  <input type="button" value="Sync Tags" @click="syncTags" />
  <input type="text" v-model="search" @change="doRequest" />
  {{ images.length }}
  <p v-for="image in images" :key="image.id">
    {{ image.capturedAtCorrected }} - {{ image.computedFileName }}
    <span class="mx-2" v-for="assignment in image.tags" :key="assignment.id">{{ assignment.tag.name }}</span>
  </p>
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
