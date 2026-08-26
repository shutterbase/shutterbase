<template>
  <div class="">
    <div class="mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8">
      <ImagesHeader
        ref="imagesHeader"
        v-model:density="density"
        :total-image-count="totalImageCount"
        :show-filter="displayMode === DisplayMode.GRID"
        :selection-count="imageIndices.length"
        :upload-filter="uploadFilter"
        :tag-facets="tagFacets"
        :time-from="timeFromFilter"
        :time-to="timeToFilter"
        @search="updateSearchText"
        @filter-tags="updateFilterTags"
        @facets-needed="loadTagFacets"
        @aspect-ratio-filter="updateAspectRatioFilter"
        @time-range="setTimeRange"
        @rerun-ai="rerunSelection"
        @upload-filter="setUploadFilter"
        @slideshow="slideshowActive = true"
      />
      <div v-if="displayMode === DisplayMode.GRID">
        <div v-if="timeFromFilter || timeToFilter" class="mt-6 flex flex-wrap items-center gap-3" data-testid="time-range-chip-row">
          <span
            class="label-mono-sm inline-flex items-center gap-2 rounded-full border border-accent-400/60 bg-accent-500/10 px-3 py-1 text-accent-600 dark:text-accent-300"
            data-testid="time-range-chip"
          >
            {{ timeRangeLabel }}
            <button class="cursor-pointer font-bold hover:text-accent-400" title="Clear time range" @click="setTimeRange(null, null)">×</button>
          </span>
        </div>
        <div v-if="personFilter" class="mt-6 flex flex-wrap items-center gap-3">
          <span class="label-mono-sm inline-flex items-center gap-2 rounded-full border border-accent-400/60 px-3 py-1 text-accent-600 dark:text-accent-300">
            photos of one person
            <button class="cursor-pointer font-bold hover:text-accent-400" title="Clear person filter" @click="clearPersonFilter()">×</button>
          </span>
          <button
            :class="[
              'label-mono-sm cursor-pointer rounded-full border px-3 py-1 transition-colors',
              personCrossProject
                ? 'border-accent-400/60 bg-accent-600/10 text-accent-600 dark:text-accent-300'
                : 'border-primary-300 text-primary-500 hover:border-primary-400 hover:text-primary-700 dark:border-primary-700 dark:text-primary-400 dark:hover:text-primary-200',
            ]"
            :title="personCrossProject ? 'Showing this person across all your projects — click to limit to this project' : 'Also search your other projects for this person'"
            @click="togglePersonScope()"
          >
            all my projects
          </button>
          <button
            v-if="personFilter && hasPausableFilters"
            :class="[
              'label-mono-sm cursor-pointer rounded-full border px-3 py-1 transition-colors',
              personFiltersPaused
                ? 'border-primary-300 text-primary-500 hover:border-primary-400 hover:text-primary-700 dark:border-primary-700 dark:text-primary-400 dark:hover:text-primary-200'
                : 'border-accent-400/60 bg-accent-600/10 text-accent-600 dark:text-accent-300',
            ]"
            :title="
              personFiltersPaused
                ? 'Search, tag and orientation filters are paused for this person — click to apply them again'
                : 'Search, tag and orientation filters are applied — click to pause them and see all photos of this person'
            "
            @click="togglePersonFilters()"
          >
            Filters
          </button>
          <button
            v-if="isProjectAdminOrHigher"
            class="label-mono-sm cursor-pointer rounded-full border border-primary-300 px-3 py-1 text-primary-500 transition-colors hover:border-primary-400 hover:text-primary-700 dark:border-primary-700 dark:text-primary-400 dark:hover:text-primary-200"
            title="Review face clusters similar to this person and merge them"
            @click="openSimilarFaces()"
          >
            similar faces
          </button>
        </div>
        <div :class="['mt-8 select-none', gridClasses]">
          <ImageGridTile
            v-for="(image, index) in images"
            :image="image"
            :key="image.id"
            :density="density"
            :selected="index === imageIndex || imageIndices.includes(index)"
            :ai-position="aiPositions[image.id]"
            :show-project="personCrossProject"
            @select="selectImage"
          />
        </div>
        <ImagesFooter :current-image-count="images.length" :total-image-count="totalImageCount" :filtered="filtered" :loading="loading" @load-more="() => loadImages(false)" />
      </div>
    </div>

    <!-- Detail view: full-bleed so the photo gets the width the grid's max-w-7xl
         would waste. Stacks image-over-details on narrow viewports, details
         panel left of the image from lg up. No bottom padding: the film strip
         is fixed, so clearance comes from the column height caps (sidebar
         max-h, hero max-h), not from flow padding — flow padding is what made
         the whole page scroll. -->
    <div v-if="displayMode === DisplayMode.DETAIL && imageIndex !== -1 && images[imageIndex]" class="w-full px-4 sm:px-6 lg:px-8">
      <div class="mx-auto mt-4 flex max-w-screen-2xl flex-col-reverse gap-6 lg:flex-row">
        <Sidebar :item="images[imageIndex]" />
        <figure class="min-w-0 flex-1">
          <!-- zoomed, the image breaks out of the column into a viewport-wide
               stage that spares only the header bar (top-16) and the film
               strip (bottom-24); z-0 keeps it the lowest element -->
          <ZoomableImage
            class="mx-auto"
            :src="heroSrc(images[imageIndex])"
            :hires-src="heroHiresSrc(images[imageIndex])"
            :alt="images[imageIndex].computedFileName"
            img-class="mx-auto max-h-[max(18rem,calc(100vh-25rem))] max-w-full rounded-sm drop-shadow-lg"
            expand-class="fixed inset-x-0 top-16 bottom-24 z-0"
            @error="onHeroError(images[imageIndex])"
          >
            <template v-if="facesVisible">
              <div
                v-for="(face, i) in faces"
                :key="i"
                :style="faceBoxStyle(face)"
                :class="[
                  'absolute rounded-sm border-2 border-accent-400/90 shadow-[0_0_0_1px_rgba(0,0,0,0.4)] transition-colors',
                  face.personRef ? 'cursor-pointer hover:border-accent-200 hover:bg-accent-400/20' : '',
                ]"
                :title="face.personRef ? (face.count ? `Detected ${face.count}× — show photos of this person` : 'Show photos of this person') : ''"
                @click="face.personRef && showPersonInGrid(face.personRef)"
              >
                <span
                  v-if="face.count"
                  class="absolute -left-1.5 -top-2.5 rounded-full bg-accent-400 px-1 font-data text-[10px] font-semibold leading-4 text-primary-950 shadow-[0_0_0_1px_rgba(0,0,0,0.4)]"
                  >{{ face.count }}</span
                >
              </div>
            </template>
            <RejectedStamp v-if="stampedImageId === images[imageIndex].id" />
          </ZoomableImage>
          <figcaption class="mt-3 flex items-baseline justify-center gap-4">
            <span class="truncate font-data text-sm text-primary-700 dark:text-primary-200">{{ images[imageIndex].computedFileName }}</span>
            <span class="label-mono-sm shrink-0 text-primary-500 dark:text-primary-400">{{ imageIndex + 1 }} / {{ totalImageCount.toLocaleString() }}</span>
          </figcaption>
        </figure>
      </div>
    </div>
    <FilmStrip v-if="displayMode === DisplayMode.DETAIL && imageIndex !== -1" :images="images" :current-index="imageIndex" @select="selectFromStrip" />
  </div>
  <!-- Zen mode: only the image, fullscreen, over everything except the tagging
       dialog (z-50) and toasts (z-[60]). The underlying grid/detail view stays
       untouched, so leaving zen lands exactly where the user was. -->
  <div v-if="zenMode && imageIndex !== -1 && images[imageIndex]" data-testid="zen-overlay" class="fixed inset-0 z-40 flex items-center justify-center bg-black">
    <ZoomableImage
      :src="heroSrc(images[imageIndex])"
      :hires-src="heroHiresSrc(images[imageIndex])"
      :alt="images[imageIndex].computedFileName"
      img-class="max-h-dvh max-w-[100vw] object-contain"
      expand-class="fixed inset-0"
      @error="onHeroError(images[imageIndex])"
    >
      <RejectedStamp v-if="stampedImageId === images[imageIndex].id" />
    </ZoomableImage>
    <!-- thin header: logo only; always the dark variant, zen is always on black -->
    <div class="absolute inset-x-0 top-0 flex justify-center bg-black/70 py-1">
      <img class="h-5" src="~assets/img/shutterbase-header-logo-dark.png" alt="shutterbase" />
    </div>
    <!-- thin footer: name | all applied tags (sidebar category order; click removes) | position -->
    <div class="absolute inset-x-0 bottom-0 flex items-center gap-4 bg-black/70 px-3 py-1">
      <span class="min-w-0 shrink truncate font-data text-sm text-primary-200">{{ images[imageIndex].computedFileName }}</span>
      <div class="flex flex-1 flex-wrap items-center justify-center gap-2">
        <ImageTagBadge
          v-for="tagAssignment in zenTags"
          :key="tagAssignment.id"
          :tagAssignment="tagAssignment"
          :removable="canRemoveTagAssignment(images[imageIndex], tagAssignment)"
          @remove="(ta) => removeTagAssignment(images[imageIndex], ta)"
        />
      </div>
      <span class="label-mono-sm shrink-0 text-primary-400">{{ imageIndex + 1 }} / {{ totalImageCount.toLocaleString() }}</span>
    </div>
  </div>
  <!-- Slideshow: plays the current (filtered, sorted) view; images keep loading
       page by page as the show approaches the end of the loaded list. -->
  <SlideshowOverlay v-if="slideshowActive" :images="images" :total-count="totalImageCount" @close="slideshowActive = false" @need-more="triggerInfiniteScroll" />
  <TaggingDialog
    v-if="imageIndex !== -1"
    ref="taggingDialog"
    :shown="taggingDialogVisible"
    @close="hideTaggingDialog"
    @close-and-next="closeAndNext"
    @selected="addImageTag"
    :image="images[imageIndex]"
  />
  <AiImageListDialog :shown="aiDialogVisible" :image-id="imageIndex !== -1 ? images[imageIndex]?.id : undefined" @close="aiDialogVisible = false" @select="selectFromAiDialog" />
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
import ZoomableImage from "src/components/image/ZoomableImage.vue";
import SlideshowOverlay from "src/components/image/SlideshowOverlay.vue";
import ImageTagBadge from "src/components/image/ImageTagBadge.vue";
import RejectedStamp from "src/components/image/RejectedStamp.vue";
import { nextStampedImageId, reviewVerdicts } from "src/util/uploadReview";
import { canRemoveTagAssignment, removeTagAssignment } from "src/util/imageTags";
import { groupTagAssignments } from "src/util/tagOrder";
import { devPlaceholder } from "src/util/devPlaceholder";
import { ImageWithTagsType } from "src/types/custom";
import { onMounted, onUnmounted, reactive, ref, computed, watch, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useDebounceFn, useStorage } from "@vueuse/core";

