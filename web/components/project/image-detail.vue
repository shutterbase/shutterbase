<template>
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
    <div class="hidden lg:block">
      <div class="flex overflow-x-scroll scroll-smooth scrollbar-hide" ref="filmstrip" v-if="images">
        <div v-for="(image, index) in images" :key="image.id" :class="`${currentImageOffset - offset === index ? 'selectedFilmstripItem' : ''} flex-shrink-0`">
          <img :src="getImageThumbnailUrl(image)" class="w-40 h-40 object-cover object-center rounded-sm m-1" @click="selectImage(image)" />
        </div>
      </div>
    </div>
  </div>
  <div>
    <link v-for="image in prefetchImages" :rel="`prefetch`" :href="getImageUrl(image)" />
  </div>
  <input type="checkbox" :checked="showTagPicker" id="tagPicker" class="modal-toggle" />
  <div class="modal">
    <form method="dialog" class="modal-box w-11/12 max-w-5xl">
      <h3 class="font-bold text-lg">Pick a tag to add</h3>
      <TagPicker :projectId="props.projectId" @selected="tagSelected" />
      <div class="modal-action">
        <!-- if there is a button, it will close the modal -->
        <button class="btn">Close</button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getFetchOptions, API_BASE_URL, ListResponse } from "~/api/common";
import { emitter } from "~/boot/mitt";
import { Tag } from "~/api/tag";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

const limit = ref(2000);
const offset = ref(0);
const totalImages = ref(0);
const currentImageOffset = ref(0);

const images = ref<Array<Image>>([]);
const currentImage = ref<Image | null>(null);

const prefetchImages = ref<Array<Image>>([]);

const filmstrip = ref<HTMLElement | null>(null);

const showTagPicker = ref(false);

function previousImage() {
  if (showTagPicker.value) return;
  if (currentImageOffset.value > 0) {
    console.log("currentImageOffset", currentImageOffset.value);
    currentImageOffset.value--;
    if (currentImageOffset.value >= 5) {
      // offset.value = currentImageOffset.value - 5;
    } else {
      // offset.value = 0;
    }
  }
}

function nextImage() {
  if (showTagPicker.value) return;
  console.log("currentImageOffset", currentImageOffset.value);
  currentImageOffset.value++;
}

emitter.on("key-ArrowLeft", previousImage);
emitter.on("key-h", previousImage);

emitter.on("key-ArrowRight", nextImage);
emitter.on("key-l", nextImage);

emitter.on("key-t", (event: any) => {
  if (showTagPicker.value) return;
  event.preventDefault();
  console.log("showing tag picker");
  emitter.emit("display-tag-picker", event);
  showTagPicker.value = true;
});

emitter.on("key-Escape", () => {
  console.log("hiding tag picker");
  showTagPicker.value = false;
});

function getImageThumbnailUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/thumb?size=200`;
}

function getImageUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/file?size=1500`;
}

async function fetchImageList() {
  const url = `/projects/${props.projectId}/images?limit=${limit.value}&offset=${offset.value}`;
  const response = await useFetch(url, getFetchOptions(Method.GET));
  if (response.data.value) {
    const data = response.data.value as ListResponse<Image>;
    images.value = data.items;
    totalImages.value = data.total;
  }
  calculatePrefetchImages();
}

function selectImage(image: Image) {
  currentImage.value = image;
  currentImageOffset.value = images.value.indexOf(image) + offset.value;
}

async function tagSelected(tag: Tag) {
  showTagPicker.value = false;
  console.log(`adding tag ${tag.name} to image ${currentImage.value?.id}`);
  /* if (currentImage) {
    const url = `/projects/${props.projectId}/images/${currentImage.id}/tags/${tagId}`;
    const response = await useFetch(url, getFetchOptions(Method.PUT));
    console.log(response.data);
    if (response.data.value) {
      const data = response.data.value as Image;
      currentImage.value = data;
    }
  } */
}

function calculatePrefetchImages() {
  const prefetchStart = Math.max(0, currentImageOffset.value - 10);
  const prefetchEnd = Math.min(images.value.length, currentImageOffset.value + 10);
  prefetchImages.value = images.value.slice(prefetchStart, prefetchEnd);
}

watch(currentImageOffset, async () => {
  // await fetchImageList();
  currentImage.value = images.value[currentImageOffset.value - offset.value];
  if (filmstrip.value) {
    if (currentImageOffset.value > 4) {
      filmstrip.value.scrollLeft = (currentImageOffset.value - 4) * 168;
    } else {
      filmstrip.value.scrollLeft = 0;
    }
  }
  calculatePrefetchImages();
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
.selectedFilmstripItem {
  -webkit-box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
  -moz-box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
  box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
}
</style>
