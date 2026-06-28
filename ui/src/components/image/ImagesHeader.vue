<template>
  <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
    <div class="min-w-0 flex-1">
      <h2 class="truncate text-2xl font-semibold tracking-tight text-primary-900 dark:text-white">{{ activeProject.name }}</h2>
      <div class="mt-1.5 flex items-center gap-1.5 text-sm text-primary-500 dark:text-primary-400">
        <PhotoIcon class="h-4 w-4 flex-shrink-0" />
        <span class="font-data tabular-nums">{{ totalImageCount }}</span>
        <span>images</span>
      </div>
    </div>

    <div v-if="showFilter" class="flex flex-wrap items-end gap-3">
      <!-- density -->
      <div class="inline-flex rounded-md border border-primary-200 dark:border-primary-700 bg-primary-50 dark:bg-primary-900 p-0.5">
        <button
          v-for="opt in densityOptions"
          :key="opt.value"
          type="button"
          :title="opt.label"
          @click="emit('update:density', opt.value)"
          :class="[
            'inline-flex items-center justify-center rounded p-1.5 transition-colors',
            density === opt.value
              ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
              : 'text-primary-500 hover:bg-primary-100 hover:text-primary-800 dark:text-primary-400 dark:hover:bg-primary-800 dark:hover:text-primary-100',
          ]"
        >
          <span class="sr-only">{{ opt.label }}</span>
          <component :is="opt.icon" class="h-4 w-4" />
        </button>
      </div>

      <AspectRatioFilter @state-changed="emitAspectRatioFilter" />
      <ProjectTagComboBox ref="projectTagComboBox" @selected="emitFilterTags" />

      <input
        id="search"
        v-model="searchText"
        placeholder="Search"
        type="text"
        @focusin="emitter.emit('block-hotkeys')"
        @focusout="emitter.emit('unblock-hotkeys')"
        class="block w-40 rounded-md border border-primary-300 bg-surface px-3 py-2 text-sm text-primary-900 placeholder:text-primary-400 transition-colors focus:border-accent-500 focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500"
      />

      <select
        id="sorting"
        name="sorting"
        v-model="preferredImageSortOrder"
        class="block rounded-md border border-primary-300 bg-surface px-3 py-2 text-sm text-primary-900 transition-colors focus:border-accent-500 focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100"
      >
        <option value="latestFirst">Latest first</option>
        <option value="oldestFirst">Oldest first</option>
        <option value="mostRecentlyUpdated">Recently updated</option>
        <option value="leastRecentlyUpdated">Least recently updated</option>
      </select>
    </div>
  </div>
</template>
<script setup lang="ts">
import { PhotoIcon, Squares2X2Icon, ViewColumnsIcon, TableCellsIcon } from "@heroicons/vue/24/outline";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { emitter } from "src/boot/mitt";
import { ref, watch } from "vue";
import ProjectTagComboBox from "../ProjectTagComboBox.vue";
import { ImageTagsResponse } from "src/types/pocketbase";
import AspectRatioFilter, { type AspectRatioState } from "./AspectRatioFilter.vue";

type Density = "gallery" | "comfortable" | "dense";

interface Props {
  totalImageCount: number;
  showFilter: boolean;
  density?: Density;
}
const props = withDefaults(defineProps<Props>(), {
  totalImageCount: 0,
  density: "comfortable",
});

const emit = defineEmits<{
  search: [string];
  filterTags: [ImageTagsResponse[]];
  aspectRatioFilter: [AspectRatioState];
  "update:density": [Density];
}>();

const densityOptions: { value: Density; label: string; icon: any }[] = [
  { value: "gallery", label: "Gallery", icon: ViewColumnsIcon },
  { value: "comfortable", label: "Grid", icon: Squares2X2Icon },
  { value: "dense", label: "Dense", icon: TableCellsIcon },
];

const { activeProject, preferredImageSortOrder } = storeToRefs(useUserStore());

const searchText = ref("");
watch(searchText, () => emit("search", searchText.value));

function emitFilterTags(tags: ImageTagsResponse[]) {
  emit("filterTags", tags);
}

function emitAspectRatioFilter(aspectRatioState: AspectRatioState) {
  emit("aspectRatioFilter", aspectRatioState);
}

const projectTagComboBox = ref<any>(null);
function setFilteredTags(tags: ImageTagsResponse[]) {
  if (projectTagComboBox.value) {
    projectTagComboBox.value.setFilteredTags(tags);
  }
}

defineExpose({
  setFilteredTags,
});
</script>
<script lang="ts">
export { SORT_ORDER } from "./sortOrder";
</script>
