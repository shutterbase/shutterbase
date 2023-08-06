<template>
  <ClientOnly>
    <div class="" style="max-height: 75%">
      <div v-if="images[currentImageOffset]" style="max-height: 75%">
        <ItemDescriptorLine :item="images[currentImageOffset]" />
        <div class="divider"></div>
        <div v-if="images[currentImageOffset].edges.tags && images[currentImageOffset].edges.tags.length !== 0" class="flex flex-row">
          <div class="btn btn-xs" @click="openTagPicker">Add Tags</div>
          <div
            v-for="tag in images[currentImageOffset].edges.tags"
            class="badge badge-primary object-center p-3 ml-2 hover click hover:cursor-pointer"
            @click="requestRemoveTag(tag)"
          >
            {{ tag.name }}
          </div>
        </div>
        <div v-else>
          <div class="btn btn-xs" @click="openTagPicker">Add Tags</div>
          No tags applied
        </div>
        <div class="divider"></div>
        <div class="" style="max-height: 50%">
          <img :src="getImageUrl(images[currentImageOffset])" class="centerImage rounded-md" />
        </div>
      </div>
      <div v-else>
        <div class="text-center">No images available</div>
      </div>
      <div class="hidden lg:block">
        <div class="flex overflow-x-scroll scroll-smooth scrollbar-hide" ref="filmstrip" v-if="images">
          <div v-for="(image, index) in images" :key="image.id" :class="`${currentImageOffset - offset === index ? 'selectedFilmstripItem' : ''} flex-shrink-0`">
            <img :src="getImageThumbnailUrl(image)" class="filmstrip h-40 w-40 object-cover object-center rounded-sm m-1" @click="selectImage(image)" />
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
        <TagPicker :projectId="props.projectId" :active="showTagPicker" @selected="tagSelected" />
        <div class="modal-action">
          <!-- if there is a button, it will close the modal -->
          <button class="btn">Close</button>
        </div>
      </form>
    </div>
    <input type="checkbox" id="removeTagDialog" :checked="showRemoveTagDialog" class="modal-toggle" />
    <div class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg">Remove Tag</h3>
        <p class="py-4">Remove tag {{ removeTagCandidate?.name }} from this image</p>
        <div class="modal-action">
          <label class="btn" @click="showRemoveTagDialog = false">Cancel</label>
          <label class="btn" @click="removeTag">OK</label>
        </div>
      </div>
    </div>
  </ClientOnly>
</template>

<script setup lang="ts">
import { ref, Ref } from "vue";
import { Image } from "~/api/image";
import { Method, getFetchOptions, API_BASE_URL, ListResponse } from "~/api/common";
import { emitter } from "~/boot/mitt";
import { Tag } from "~/api/tag";
import { useStore } from "~/stores/store";

const store = useStore();

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

const showTagPicker = ref(false);
const showRemoveTagDialog = ref(false);

function previousImage() {
  if (showTagPicker.value) return;
  if (currentImageOffset.value > 0) {
    currentImageOffset.value--;
  }
}

function nextImage() {
  if (showTagPicker.value) return;
  if (currentImageOffset.value < totalImages.value - 1) {
    currentImageOffset.value++;
  }
}

emitter.on("key-ArrowLeft", previousImage);
emitter.on("key-h", previousImage);

emitter.on("key-ArrowRight", nextImage);
emitter.on("key-l", nextImage);

emitter.on("key-t", (event: any) => {
  openTagPickerWithHotkey(event);
});

emitter.on("key-Escape", () => {
  showTagPicker.value = false;
});

function openTagPickerWithHotkey(event: any) {
  if (showTagPicker.value) return;
  event.preventDefault();
  emitter.emit("display-tag-picker", event);
  showTagPicker.value = true;
}

function openTagPicker() {
  if (showTagPicker.value) return;
  emitter.emit("display-tag-picker");
  showTagPicker.value = true;
}

function getImageThumbnailUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/thumb?size=200`;
}

function getImageUrl(image: Image): string {
  return `${API_BASE_URL}/projects/${props.projectId}/images/${image.id}/file?size=1500`;
}

function sortTags(image: Image): Image {
  if (!image.edges) return image;
  if (!image.edges.tags) return image;
  image.edges.tags = image.edges.tags.sort((a, b) => {
    return a.name.localeCompare(b.name);
  });
  return image;
}

async function fetchImageList() {
  const url = `/projects/${props.projectId}/images?limit=${limit.value}&offset=${offset.value}`;
  const response = await useFetch(url, getFetchOptions(Method.GET));
  if (response.data.value) {
    const data = response.data.value as ListResponse<Image>;
    images.value = data.items.map((image) => sortTags(image));
    totalImages.value = data.total;
  }
  calculatePrefetchImages();
}

async function fetchCurrentImage() {
  if (images.value[currentImageOffset.value]) {
    const url = `/projects/${props.projectId}/images/${images.value[currentImageOffset.value].id}`;
    const response = await useFetch(url, getFetchOptions(Method.GET));
    if (response.data.value) {
      const data = response.data.value as Image;
      images.value[currentImageOffset.value] = sortTags(data);
    }
  }
}

function selectImage(image: Image) {
  currentImageOffset.value = images.value.indexOf(image);
}

function getImageIndex(image: Image): number {
  return images.value.indexOf(image);
}

async function tagSelected(tag: Tag) {
  showTagPicker.value = false;
  if (images.value[currentImageOffset.value]) {
    const imageIndex = getImageIndex(images.value[currentImageOffset.value]);
    let currentTags: Array<Tag> = [];
    if (images.value[imageIndex].edges && images.value[imageIndex].edges.tags) {
      currentTags = images.value[imageIndex].edges.tags;
    }
    const url = `/projects/${props.projectId}/images/${images.value[imageIndex].id}`;
    const response = await useFetch(url, getFetchOptions(Method.PUT, { tags: [...currentTags.map((t: Tag) => t.id), tag.id] }));
    if (response.data.value) {
      const data = response.data.value as Image;
      const ownUser = store.getOwnUser();
      images.value[imageIndex].updatedAt = data.updatedAt;
      // @ts-ignore
      if (!images.value[imageIndex].edges) images.value[imageIndex].edges = {};
      if (ownUser) {
        images.value[imageIndex].edges.updatedBy = ownUser;
      }
      images.value[imageIndex].edges.tags = [...currentTags, tag].sort((a, b) => a.name.localeCompare(b.name));
    }
  }
}

const removeTagCandidate = ref<Tag | null>(null);
function requestRemoveTag(tag: Tag) {
  removeTagCandidate.value = tag;
  showRemoveTagDialog.value = true;
}
async function removeTag() {
  showRemoveTagDialog.value = false;
  if (images.value[currentImageOffset.value]) {
    const imageIndex = getImageIndex(images.value[currentImageOffset.value]);
    let currentTags: Array<Tag> = [];
    if (images.value[imageIndex].edges && images.value[imageIndex].edges.tags) {
      currentTags = images.value[imageIndex].edges.tags.filter((t: Tag) => t.id !== removeTagCandidate.value?.id);
    }
    const url = `/projects/${props.projectId}/images/${images.value[imageIndex].id}`;
    const response = await useFetch(url, getFetchOptions(Method.PUT, { tags: [...currentTags.map((t: Tag) => t.id)] }));
    if (response.data.value) {
      const data = response.data.value as Image;
      const ownUser = store.getOwnUser();
      images.value[imageIndex].updatedAt = data.updatedAt;
      // @ts-ignore
      if (!images.value[imageIndex].edges) images.value[imageIndex].edges = {};
      if (ownUser) {
        images.value[imageIndex].edges.updatedBy = ownUser;
      }
      images.value[imageIndex].edges.tags = [...currentTags.filter((t: Tag) => t.id !== removeTagCandidate.value?.id)].sort((a, b) => a.name.localeCompare(b.name));
    }
  }
}

function calculatePrefetchImages() {
  const prefetchStart = Math.max(0, currentImageOffset.value - 10);
  const prefetchEnd = Math.min(images.value.length, currentImageOffset.value + 10);
  prefetchImages.value = images.value.slice(prefetchStart, prefetchEnd);
}

watch(currentImageOffset, async () => {
  fetchCurrentImage();
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
