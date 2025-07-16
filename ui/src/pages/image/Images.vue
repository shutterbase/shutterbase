<template>
  <div class="">
    <div class="mx-auto max-w-7xl w-full overflow-hidden sm:px-6 lg:px-8">
      <ImagesHeader
        ref="imagesHeader"
        :total-image-count="totalImageCount"
        :show-filter="displayMode === DisplayMode.GRID"
        @search="updateSearchText"
        @filter-tags="updateFilterTags"
        @aspect-ratio-filter="updateAspectRatioFilter"
      />
      <div v-if="displayMode === DisplayMode.GRID">
        <div class="mt-10 grid grid-cols-1 border-l border-gray-200 dark:border-gray-600 sm:mx-0 md:grid-cols-3 lg:grid-cols-4 select-none">
          <ImageGridTile v-for="(image, index) in images" :image="image" :key="image.id" :selected="index === imageIndex || imageIndices.includes(index)" @select="selectImage" />
        </div>
        <ImagesFooter :current-image-count="images.length" :total-image-count="totalImageCount" :filtered="filtered" @load-more="() => loadImages(false)" />
      </div>
      <div class="flex" v-if="displayMode === DisplayMode.DETAIL && imageIndex !== -1">
        <Sidebar :item="images[imageIndex]" />
        <div v-if="images[imageIndex]" class="flex-1 flex items-center justify-center mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8">
          <div class="relative">
            <img :src="images[imageIndex].downloadUrls['2048']" alt="Centered Image" class="max-w-full max-h-[52rem] mx-auto drop-shadow-lg" />
          </div>
        </div>
      </div>
    </div>
  </div>
  <TaggingDialog
    v-if="imageIndex !== -1"
    ref="taggingDialog"
    :shown="taggingDialogVisible"
    @close="hideTaggingDialog"
    @close-and-next="closeAndNext"
    @selected="addImageTag"
    :image="images[imageIndex]"
  />
  <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
</template>
<script setup lang="ts">
import { storeToRefs } from "pinia";
import pb from "src/boot/pocketbase";
import ImageGridTile from "src/components/image/ImageGridTile.vue";
import ImagesHeader, { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import ImagesFooter from "src/components/image/ImagesFooter.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import Sidebar from "src/components/image/Sidebar.vue";
import TaggingDialog from "src/components/image/TaggingDialog.vue";
import { onMounted, ref, watch, nextTick } from "vue";
import { useRouter } from "vue-router";
import { useDebounceFn } from "@vueuse/core";

import { DisplayMode, loadImages, triggerInfiniteScroll } from "./imageQueryLogic";
import { preferredImageSortOrder, searchText, updateSearchText, filterTags, updateFilterTags, aspectRatioFilter, updateAspectRatioFilter, filtered } from "./imageQueryLogic";
import { totalImageCount, images, imageIndex, imageIndices, multiselectStart, multiselectEnd } from "./imageQueryLogic";
import { taggingDialogVisible, addImageTag } from "./imageQueryLogic";
import { showUnexpectedErrorMessage, unexpectedError } from "./imageQueryLogic";
import { HotkeyEvent, onHotkey } from "src/util/keyEvents";
import { emitter } from "src/boot/mitt";
import { debug } from "src/util/logger";

const router = useRouter();

const displayMode = ref(DisplayMode.GRID);

const imagesHeader = ref<any>(null);

window.onscroll = async function (ev) {
  if (window.innerHeight + window.scrollY + 100 >= document.body.scrollHeight) {
    triggerInfiniteScroll();
  }
};

onMounted(() => loadImages(true));
const reloadDebounced = useDebounceFn(() => loadImages(true), 500);
watch(preferredImageSortOrder, () => loadImages(true));
watch(searchText, reloadDebounced);
watch(filterTags, reloadDebounced);
watch(aspectRatioFilter, reloadDebounced);

onHotkey({ key: "g", modifierKeys: [] }, toggleGridDetail);
function toggleGridDetail(event: HotkeyEvent) {
  if (taggingDialogVisible.value) {
    return;
  }

  event.event.preventDefault();
  if (displayMode.value === DisplayMode.GRID) {
    showDetail();
    displayMode.value = DisplayMode.DETAIL;
  } else {
    displayMode.value = DisplayMode.GRID;
    nextTick(() => {
      scrollToSelectedImage();
      if (imagesHeader.value) {
        imagesHeader.value.setFilteredTags(filterTags.value);
      }
    });
  }
}

function showDetail() {
  multiselectStart.value = null;
  multiselectEnd.value = null;
  imageIndices.value = [];
  if (imageIndex.value === -1) {
    imageIndex.value = 0;
  }
  displayMode.value = DisplayMode.DETAIL;
}

onMounted(clearFilterTags);
function clearFilterTags() {
  filterTags.value = [];
}

const taggingDialog = ref<InstanceType<typeof TaggingDialog> | null>(null);

onHotkey({ key: "t", modifierKeys: [] }, showTaggingDialogViaHotkey);
emitter.on("show-tagging-dialog", showTaggingDialog); // from sidebar button
function showTaggingDialogViaHotkey(event: HotkeyEvent) {
  if (!taggingDialogVisible.value) {
    event.event.preventDefault();
  }
  showTaggingDialog();
}
function showTaggingDialog() {
  if (!taggingDialogVisible.value) {
    taggingDialogVisible.value = true;
    nextTick(() => {
      taggingDialog.value?.focusSearchText();
      taggingDialog.value?.clearSearchText();
    });
    debug("show tag dialog");
  }
}
onHotkey({ key: "Escape", modifierKeys: [] }, hideTaggingDialog);
function hideTaggingDialog() {
  if (taggingDialogVisible.value) {
    taggingDialogVisible.value = false;
    debug("hide tag dialog");
  }
}
function closeAndNext() {
  hideTaggingDialog();
  nextTick(() => {
    if (imageIndex.value + 1 < images.value.length) {
      imageIndex.value++;
    }
  });
}

emitter.on("reset-tagging-dialog", resetTaggingDialog);
function resetTaggingDialog() {
  nextTick(() => {
    taggingDialog.value?.focusSearchText();
    taggingDialog.value?.clearSearchText();
  });
}

function selectImage(imageId: string, event: MouseEvent) {
  const index = images.value.findIndex((image) => image.id === imageId);
  if (event.shiftKey) {
    if (multiselectStart.value !== null && multiselectEnd.value !== null) {
      multiselectStart.value = null;
      multiselectEnd.value = null;
      imageIndices.value = [];
    }

    if (multiselectStart.value === null) {
      multiselectStart.value = index;
    } else {
      multiselectEnd.value = index;
    }
    imageIndex.value = index;
  } else {
    imageIndex.value = index;
    showDetail();
  }

  if (multiselectStart.value !== null && multiselectEnd.value !== null) {
    const start = Math.min(multiselectStart.value, multiselectEnd.value);
    const end = Math.max(multiselectStart.value, multiselectEnd.value);
    for (let i = start; i <= end; i++) {
      imageIndices.value.push(i);
    }
  }
}

emitter.on("update-image-grid-scroll-position", scrollToSelectedImage);
function scrollToSelectedImage() {
  const activeItem = document.querySelector(`#grid-tile-${images.value[imageIndex.value].id}`);
  if (activeItem) {
    activeItem.scrollIntoView({ behavior: `instant`, block: `nearest` });
  }
}
</script>