import AiImageListDialog from "src/components/image/AiImageListDialog.vue";
import { api } from "src/api";
import { AiFace } from "src/api/ai";
import { faceBoxStyle } from "src/util/aiDetection";
import { useUserStore } from "src/stores/user-store";
import * as websocket from "src/util/websocket";

import { DisplayMode, loadImages, triggerInfiniteScroll, jumpToImage, soloImage } from "./imageQueryLogic";
import {
  preferredImageSortOrder,
  searchText,
  updateSearchText,
  filterTags,
  excludeFilterTags,
  updateFilterTags,
  aspectRatioFilter,
  updateAspectRatioFilter,
  filtered,
  personFilter,
  personCrossProject,
  personFiltersPaused,
  uploadFilter,
  timeFromFilter,
  timeToFilter,
  snapshotGrid,
  restoreGridSnapshot,
  invalidateGridSnapshot,
  resetTransientFilters,
  tagFacets,
  loadTagFacets,
} from "./imageQueryLogic";
import { totalImageCount, images, imageIndex, imageIndices, multiselectStart, multiselectEnd, loading } from "./imageQueryLogic";
import { taggingDialogVisible, addImageTag } from "./imageQueryLogic";
import { showUnexpectedErrorMessage, unexpectedError } from "./imageQueryLogic";
import { nextImage, previousImage, previousRow, nextRow, repeatLastTagAssignment, toggleTagByName } from "./imageQueryLogic";
import { aiPositions, refreshAiPositions, applyAiEvent, rerunAiSelection } from "./imageQueryLogic";
import { useHotkeyAction, useHotkeyContext, useTagHotkey } from "src/util/hotkeys";
import { emitter, showNotificationToast } from "src/boot/mitt";
import { debug } from "src/util/logger";

