<template>
  <div class="space-y-5">
    <!-- title row -->
    <div class="flex items-end justify-between gap-4">
      <div class="min-w-0">
        <p class="label-mono text-accent-600 dark:text-accent-400">Gallery</p>
        <h1 class="display mt-2 truncate text-[2rem] leading-none text-primary-900 dark:text-white sm:text-[2.6rem]">{{ activeProject.name }}</h1>
        <p class="label-mono mt-3 text-primary-500 dark:text-primary-400">
          <span class="font-data text-primary-700 dark:text-primary-200">{{ totalImageCount.toLocaleString() }}</span>
          {{ totalImageCount === 1 ? "frame" : "frames" }}
        </p>
      </div>

      <!-- density / view -->
      <div v-if="showFilter" class="hidden shrink-0 sm:flex rounded-lg border border-primary-200 dark:border-primary-700 bg-surface dark:bg-surface-dark p-0.5">
        <button
          v-for="opt in densityOptions"
          :key="opt.value"
          type="button"
          :title="`${opt.label} view`"
          @click="emit('update:density', opt.value)"
          :class="[
            'inline-flex h-7 w-8 items-center justify-center rounded-md transition-colors',
            density === opt.value
              ? 'bg-accent-500/15 text-accent-700 dark:bg-accent-500/20 dark:text-accent-200'
              : 'text-primary-400 hover:bg-primary-100 hover:text-primary-700 dark:text-primary-500 dark:hover:bg-primary-800 dark:hover:text-primary-200',
          ]"
        >
          <span class="sr-only">{{ opt.label }}</span>
          <component :is="opt.icon" class="h-[18px] w-[18px]" />
        </button>
      </div>
    </div>

    <!-- toolbar -->
    <div v-if="showFilter" class="flex flex-wrap items-center gap-2.5">
      <!-- search -->
      <div class="relative min-w-[200px] flex-1">
        <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-primary-400" />
        <input
          id="search"
          v-model="searchText"
          placeholder="Search images"
          type="text"
          @focusin="emitter.emit('block-hotkeys')"
          @focusout="emitter.emit('unblock-hotkeys')"
          class="h-9 w-full rounded-md border border-primary-200 bg-surface pl-9 pr-9 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600"
        />
        <button
          v-if="searchText"
          type="button"
          @click="searchText = ''"
          class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-primary-400 hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-200"
        >
          <XMarkIcon class="h-4 w-4" />
          <span class="sr-only">Clear search</span>
        </button>
      </div>

      <!-- tags filter -->
      <Popover class="relative">
        <PopoverButton :class="[triggerBase, selectedTags.length ? triggerActive : triggerIdle]">
          <TagIcon class="h-[18px] w-[18px]" />
          <span>Tags</span>
          <span
            v-if="selectedTags.length"
            class="ml-0.5 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-accent-500/20 px-1.5 font-data text-xs font-semibold text-accent-700 dark:text-accent-200"
          >
            {{ selectedTags.length }}
          </span>
          <ChevronDownIcon class="h-4 w-4 opacity-60" />
        </PopoverButton>
        <transition
          enter-active-class="transition duration-150 ease-out"
          enter-from-class="opacity-0 translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 translate-y-1"
        >
          <PopoverPanel
            class="absolute right-0 z-30 mt-2 w-[calc(100vw-2rem)] max-w-72 origin-top-right overflow-hidden rounded-lg border border-primary-200 bg-surface shadow-xl dark:border-primary-700 dark:bg-surface-dark"
          >
            <div class="border-b border-primary-100 p-2 dark:border-primary-800">
              <input
                v-model="tagQuery"
                placeholder="Filter tags…"
                type="text"
                class="h-8 w-full rounded-md border border-primary-200 bg-surface-muted px-2.5 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-primary-100"
              />
            </div>
            <div class="scrollbar-tool max-h-64 overflow-y-auto p-1">
              <button
                v-for="tag in filteredTags"
                :key="tag.id"
                type="button"
                @click="toggleTag(tag)"
                class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-sm text-primary-700 transition-colors hover:bg-primary-100 dark:text-primary-200 dark:hover:bg-primary-800"
              >
                <span
                  :class="[
                    'flex h-4 w-4 flex-shrink-0 items-center justify-center rounded border',
                    isSelected(tag) ? 'border-accent-500 bg-accent-500 text-white' : 'border-primary-300 dark:border-primary-600',
                  ]"
                >
                  <CheckIcon v-if="isSelected(tag)" class="h-3 w-3" />
                </span>
                <span class="truncate">{{ tag.name }}</span>
              </button>
              <p v-if="!filteredTags.length" class="px-2.5 py-6 text-center text-sm text-primary-400">No tags found</p>
            </div>
            <div v-if="selectedTags.length" class="border-t border-primary-100 p-1 dark:border-primary-800">
              <button
                type="button"
                @click="clearTags"
                class="w-full rounded-md px-2.5 py-1.5 text-left text-sm font-medium text-accent-600 hover:bg-primary-100 dark:text-accent-300 dark:hover:bg-primary-800"
              >
                Clear {{ selectedTags.length }} selected
              </button>
            </div>
          </PopoverPanel>
        </transition>
      </Popover>

      <!-- orientation -->
      <Listbox v-model="orientation">
        <div class="relative">
          <ListboxButton :class="[triggerBase, orientation !== 'neutral' ? triggerActive : triggerIdle]">
            <component :is="currentOrientation.icon" class="h-[18px] w-[18px]" />
            <span>{{ currentOrientation.label }}</span>
            <ChevronDownIcon class="h-4 w-4 opacity-60" />
          </ListboxButton>
          <transition leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <ListboxOptions
              class="absolute right-0 z-30 mt-2 w-44 overflow-hidden rounded-lg border border-primary-200 bg-surface p-1 shadow-xl focus:outline-none dark:border-primary-700 dark:bg-surface-dark"
            >
              <ListboxOption v-for="opt in orientationOptions" :key="opt.value" :value="opt.value" v-slot="{ active, selected }">
                <li
                  :class="[
                    'flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-sm',
                    active ? 'bg-primary-100 dark:bg-primary-800' : '',
                    selected ? 'text-accent-700 dark:text-accent-200' : 'text-primary-700 dark:text-primary-200',
                  ]"
                >
                  <component :is="opt.icon" class="h-[18px] w-[18px]" />
                  <span class="flex-1">{{ opt.label }}</span>
                  <CheckIcon v-if="selected" class="h-4 w-4" />
                </li>
              </ListboxOption>
            </ListboxOptions>
          </transition>
        </div>
      </Listbox>

      <!-- sort -->
      <Listbox v-model="preferredImageSortOrder">
        <div class="relative">
          <ListboxButton :class="[triggerBase, triggerIdle]">
            <ArrowsUpDownIcon class="h-[18px] w-[18px]" />
            <span>{{ currentSort.label }}</span>
            <ChevronDownIcon class="h-4 w-4 opacity-60" />
          </ListboxButton>
          <transition leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <ListboxOptions
              class="absolute right-0 z-30 mt-2 w-52 overflow-hidden rounded-lg border border-primary-200 bg-surface p-1 shadow-xl focus:outline-none dark:border-primary-700 dark:bg-surface-dark"
            >
              <ListboxOption v-for="opt in sortOptions" :key="opt.value" :value="opt.value" v-slot="{ active, selected }">
                <li
                  :class="[
                    'flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-sm',
                    active ? 'bg-primary-100 dark:bg-primary-800' : '',
                    selected ? 'text-accent-700 dark:text-accent-200' : 'text-primary-700 dark:text-primary-200',
                  ]"
                >
                  <span class="flex-1">{{ opt.label }}</span>
                  <CheckIcon v-if="selected" class="h-4 w-4" />
                </li>
              </ListboxOption>
            </ListboxOptions>
          </transition>
        </div>
      </Listbox>
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  PhotoIcon,
  MagnifyingGlassIcon,
  XMarkIcon,
  TagIcon,
  ChevronDownIcon,
  CheckIcon,
  ArrowsUpDownIcon,
  Squares2X2Icon,
  ViewColumnsIcon,
  TableCellsIcon,
  RectangleStackIcon,
} from "@heroicons/vue/24/outline";
import { Popover, PopoverButton, PopoverPanel, Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@headlessui/vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { emitter } from "src/boot/mitt";
import { computed, h, ref, watch } from "vue";
import { ImageTag } from "src/types/api";

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
  filterTags: [ImageTag[]];
  aspectRatioFilter: [string];
  "update:density": [Density];
}>();

