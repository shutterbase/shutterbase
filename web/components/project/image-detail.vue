<template>
  <ClientOnly>
    <div class="" style="max-height: 75%">
      <div v-if="images[currentImageOffset]" style="max-height: 75%">
        <ItemDescriptorLine :item="images[currentImageOffset]" />
        <div class="divider"></div>
        <DetailTagHeader :image="images[currentImageOffset]" :projectId="projectId" @tag-picker-state="setTagPickerState" @image-update="updateImage"></DetailTagHeader>
        <div class="divider"></div>
        <div class="" style="max-height: 50%">
          <img :src="getImageUrl(images[currentImageOffset])" class="centerImage rounded-md" />
        </div>
      </div>
      <div v-else>
        <div class="text-center">No images available</div>
      </div>
      <div class="hidden lg:block">
        <div class="flex overflow-x-scroll scrollbar-hide" ref="filmstrip" v-if="images">
          <div v-for="(image, index) in images" :key="image.id" :class="`${currentImageOffset - offset === index ? 'selectedFilmstripItem' : ''} flex-shrink-0`">
            <img :src="getImageThumbnailUrl(image)" class="filmstrip h-40 w-40 object-cover object-center rounded-sm m-1" @click="selectImage(image)" />
          </div>
        </div>
      </div>
    </div>
    <div>
      <link v-for="image in prefetchImages" :rel="`prefetch`" :href="getImageUrl(image)" />
    </div>
  </ClientOnly>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getFetchOptions, API_BASE_URL, ListResponse } from "~/api/common";
import { emitter } from "~/boot/mitt";

const router = useRouter();
const route = useRoute();

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
const prefetchImages = ref<Array<Image>>([]);
const filmstrip = ref<HTMLElement | null>(null);

const tagPickerOpen = ref(false);

function previousImage() {
  if (tagPickerOpen.value) return;
  if (currentImageOffset.value > 0) {
    currentImageOffset.value--;
  }
}

function nextImage() {
  if (tagPickerOpen.value) return;
  if (currentImageOffset.value < totalImages.value - 1) {
    currentImageOffset.value++;
  }
}

function setTagPickerState(arg: boolean) {
  tagPickerOpen.value = arg;
}

emitter.on("key-ArrowLeft", previousImage);
emitter.on("key-h", previousImage);

emitter.on("key-ArrowRight", nextImage);
emitter.on("key-l", nextImage);

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
  updateDisplayedImage();
}

async function fetchCurrentImage() {
  if (images.value[currentImageOffset.value]) {
    const url = `/projects/${props.projectId}/images/${images.value[currentImageOffset.value].id}`;
    const response = await useFetch(url, getFetchOptions(Method.GET));
    if (response.data.value) {
      const data = response.data.value as Image;
      const index = getImageIndex(data);
      images.value[index] = data;
    }
  }
}

function selectImage(image: Image) {
  currentImageOffset.value = images.value.indexOf(image);
}

function getImageIndex(image: Image): number {
  const index = images.value.findIndex((i) => i.id === image.id);
  return index;
}

function updateImage(updatedImage: Image) {
  const imageIndex = getImageIndex(updatedImage);
  images.value[imageIndex] = updatedImage;
}

function calculatePrefetchImages() {
  const prefetchStart = Math.max(0, currentImageOffset.value - 10);
  const prefetchEnd = Math.min(images.value.length, currentImageOffset.value + 10);
  prefetchImages.value = images.value.slice(prefetchStart, prefetchEnd);
}

let filmstripScrollDebounceTimeout: any = null;
const initialFilmStripUpdate = ref(true);
function updateFilmstripScroll() {
  const getTargetOffset = () => {
    if (filmstrip.value) {
      if (currentImageOffset.value > 4) {
        return (currentImageOffset.value - 4) * 168;
      } else {
        return 0;
      }
    }
  };
  if (filmstripScrollDebounceTimeout) {
    clearTimeout(filmstripScrollDebounceTimeout);
  }
  filmstripScrollDebounceTimeout = setTimeout(() => {
    if (filmstrip.value) {
      filmstrip.value.scrollTo({ left: getTargetOffset(), behavior: initialFilmStripUpdate.value ? "auto" : "smooth" });
    }
    initialFilmStripUpdate.value = false;
  }, 100);
}

function updateUrl() {
  if (typeof currentImageOffset.value !== "undefined" && currentImageOffset.value !== null && images.value[currentImageOffset.value]) {
    const newImageId = images.value[currentImageOffset.value].id;
    router.push({ query: { image: newImageId } });
  }
}

function updateImageFromUrl() {
  if (router.currentRoute.value.query.image) {
    const imageId = router.currentRoute.value.query.image as string;
    const image = images.value.find((image) => image.id === imageId);
    if (image) {
      currentImageOffset.value = images.value.indexOf(image);
      updateDisplayedImage();
    }
  }
}

async function updateDisplayedImage() {
  fetchCurrentImage();
  updateFilmstripScroll();
  calculatePrefetchImages();
}

await fetchImageList();

watch(currentImageOffset, updateUrl, { immediate: true });
watch(() => route.fullPath, updateImageFromUrl, { immediate: true });
</script>
<style scoped>
.centerImage {
  max-height: 50vh;
  margin: 0 auto;
}
.filmstrip {
  max-height: 10vh;
}
.selectedFilmstripItem {
  -webkit-box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
  -moz-box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
  box-shadow: inset 0px 0px 0px 10px rgb(80, 80, 80);
}
</style>