const router = useRouter();
const route = useRoute();

const displayMode = ref(DisplayMode.GRID);

// --- history-backed view state ----------------------------------------------
// The visible view is encoded in the route query — ?person=<ref> for the
// implicit person filter, ?image=<id> for the detail view — so the browser
// back button walks grid ⇄ detail ⇄ filter states. UI events only push/replace
// the query; applyRoute() is the single place state actually changes.

function pushQuery(mutate: (q: Record<string, any>) => void, replace = false) {
  const q: Record<string, any> = { ...route.query };
  mutate(q);
  if (JSON.stringify(q) === JSON.stringify(route.query)) return;
  const target = { query: q };
  if (replace) router.replace(target);
  else router.push(target);
}

const openDetail = (imageId: string) => pushQuery((q) => (q.image = imageId));
const closeDetail = () => pushQuery((q) => delete q.image);
const clearPersonFilter = () =>
  pushQuery((q) => {
    delete q.person;
    delete q.personScope;
  });
const setUploadFilter = (id: string | null) =>
  pushQuery((q) => {
    if (id) q.upload = id;
    else delete q.upload;
  });
const setTimeRange = (from: string | null, to: string | null) =>
  pushQuery((q) => {
    if (from) q.from = from;
    else delete q.from;
    if (to) q.to = to;
    else delete q.to;
  });

