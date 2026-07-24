<template>
  <div class="">
    <div class="mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8">
      <ImagesHeader
        ref="imagesHeader"
        v-model:density="density"
        :total-image-count="totalImageCount"
        :show-filter="displayMode === DisplayMode.GRID"
        @search="updateSearchText"
        @filter-tags="updateFilterTags"
        @aspect-ratio-filter="updateAspectRatioFilter"
      />
      <div v-if="displayMode === DisplayMode.GRID">
        <div :class="['mt-8 select-none', gridClasses]">
          <ImageGridTile
            v-for="(image, index) in images"
            :image="image"
            :key="image.id"
            :density="density"
            :selected="index === imageIndex || imageIndices.includes(index)"
            @select="selectImage"
          />
        </div>
        <ImagesFooter :current-image-count="images.length" :total-image-count="totalImageCount" :filtered="filtered" :loading="loading" @load-more="() => loadImages(false)" />
      </div>
    </div>

    <!-- Detail view: full-bleed so the photo gets the width the grid's max-w-7xl
         would waste. Stacks image-over-details on narrow viewports, details
         panel left of the image from lg up. pb clears the fixed film strip. -->
    <div v-if="displayMode === DisplayMode.DETAIL && imageIndex !== -1 && images[imageIndex]" class="w-full px-4 pb-32 sm:px-6 lg:px-8">
      <div class="mx-auto mt-8 flex max-w-screen-2xl flex-col-reverse gap-6 lg:flex-row">
        <Sidebar :item="images[imageIndex]" />
        <figure class="min-w-0 flex-1">
          <img
            :src="heroSrc(images[imageIndex])"
            @error="onHeroError(images[imageIndex])"
            :alt="images[imageIndex].computedFileName"
            class="mx-auto max-h-[max(18rem,calc(100vh-24rem))] max-w-full rounded-sm drop-shadow-lg"
          />
          <figcaption class="mt-3 flex items-baseline justify-center gap-4">
            <span class="truncate font-data text-sm text-primary-700 dark:text-primary-200">{{ images[imageIndex].computedFileName }}</span>
            <span class="label-mono-sm shrink-0 text-primary-500 dark:text-primary-400">{{ imageIndex + 1 }} / {{ totalImageCount.toLocaleString() }}</span>
          </figcaption>
        </figure>
      </div>
    </div>
    <FilmStrip v-if="displayMode === DisplayMode.DETAIL && imageIndex !== -1" :images="images" :current-index="imageIndex" @select="selectFromStrip" />
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
import ImageGridTile from "src/components/image/ImageGridTile.vue";
import ImagesHeader, { SORT_ORDER } from "src/components/image/ImagesHeader.vue";
import ImagesFooter from "src/components/image/ImagesFooter.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import Sidebar from "src/components/image/Sidebar.vue";
import TaggingDialog from "src/components/image/TaggingDialog.vue";
import FilmStrip from "src/components/image/FilmStrip.vue";
import { devPlaceholder } from "src/util/devPlaceholder";
import { ImageWithTagsType } from "src/types/custom";
import { onMounted, onUnmounted, reactive, ref, computed, watch, nextTick } from "vue";
import { useRouter } from "vue-router";
import { useDebounceFn, useStorage } from "@vueuse/core";

import { DisplayMode, loadImages, triggerInfiniteScroll } from "./imageQueryLogic";
import { preferredImageSortOrder, searchText, updateSearchText, filterTags, updateFilterTags, aspectRatioFilter, updateAspectRatioFilter, filtered } from "./imageQueryLogic";
import { totalImageCount, images, imageIndex, imageIndices, multiselectStart, multiselectEnd, loading } from "./imageQueryLogic";
import { taggingDialogVisible, addImageTag } from "./imageQueryLogic";
import { showUnexpectedErrorMessage, unexpectedError } from "./imageQueryLogic";
import { nextImage, previousImage, previousRow, nextRow, repeatLastTagAssignment, toggleTagByName } from "./imageQueryLogic";
import { useHotkeyAction, useHotkeyContext, useTagHotkey } from "src/util/hotkeys";
import { emitter } from "src/boot/mitt";
import { debug } from "src/util/logger";

