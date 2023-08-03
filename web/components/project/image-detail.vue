<template>
  <client-only>
    <div class="columns-1">
      <div v-if="currentImage">
        <ItemDescriptorLine :item="currentImage" />
        <div class="relative mb-4 before:content-[''] before:rounded-md before:absolute before:inset-0 before:bg-black before:bg-opacity-20">
          <img :src="getImageUrl(currentImage)" class="centerImage rounded-md" />
        </div>
      </div>
      <div v-else>
        <div class="text-center">No images available</div>
      </div>
      <div class="hidden lg:block" v-if="images">
        <div class="flex justify-evenly">
          <div v-for="(image, index) in images" :key="image.id" :class="`${currentImageOffset - offset === index ? 'border-8' : ''}`">
            <img :src="getImageThumbnailUrl(image)" class="h-20 rounded-md" @click="currentImage = image" />
          </div>
        </div>
      </div>
    </div>
  </client-only>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getFetchOptions, getDateTimeString, requestList, API_BASE_URL, ListResult, ListResponse } from "~/api/common";
import { emitter } from "~/boot/mitt";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

const limit = ref(11);
const offset = ref(0);
const totalImages = ref(0);
const currentImageOffset = ref(0);

const images = ref<Array<Image>>([]);
const currentImage = ref<Image | null>(null);

emitter.on("arrow-left", () => {
  if (currentImageOffset.value > 0) {
    console.log("currentImageOffset", currentImageOffset.value);
    currentImageOffset.value--;
    if (currentImageOffset.value >= 5) {
      offset.value = currentImageOffset.value - 5;
    } else {
      offset.value = 0;
    }
  }
});

emitter.on("arrow-right", () => {
  console.log("currentImageOffset", currentImageOffset.value);
  currentImageOffset.value++;
  if (currentImageOffset.value >= 5) {
    offset.value = currentImageOffset.value - 5;
  } else {
    offset.value = 0;
  }
});

function getImageThumbnailUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/thumb`;
}

function getImageUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/file`;
}

async function fetchImageList() {
  const url = `/projects/${props.projectId}/images?limit=${limit.value}&offset=${offset.value}`;
  const response = await useFetch(url, getFetchOptions(Method.GET));
  console.log(response.data);
  if (response.data.value) {
    const data = response.data.value as ListResponse<Image>;
    images.value = data.items;
    totalImages.value = data.total;
  }
}

watch(currentImageOffset, async () => {
  await fetchImageList();
  currentImage.value = images.value[currentImageOffset.value - offset.value];
});

await fetchImageList();
currentImage.value = images.value[0];
</script>
<style scoped>
.centerImage {
  max-width: 100%;
  max-height: 1000px;
  margin: 0 auto;
}
</style>