// chip label for the active range; dates shown only when the window spans days
const timeRangeLabel = computed(() => {
  const f = timeFromFilter.value ? new Date(timeFromFilter.value) : null;
  const t = timeToFilter.value ? new Date(timeToFilter.value) : null;
  if (!f && !t) return "";
  const hm = (d: Date) => d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  const day = (d: Date) => d.toLocaleDateString([], { day: "2-digit", month: "short" });
  if (f && t) {
    return f.toDateString() === t.toDateString() ? `${day(f)} · ${hm(f)} – ${hm(t)}` : `${day(f)} ${hm(f)} – ${day(t)} ${hm(t)}`;
  }
  return f ? `${day(f)} ${hm(f)} →` : `→ ${day(t!)} ${hm(t!)}`;
});
const togglePersonScope = () =>
  pushQuery((q) => {
    if (q.personScope === "all") delete q.personScope;
    else q.personScope = "all";
  });

// "Filters" pill: pause/resume the other narrowing filters while a person
// filter is active. Pure in-memory state — leaving the person view re-arms it.
const hasPausableFilters = computed(() => !!searchText.value || filterTags.value.length > 0 || excludeFilterTags.value.length > 0 || aspectRatioFilter.value !== "neutral");
function togglePersonFilters() {
  personFiltersPaused.value = !personFiltersPaused.value;
  loadImages(true);
}

// "similar faces": person-scoped merge review on the People page; back
// returns here and remounts the grid, so fresh merges show up immediately.
const isProjectAdminOrHigher = useUserStore().isProjectAdminOrHigher();
const openSimilarFaces = () => router.push({ name: "people", query: { person: personFilter.value } });