const router = useRouter();

const displayMode = ref(DisplayMode.GRID);

// Grid density: relaxed fine-art masonry, comfortable grid, or Immich-dense.
type Density = "gallery" | "comfortable" | "dense";
const density = useStorage<Density>("image-grid-density", "comfortable");
const gridClasses = computed(() => {
  switch (density.value) {
    case "gallery":
      return "columns-2 md:columns-3 xl:columns-4 gap-4 [column-fill:_balance]";
    case "dense":
      return "grid grid-cols-3 sm:grid-cols-6 lg:grid-cols-8 2xl:grid-cols-10 gap-px";
    default:
      return "grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4";
  }
});

const imagesHeader = ref<any>(null);

async function onScroll() {
  if (window.innerHeight + window.scrollY + 100 >= document.body.scrollHeight) {
    triggerInfiniteScroll();
  }
}
window.addEventListener("scroll", onScroll);

onMounted(() => loadImages(true));
const reloadDebounced = useDebounceFn(() => loadImages(true), 500);
watch(preferredImageSortOrder, () => loadImages(true));
watch(searchText, reloadDebounced);
watch(filterTags, reloadDebounced);
watch(aspectRatioFilter, reloadDebounced);

// Hotkey wiring: the images context and its handlers are only active while
// this page is mounted; the dispatcher suppresses them while the tagging
// dialog (own context) is open.
useHotkeyContext("images");
useHotkeyAction("images.next-image", nextImage);
useHotkeyAction("images.previous-image", previousImage);
useHotkeyAction("images.previous-row", previousRow);
useHotkeyAction("images.next-row", nextRow);
useHotkeyAction("images.repeat-last-tag", repeatLastTagAssignment);
useHotkeyAction("images.toggle-view", toggleGridDetail);
useHotkeyAction("images.open-tagging", showTaggingDialog);
useHotkeyAction("tagging.close", hideTaggingDialog);
useTagHotkey(toggleTagByName);

function toggleGridDetail() {
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

emitter.on("show-tagging-dialog", showTaggingDialog); // from sidebar button
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

emitter.on("current-image-deleted", handleCurrentImageDeleted);
function handleCurrentImageDeleted(deletedImageId: string) {
  const index = images.value.findIndex((image) => image.id === deletedImageId);
  if (index !== -1) {
    images.value.splice(index, 1);
  }
  imageIndex.value = Math.max(0, imageIndex.value - 1);
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

function selectFromStrip(index: number) {
  imageIndex.value = index;
  if (index >= images.value.length - 5) {
    triggerInfiniteScroll();
  }
}

// hero fallback mirrors the thumbnail behavior (see devPlaceholder.ts)
const heroOverrides = reactive<Record<string, string>>({});
function heroSrc(image: ImageWithTagsType): string {
  return heroOverrides[image.id] ?? image.downloadUrls?.["2048"] ?? "";
}
function onHeroError(image: ImageWithTagsType) {
  const placeholder = devPlaceholder(image.id);
  if (placeholder && heroOverrides[image.id] !== placeholder) {
    heroOverrides[image.id] = placeholder;
  }
}

emitter.on("update-image-grid-scroll-position", scrollToSelectedImage);
function scrollToSelectedImage() {
  const activeItem = document.querySelector(`#grid-tile-${images.value[imageIndex.value].id}`);
  if (activeItem) {
    activeItem.scrollIntoView({ behavior: `instant`, block: `nearest` });
  }
}

onUnmounted(() => {
  window.removeEventListener("scroll", onScroll);
  emitter.off("show-tagging-dialog", showTaggingDialog);
  emitter.off("reset-tagging-dialog", resetTaggingDialog);
  emitter.off("current-image-deleted", handleCurrentImageDeleted);
  emitter.off("update-image-grid-scroll-position", scrollToSelectedImage);
});
</script>
