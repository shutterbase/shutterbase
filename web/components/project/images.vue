<template>
  <client-only>
    <div class="columns-1 md:columns-2 lg:columns-3">
      <div v-for="image in images" :key="image.id">
        <div class="relative mb-4 before:content-[''] before:rounded-md before:absolute before:inset-0 before:bg-black before:bg-opacity-20">
          <img :src="getImageThumbnailUrl(image)" class="w-full rounded-md" />
          <div class="absolute inset-0 p-8 text-white flex flex-col">
            <div class="relative">
              <a class="absolute inset-0" target="_blank" href="/"></a>
              <h1 class="text-md font-bold mb-3">{{ image.fileName }}</h1>
              <p class="font-sm font-light">{{ image.edges.createdBy.firstName }} {{ image.edges.createdBy.lastName }}</p>
            </div>
            <div class="mt-auto">
              <span class="bg-white bg-opacity-60 py-1 px-4 rounded-md text-black">#tag</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </client-only>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getFetchOptions, getDateTimeString, requestList, API_BASE_URL } from "~/api/common";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

function getImageThumbnailUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/thumb`;
}

const { items: images } = await requestList<Image>(`/projects/${props.projectId}/images`, getFetchOptions(Method.GET));
</script>