async function applyRoute(initial = false) {
  if (route.name !== "images") return;
  const person = (route.query.person as string) || null;
  const crossProject = route.query.personScope === "all";
  const uploadId = (route.query.upload as string) || null;
  const imageId = (route.query.image as string) || null;

  // Time range (?from=/?to=): assign before any load so a combined
  // person/upload change picks the new bounds up in the same reload; an
  // isolated range change reloads here.
  const from = (route.query.from as string) || null;
  const to = (route.query.to as string) || null;
  const timeChanged = from !== timeFromFilter.value || to !== timeToFilter.value;
  if (timeChanged) {
    timeFromFilter.value = from;
    timeToFilter.value = to;
  }

  if (initial || person !== personFilter.value || crossProject !== personCrossProject.value || uploadId !== uploadFilter.value) {
    if (person || uploadId) {
      // entering an implicit filter from the unfiltered grid: remember where we were
      if (!initial && !personFilter.value && !uploadFilter.value) snapshotGrid();
      personFilter.value = person;
      personCrossProject.value = crossProject;
      uploadFilter.value = uploadId;
      personFiltersPaused.value = true;
      imageIndex.value = -1;
      imageIndices.value = [];
      multiselectStart.value = null;
      multiselectEnd.value = null;
      await loadImages(true);
    } else {
      personFilter.value = null;
      personCrossProject.value = false;
      uploadFilter.value = null;
      personFiltersPaused.value = true;
      const scrollY = initial ? null : restoreGridSnapshot();
      if (scrollY === null) {
        await loadImages(true);
      } else {
        await nextTick();
        window.scrollTo({ top: scrollY, behavior: "instant" as ScrollBehavior });
      }
    }
  } else if (timeChanged) {
    // only the range moved: one reload, no snapshot churn
    invalidateGridSnapshot();
    imageIndex.value = -1;
    await loadImages(true);
  }

  if (imageId) {
    let index = images.value.findIndex((image) => image.id === imageId);
    if (index === -1) {
      // permalink beyond the loaded pages, into another project, or dead —
      // resolve it: jump-to-context, solo detail, or an explanatory toast
      const jump = await jumpToImage(imageId);
      if (jump.projectSwitched || jump.status === "solo") {
        // person/upload context params belonged to the previous view — a
        // no-op for applyRoute since the matching refs were cleared with them
        pushQuery((q) => {
          delete q.person;
          delete q.personScope;
          delete q.upload;
        }, true);
      }
      if (jump.status === "unavailable") {
        displayMode.value = DisplayMode.GRID;
        pushQuery((q) => delete q.image, true);
        return;
      }
      index = images.value.findIndex((image) => image.id === imageId);
      if (index === -1) return; // superseded by a newer navigation mid-flight
    }
    imageIndex.value = index;
    if (displayMode.value !== DisplayMode.DETAIL) {
      multiselectStart.value = null;
      multiselectEnd.value = null;
      imageIndices.value = [];
      displayMode.value = DisplayMode.DETAIL;
    }
  } else {
    if (soloImage.value) {
      // leaving a solo detail: the one-image array is no grid — load a real one
      imageIndex.value = -1;
      await loadImages(true);
    }
    if (displayMode.value !== DisplayMode.GRID) {
      displayMode.value = DisplayMode.GRID;
      nextTick(() => {
        if (imageIndex.value !== -1) scrollToSelectedImage();
        if (imagesHeader.value) imagesHeader.value.setFilteredTags(filterTags.value, excludeFilterTags.value);
      });
    }
  }
}

watch(
  () => route.query,
  () => applyRoute(),
);

// Stepping through images inside the detail view (hotkeys, film strip) keeps
// the URL current without growing the history.
watch(imageIndex, () => {
  if (displayMode.value !== DisplayMode.DETAIL) return;
  const id = images.value[imageIndex.value]?.id;
  if (id && route.query.image !== id) pushQuery((q) => (q.image = id), true);
});

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

// before the watchers below exist, so the reset itself never queues a reload
resetTransientFilters();

onMounted(() => applyRoute(true));
// any other filter/sort change makes the saved unfiltered-grid position stale
const reloadDebounced = useDebounceFn(() => {
  invalidateGridSnapshot();
  loadImages(true);
}, 500);
watch(preferredImageSortOrder, () => {
  invalidateGridSnapshot();
  loadImages(true);
});
watch(searchText, reloadDebounced);
watch([filterTags, excludeFilterTags], reloadDebounced);
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
useHotkeyAction("images.zen-toggle", toggleZen);
useHotkeyAction("images.open-tagging", showTaggingDialog);
useHotkeyAction("tagging.close", hideTaggingDialog);
useTagHotkey(toggleTagByName);

function toggleGridDetail() {
  if (zenMode.value) return; // zen owns the screen; z leaves it first
  if (displayMode.value === DisplayMode.GRID) {
    const id = images.value[imageIndex.value === -1 ? 0 : imageIndex.value]?.id;
    if (id) openDetail(id);
  } else {
    closeDetail();
  }
}

// Slideshow is an overlay like zen: the grid state underneath stays untouched.
const slideshowActive = ref(false);

// Zen mode is an overlay, not a display mode: grid/detail (and the route) stay
// as they are underneath, so z always returns to the previously active view.
const zenMode = ref(false);
function toggleZen() {
  if (zenMode.value) {
    zenMode.value = false;
    return;
  }
  if (imageIndex.value === -1 && images.value.length > 0) imageIndex.value = 0;
  if (images.value[imageIndex.value]) zenMode.value = true;
}
// All tags, not just the EXIF-exported subset — zen is primarily a tagging
// view, so custom/AI tags must be visible too. Sidebar category order.
const zenTags = computed(() => groupTagAssignments(images.value[imageIndex.value]?.tags ?? []).flatMap((g) => g.assignments));