const { activeProject, preferredImageSortOrder, projectTags } = storeToRefs(useUserStore());

// shared trigger styling so every control aligns to one spec
const triggerBase =
  "inline-flex h-9 items-center gap-1.5 rounded-md border px-3 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-accent-500";
const triggerIdle =
  "border-primary-200 bg-surface text-primary-700 hover:border-primary-300 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600";
const triggerActive = "border-accent-400/60 bg-accent-500/10 text-accent-700 dark:border-accent-400/40 dark:text-accent-200";

// orientation rectangles drawn inline so the aspect reads unambiguously
const rect = (w: number, h0: number) => () =>
  h("svg", { viewBox: "0 0 20 20", fill: "none", class: "h-[18px] w-[18px]" }, [
    h("rect", { x: (20 - w) / 2, y: (20 - h0) / 2, width: w, height: h0, rx: 1.5, stroke: "currentColor", "stroke-width": 1.6 }),
  ]);
const orientationOptions = [
  { value: "neutral", label: "All orientations", icon: Squares2X2Icon },
  { value: "portrait", label: "Portrait", icon: rect(9, 14) },
  { value: "landscape", label: "Landscape", icon: rect(14, 9) },
];
const currentOrientation = computed(() => orientationOptions.find((o) => o.value === orientation.value) || orientationOptions[0]);