// Rejected stamp: slams onto the on-screen image the moment the reserved
// "rejected" tag is applied (any path — dialog, hotkey, repeat-last) and stays
// until the user moves to another image or leaves the detail view.
const stampedImageId = ref<string | null>(null);
const currentReviewView = computed(() => {
  const image = images.value[imageIndex.value];
  if (!image) return null;
  return { id: image.id, rejected: reviewVerdicts({ reviewEnabled: !!image.project?.uploadReviewEnabled, tags: image.tags }).rejected };
});
watch(currentReviewView, (now, prev) => {
  stampedImageId.value = nextStampedImageId(stampedImageId.value, prev ?? null, now);
});
watch(displayMode, () => (stampedImageId.value = null));

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
    openDetail(imageId);
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
// full-res rendition for deep zoom — never paired with a placeholder base,
// the two would show different content
function heroHiresSrc(image: ImageWithTagsType): string | undefined {
  return heroOverrides[image.id] ? undefined : image.downloadUrls?.["original"];
}

emitter.on("update-image-grid-scroll-position", scrollToSelectedImage);
function scrollToSelectedImage() {
  const activeItem = document.querySelector(`#grid-tile-${images.value[imageIndex.value].id}`);
  if (activeItem) {
    activeItem.scrollIntoView({ behavior: `instant`, block: `nearest` });
  }
}

// --- AI detection: live status, face overlay, person/similar dialogs --------

const faces = ref<AiFace[]>([]);
const facesVisible = ref(false);
const aiDialogVisible = ref(false);

emitter.on("ai-toggle-faces", toggleFaces);
emitter.on("ai-show-similar", showSimilarDialog);

function toggleFaces() {
  facesVisible.value = !facesVisible.value;
  if (facesVisible.value) loadFaces();
}

async function loadFaces() {
  faces.value = [];
  const image = images.value[imageIndex.value];
  if (!image) return;
  try {
    faces.value = await api.ai.faces(image.id);
    if (faces.value.length === 0) {
      showNotificationToast({ headline: "No faces detected in this image", type: "info" });
    }
  } catch (error: any) {
    const status = error?.response?.status;
    showNotificationToast({
      headline: status === 404 ? "This image has not been analyzed yet" : "Face lookup unavailable",
      type: "info",
    });
    facesVisible.value = false;
  }
}

// changing the displayed image refreshes the overlay (when active)
watch(imageIndex, () => {
  if (facesVisible.value) loadFaces();
});

// Clicking a face filters the normal grid by that person — no result dialog.
// A route push, so browser-back returns to the view the face was clicked in.
function showPersonInGrid(personRef: string) {
  pushQuery((q) => {
    q.person = personRef;
    delete q.image;
  });
}

function showSimilarDialog() {
  aiDialogVisible.value = true;
}

function selectFromAiDialog(imageId: string) {
  const index = images.value.findIndex((image) => image.id === imageId);
  if (index === -1) {
    showNotificationToast({ headline: "Image is not in the current view — adjust filters to navigate to it", type: "info" });
    return;
  }
  aiDialogVisible.value = false;
  imageIndex.value = index;
  openDetail(imageId);
}

// selection rerun (button in the header, active when a selection exists)
async function rerunSelection() {
  await rerunAiSelection();
  refreshAiPositionsDebounced();
}

// live queue updates: patch statuses from the broadcast, refresh positions in
// one debounced batch (events arrive per image).
const refreshAiPositionsDebounced = useDebounceFn(refreshAiPositions, 1000);
let wsListenerId: string | null = null;
onMounted(() => {
  websocket.connect();
  wsListenerId = websocket.on({ object: "image", action: "changed" }, (message) => {
    applyAiEvent(message.data as { projectId: string; imageId: string; status: string });
    refreshAiPositionsDebounced();
  });
});
// initial + post-load position fetch
watch(() => images.value.length, refreshAiPositionsDebounced);

onUnmounted(() => {
  window.removeEventListener("scroll", onScroll);
  emitter.off("show-tagging-dialog", showTaggingDialog);
  emitter.off("reset-tagging-dialog", resetTaggingDialog);
  emitter.off("current-image-deleted", handleCurrentImageDeleted);
  emitter.off("update-image-grid-scroll-position", scrollToSelectedImage);
  emitter.off("ai-toggle-faces", toggleFaces);
  emitter.off("ai-show-similar", showSimilarDialog);
  if (wsListenerId) websocket.off(wsListenerId);
});
</script>