const sortOptions = [
  { value: "latestFirst", label: "Latest first" },
  { value: "oldestFirst", label: "Oldest first" },
  { value: "mostRecentlyUpdated", label: "Recently updated" },
  { value: "leastRecentlyUpdated", label: "Least recently updated" },
];
const currentSort = computed(() => sortOptions.find((s) => s.value === preferredImageSortOrder.value) || sortOptions[0]);

const densityOptions: { value: Density; label: string; icon: any }[] = [
  { value: "gallery", label: "Gallery", icon: RectangleStackIcon },
  { value: "comfortable", label: "Grid", icon: Squares2X2Icon },
  { value: "dense", label: "Dense", icon: TableCellsIcon },
];

// search
const searchText = ref("");
watch(searchText, () => emit("search", searchText.value));

// orientation
const orientation = ref<string>("neutral");
watch(orientation, () => emit("aspectRatioFilter", orientation.value));

// tags
const tagQuery = ref("");
const selectedTags = ref<ImageTag[]>([]);
watch(selectedTags, () => emit("filterTags", selectedTags.value), { deep: true });
const selectableTags = computed(() => projectTags.value.filter((t: ImageTag) => t.type !== "template"));
const filteredTags = computed(() => {
  const q = tagQuery.value.toLowerCase();
  return selectableTags.value.filter((t: ImageTag) => t.name.toLowerCase().includes(q));
});
const isSelected = (tag: ImageTag) => selectedTags.value.some((t) => t.id === tag.id);
function toggleTag(tag: ImageTag) {
  selectedTags.value = isSelected(tag) ? selectedTags.value.filter((t) => t.id !== tag.id) : [...selectedTags.value, tag];
}
function clearTags() {
  selectedTags.value = [];
}

// kept for Images.vue: re-sync selected tags when toggling grid/detail
function setFilteredTags(tags: ImageTag[]) {
  selectedTags.value = tags;
}
defineExpose({ setFilteredTags });
</script>
<script lang="ts">
export { SORT_ORDER } from "./sortOrder";
</script>
