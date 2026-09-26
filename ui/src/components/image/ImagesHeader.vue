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

      <div class="flex shrink-0 items-center gap-2">
        <!-- hotkey help -->
        <button
          type="button"
          title="Keyboard shortcuts (?)"
          @click="emitter.emit('show-hotkey-help')"
          class="hidden sm:inline-flex h-8 w-8 items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:text-primary-500 dark:hover:bg-primary-800 dark:hover:text-primary-200"
        >
          <span class="sr-only">Keyboard shortcuts</span>
          <QuestionMarkCircleIcon class="h-5 w-5" />
        </button>

        <!-- slideshow over the current (filtered) view -->
        <button
          v-if="showFilter && totalImageCount > 0"
          type="button"
          title="Start slideshow"
          data-testid="start-slideshow"
          class="hidden sm:inline-flex h-8 w-8 items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:text-primary-500 dark:hover:bg-primary-800 dark:hover:text-primary-200"
          @click="emit('slideshow')"
        >
          <span class="sr-only">Start slideshow</span>
          <PlayIcon class="h-5 w-5" />
        </button>

        <!-- density / view -->
        <div v-if="showFilter" class="hidden sm:flex rounded-lg border border-primary-200 dark:border-primary-700 bg-surface dark:bg-surface-dark p-0.5">
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
    </div>

    <!-- toolbar -->
    <div v-if="showFilter" class="flex flex-wrap items-center gap-2.5">
      <!-- multi-select AI rerun -->
      <button
        v-if="selectionCount > 0"
        type="button"
        :class="[triggerBase, triggerActive]"
        :title="`Rerun AI detection on ${selectionCount} selected images`"
        @click="emit('rerunAi')"
      >
        <SparklesIcon class="h-[18px] w-[18px]" />
        <span>Rerun AI ({{ selectionCount }})</span>
      </button>

      <!-- search -->
      <div class="relative min-w-[200px] flex-1">
        <MagnifyingGlassIcon class="pointer-events-none absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-primary-400" />
        <input
          id="search"
          v-model="searchText"
          placeholder="Search images · Enter = describe what you're looking for"
          type="text"
          @keydown.enter.prevent="semanticSearch()"
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
      <button
        type="button"
        :class="[triggerBase, searchText.trim() ? triggerIdle : 'cursor-default opacity-50']"
        :disabled="!searchText.trim()"
        title="Semantic search: describe the moment you're looking for (e.g. “team celebrating in the rain”) — ranks photos by their AI description"
        @click="semanticSearch()"
      >
        <SparklesIcon class="h-[18px] w-[18px]" />
        <span>Ask</span>
      </button>

      <!-- tags filter -->
      <Popover class="relative" v-slot="{ open }">
        <PopoverButton :class="[triggerBase, selectedTags.length ? triggerActive : triggerIdle]" @click="!open && emit('facetsNeeded', true)">
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
            class="absolute right-0 z-30 mt-2 w-[calc(100vw-2rem)] max-w-80 origin-top-right overflow-hidden rounded-lg border border-primary-200 bg-surface shadow-xl dark:border-primary-700 dark:bg-surface-dark"
          >
            <div class="border-b border-primary-100 p-2 dark:border-primary-800">
              <input
                v-model="tagQuery"
                placeholder="Filter tags…"
                type="text"
                class="h-8 w-full rounded-md border border-primary-200 bg-surface-muted px-2.5 text-sm text-primary-900 placeholder:text-primary-400 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-primary-100"
              />
            </div>
            <!-- active filters: pinned above the list, immune to the text filter -->
            <div v-if="selectedTags.length" class="flex flex-wrap gap-1.5 border-b border-primary-100 p-2 dark:border-primary-800">
              <button
                v-for="entry in selectedTags"
                :key="entry.tag.id"
                type="button"
                :aria-label="`Remove filter ${tagLabel(entry.tag)}`"
                :title="entry.exclude ? 'Excluded — click to remove' : 'Included — click to remove'"
                @click="removeTagFilter(entry)"
                :class="[
                  'group inline-flex max-w-full items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium transition-colors',
                  entry.exclude
                    ? 'border-error-400/60 bg-error-500/10 text-error-700 hover:border-error-500 dark:border-error-400/40 dark:text-error-300'
                    : 'border-success-400/60 bg-success-500/10 text-success-700 hover:border-success-500 dark:border-success-400/40 dark:text-success-300',
                ]"
              >
                <span class="font-semibold">{{ entry.exclude ? "−" : "+" }}</span>
                <span class="truncate">{{ tagLabel(entry.tag) }}</span>
                <XMarkIcon class="h-3.5 w-3.5 shrink-0 opacity-60 group-hover:opacity-100" />
              </button>
            </div>
            <div class="scrollbar-tool max-h-64 overflow-y-auto p-1">
              <div
                v-for="tag in visibleTags"
                :key="tag.id"
                class="group flex w-full items-center gap-1 rounded-md px-2.5 py-1.5 text-left text-sm text-primary-700 transition-colors hover:bg-primary-100 dark:text-primary-200 dark:hover:bg-primary-800"
              >
                <span class="min-w-0 flex-1 truncate">{{ tagLabel(tag) }}</span>
                <button
                  type="button"
                  :aria-label="`Include ${tagLabel(tag)}`"
                  :title="`Show only images with ${tagLabel(tag)}`"
                  :disabled="includeCount(tag) === 0"
                  @click="addTagFilter(tag, false)"
                  class="flex items-center gap-0.5 rounded p-1 text-primary-400 transition-colors hover:bg-success-500/15 hover:text-success-600 disabled:cursor-default disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-primary-400 dark:text-primary-500 dark:hover:text-success-300"
                >
                  <MagnifyingGlassPlusIcon class="h-4 w-4" />
                  <span v-if="includeCount(tag) !== null" class="font-data text-[10px] tabular-nums">{{ includeCount(tag) }}</span>
                </button>
                <button
                  type="button"
                  :aria-label="`Exclude ${tagLabel(tag)}`"
                  :title="`Hide images with ${tagLabel(tag)}`"
                  :disabled="excludeCount(tag) === 0"
                  @click="addTagFilter(tag, true)"
                  class="flex items-center gap-0.5 rounded p-1 text-primary-400 transition-colors hover:bg-error-500/15 hover:text-error-600 disabled:cursor-default disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-primary-400 dark:text-primary-500 dark:hover:text-error-300"
                >
                  <MagnifyingGlassMinusIcon class="h-4 w-4" />
                  <span v-if="excludeCount(tag) !== null" class="font-data text-[10px] tabular-nums">{{ excludeCount(tag) }}</span>
                </button>
              </div>
              <p v-if="!visibleTags.length" class="px-2.5 py-6 text-center text-sm text-primary-400">No tags found</p>
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

      <!-- time range -->
      <Popover class="relative" v-slot="{ open: timeOpen }">
        <PopoverButton
          :class="[triggerBase, timeFrom || timeTo ? triggerActive : triggerIdle]"
          data-testid="time-range-button"
          title="Filter by capture time"
          @click="!timeOpen && emit('timeBoundsNeeded')"
        >
          <ClockIcon class="h-[18px] w-[18px]" />
          <span>Time</span>
          <span
            v-if="timeFrom || timeTo"
            class="ml-0.5 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-accent-500/20 px-1.5 font-data text-xs font-semibold text-accent-700 dark:text-accent-200"
          >
            ·
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
            class="absolute right-0 z-30 mt-2 w-64 rounded-lg border border-primary-200 bg-surface p-3 shadow-xl dark:border-primary-700 dark:bg-surface-dark"
            data-testid="time-range-panel"
          >
            <div class="flex flex-col gap-2.5">
              <!-- quick way: two thumbs over the currently filtered gallery's
                   time span (excluding the range itself). Hidden when the
                   domain is empty/degenerate; manual inputs stay the override. -->
              <TimeRangeSlider
                v-if="sliderDomain"
                :min="sliderDomain.min"
                :max="sliderDomain.max"
                :from="timeFrom"
                :to="timeTo"
                :ticks="timeTicks"
                :disabled="timeRangeSuspended"
                @preview="(f, t) => setLocalsSilently(f, t)"
                @restore="(f, t) => setLocalsSilently(f, t)"
                @change="(f, t, o) => emit('timeRange', f, widenToEndOfMinute(t), o)"
              />
              <label class="flex flex-col gap-1 text-xs font-medium text-primary-500 dark:text-primary-400">
                From
                <input
                  v-model="fromLocal"
                  type="datetime-local"
                  data-testid="time-from-input"
                  class="h-8 rounded-md border border-primary-200 bg-surface-muted px-2.5 text-sm text-primary-900 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-primary-100"
                />
              </label>
              <label class="flex flex-col gap-1 text-xs font-medium text-primary-500 dark:text-primary-400">
                To
                <input
                  v-model="toLocal"
                  type="datetime-local"
                  data-testid="time-to-input"
                  class="h-8 rounded-md border border-primary-200 bg-surface-muted px-2.5 text-sm text-primary-900 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-primary-900 dark:text-primary-100"
                />
              </label>
              <button
                v-if="timeFrom || timeTo || fromLocal || toLocal"
                class="rounded-md px-2.5 py-1.5 text-left text-sm font-medium text-accent-600 hover:bg-primary-100 dark:text-accent-300 dark:hover:bg-primary-800"
                data-testid="clear-time-range"
                @click="clearTimeRange()"
              >
                Clear time range
              </button>
            </div>
          </PopoverPanel>
        </transition>
      </Popover>

      <!-- upload batch -->
      <Listbox :model-value="uploadFilter" @update:model-value="onUploadSelect">
        <div class="relative">
          <ListboxButton :class="[triggerBase, uploadFilter ? triggerActive : triggerIdle]">
            <ArrowUpTrayIcon class="h-[18px] w-[18px]" />
            <span class="max-w-36 truncate">{{ currentUploadLabel }}</span>
            <ChevronDownIcon class="h-4 w-4 opacity-60" />
          </ListboxButton>
          <transition leave-active-class="transition duration-100 ease-in" leave-from-class="opacity-100" leave-to-class="opacity-0">
            <ListboxOptions
              class="scrollbar-tool absolute right-0 z-30 mt-2 max-h-72 w-64 overflow-y-auto rounded-lg border border-primary-200 bg-surface p-1 shadow-xl focus:outline-none dark:border-primary-700 dark:bg-surface-dark"
            >
              <ListboxOption v-for="opt in uploadOptions" :key="opt.value ?? 'all'" :value="opt.value" v-slot="{ active, selected }">
                <li
                  :class="[
                    'flex cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-sm',
                    active ? 'bg-primary-100 dark:bg-primary-800' : '',
                    selected ? 'text-accent-700 dark:text-accent-200' : 'text-primary-700 dark:text-primary-200',
                  ]"
                >
                  <span class="flex-1 truncate">{{ opt.label }}</span>
                  <CheckIcon v-if="selected" class="h-4 w-4 shrink-0" />
                </li>
              </ListboxOption>
            </ListboxOptions>
          </transition>
        </div>
      </Listbox>

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

      <!-- sort. `sortOrder` is the EFFECTIVE order: the timespan context pins
           ?sort=oldestFirst for one view without rewriting the user's persisted
           preference, so the trigger must show the order actually in force —
           binding straight to the store made the label relabel to a choice that
           then did nothing. Picking an option emits instead of writing the
           store, so the page can clear the route pin. -->
      <Listbox :model-value="effectiveSortOrder" @update:model-value="onSortSelect">
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
  MagnifyingGlassPlusIcon,
  MagnifyingGlassMinusIcon,
  QuestionMarkCircleIcon,
  XMarkIcon,
  TagIcon,
  ChevronDownIcon,
  CheckIcon,
  ArrowsUpDownIcon,
  Squares2X2Icon,
  ViewColumnsIcon,
  TableCellsIcon,
  RectangleStackIcon,
  SparklesIcon,
  ArrowUpTrayIcon,
  PlayIcon,
  ClockIcon,
} from "@heroicons/vue/24/outline";
import { Popover, PopoverButton, PopoverPanel, Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@headlessui/vue";
import { storeToRefs } from "pinia";
import { useDebounceFn } from "@vueuse/core";
import { useUserStore } from "src/stores/user-store";
import { emitter } from "src/boot/mitt";
import { computed, h, nextTick, ref, watch } from "vue";
import { ImageTag, Upload } from "src/types/api";
import type { TagFacetsResponse, ImageTimeBounds } from "src/api/images";
import { tagLabel } from "src/util/tagOrder";
import { isoToEndOfMinute, isoToLocalInput, localInputToIso, localInputToIsoInclusive } from "src/util/dateTimeUtil";
import TimeRangeSlider from "src/components/image/TimeRangeSlider.vue";
import { api } from "src/api";

type Density = "gallery" | "comfortable" | "dense";

interface Props {
  totalImageCount: number;
  showFilter: boolean;
  density?: Density;
  // multi-selected image count — enables the "Rerun AI" toolbar action
  selectionCount?: number;
  // active upload-batch filter — route-driven, Images.vue owns the query sync
  uploadFilter?: string | null;
  // inclusive capture-time bounds (ISO) — route-driven like uploadFilter
  timeFrom?: string | null;
  timeTo?: string | null;
  // slider domain for the Time popover — fetched on popover open
  timeBounds?: ImageTimeBounds | null;
  // density ticks for the slider track — fetched alongside bounds
  timeTicks?: string[] | null;
  // the range is currently suspended: the slider greys out so the thumbs
  // cannot be dragged into a filter that is not being applied
  timeRangeSuspended?: boolean;
  // the sort order actually in force; omit to follow the persisted preference
  sortOrder?: string | null;
  // per-tag counts under the current filter — Images.vue fetches on facetsNeeded
  tagFacets?: TagFacetsResponse | null;
  // any narrowing filter active (search, tags, orientation, person, upload, ask)
  filterActive?: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  totalImageCount: 0,
  density: "comfortable",
  selectionCount: 0,
  uploadFilter: null,
  timeFrom: null,
  timeTo: null,
  timeBounds: null,
  timeTicks: null,
  timeRangeSuspended: false,
  sortOrder: null,
  tagFacets: null,
  filterActive: false,
});

const emit = defineEmits<{
  search: [string];
  semanticSearch: [string];
  filterTags: [{ include: ImageTag[]; exclude: ImageTag[] }];
  aspectRatioFilter: [string];
  "update:density": [Density];
  rerunAi: [];
  uploadFilter: [string | null];
  timeRange: [string | null, string | null, { replace?: boolean }?];
  timeBoundsNeeded: [];
  sortOrderChange: [string];
  slideshow: [];
  // true = force a refresh (popover open), false = only if the filter changed
  facetsNeeded: [boolean];
}>();

const { activeProject, preferredImageSortOrder, projectTags } = storeToRefs(useUserStore());

// slider domain: only render when both ends exist and span at least a minute
const sliderDomain = computed(() => {
  const b = props.timeBounds;
  if (!b?.min || !b?.max) return null;
  return new Date(b.max).getTime() - new Date(b.min).getTime() >= 60_000 ? { min: b.min, max: b.max } : null;
});

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
const effectiveSortOrder = computed(() => props.sortOrder || preferredImageSortOrder.value);
const currentSort = computed(() => sortOptions.find((s) => s.value === effectiveSortOrder.value) || sortOptions[0]);
// The page owns the route: it clears a pinned ?sort= and writes the preference.
function onSortSelect(value: string) {
  emit("sortOrderChange", value);
}

const densityOptions: { value: Density; label: string; icon: any }[] = [
  { value: "gallery", label: "Gallery", icon: RectangleStackIcon },
  { value: "comfortable", label: "Grid", icon: Squares2X2Icon },
  { value: "dense", label: "Dense", icon: TableCellsIcon },
];

// search
const searchText = ref("");
watch(searchText, () => emit("search", searchText.value));
const semanticSearch = () => {
  const q = searchText.value.trim();
  if (q) emit("semanticSearch", q);
};

// orientation
const orientation = ref<string>("neutral");
watch(orientation, () => emit("aspectRatioFilter", orientation.value));

// time range — props are the source of truth (route-driven in Images.vue);
// the datetime-local inputs work in local wall clock, so convert both ways.
// A props->locals sync must NOT echo back out as an emit: the round trip goes
// through Images.vue's setTimeRange, which is a USER-EDIT writer (it drops
// ?rangeScope=). The flag marks exactly one inbound sync as non-emitting.
//
// The inputs start EMPTY when no range is applied. Prefilling them with the
// slider domain bounds (first/last photo) meant that editing only "From"
// silently committed ?to=<last photo> — a range the user never asked for, on a
// panel whose "Clear time range" button was hidden because no range existed.
const fromLocal = ref(isoToLocalInput(props.timeFrom));
const toLocal = ref(isoToLocalInput(props.timeTo));
let syncingFromProps = false;
// Bumped by every INBOUND sync (props change, slider preview or restore) so a
// debounced keystroke that was already in flight knows it has been overtaken and
// must not write its stale value over the newer state.
let inboundToken = 0;
// A token the user actually typed, as opposed to one that arrived from outside.
// An inbound sync that was NOT a slider preview (a props change, or a slider
// restore) must not eat a pending edit: the user typed it, it is still the
// truth, and the only thing that changed the fields under them is state the
// route does not own. Those syncs re-arm the debounce instead of dropping it.
let pendingUserEdit = false;
function applyInbound(fLocal: string, tLocal: string, reArmPendingEdit = false) {
  if (reArmPendingEdit) pendingUserEdit = true;
  inboundToken++;
  syncingFromProps = true;
  if (fLocal !== fromLocal.value || tLocal !== toLocal.value) {
    fromLocal.value = fLocal;
    toLocal.value = tLocal;
  }
  nextTick(() => (syncingFromProps = false));
}
watch(
  () => [props.timeFrom, props.timeTo],
  ([f, t]) => applyInbound(isoToLocalInput(f), isoToLocalInput(t), true),
);

// Typing is debounced and lands as a history REPLACE: emitting on every
// keystroke meant one router.push + one full loadImages(true) per character,
// and every intermediate value ("2026-0", "2026-08") became a back-step the
// user had to click through.
const emitTimeRange = useDebounceFn((token: number) => {
  const stale = token !== inboundToken;
  // A stale token is only forgivable when an inbound sync re-armed the edit.
  if (stale && !pendingUserEdit) return;
  pendingUserEdit = false;
  const from = localInputToIso(fromLocal.value);
  // Inclusive upper bound: the input and the slider both carry minute
  // precision, and the backend compares `to` with LTE — without widening to the
  // last millisecond of the minute, every photo inside that final minute is
  // dropped.
  let to = localInputToIsoInclusive(toLocal.value);
  // The backend rejects an inverted range with 400 invalid_time_range, which
  // renders the whole grid as an error page. Clamp instead: a user whose To
  // lands before the From gets a valid range, not a dead end.
  if (from && to && new Date(to) < new Date(from)) {
    // Clamp to the END of the From minute, not its start, and show the user the
    // value that was actually applied — an unwidened `to === from` is a
    // one-instant range (an empty grid) while the input still shows the earlier
    // time they typed, which reads as the app ignoring them.
    to = widenToEndOfMinute(from);
    applyInbound(fromLocal.value, isoToLocalInput(to));
  }
  emit("timeRange", from, to, { replace: true });
}, 400);
watch([fromLocal, toLocal], () => {
  if (syncingFromProps) return;
  pendingUserEdit = true;
  emitTimeRange(inboundToken);
});

// slider drag feedback: mirror the thumbs into the inputs so their values
// update live — silently, like a props sync (the actual commit is `change`)
function setLocalsSilently(f: string, t: string) {
  applyInbound(isoToLocalInput(f), isoToLocalInput(t));
}

// The slider works in whole minutes, so its `to` needs the same inclusive
// widening as the manual input — otherwise releasing the thumb at 23:59 cuts
// the last minute off the range. A null side stays null: that is the slider
// signalling an open-ended range, and widening would close it. Delegates to
// localInputToIsoInclusive so there is ONE end-of-minute rule, not two that can
// drift.
function widenToEndOfMinute(iso: string | null): string | null {
  return isoToEndOfMinute(iso);
}

// Clear from the panel: the locals are reset through the same inbound path
// (so the pending debounce is invalidated) and the route is cleared directly,
// so the button works even when only the inputs (not the URL) carried a range.
function clearTimeRange() {
  applyInbound("", "");
  emit("timeRange", null, null);
}

// tags — Grafana-style polarity filter: + narrows to images WITH the tag,
// − drops images WITH the tag. Selected entries pin above the list.
interface TagFilterEntry {
  tag: ImageTag;
  exclude: boolean;
}
const tagQuery = ref("");
const selectedTags = ref<TagFilterEntry[]>([]);
watch(selectedTags, () => {
  emit("filterTags", {
    include: selectedTags.value.filter((e) => !e.exclude).map((e) => e.tag),
    exclude: selectedTags.value.filter((e) => e.exclude).map((e) => e.tag),
  });
  // after filterTags, so the facet fetch reads the already-updated filter state
  emit("facetsNeeded", false);
});
const selectableTags = computed(() => projectTags.value.filter((t: ImageTag) => t.type !== "template"));
const filteredTags = computed(() => {
  const q = tagQuery.value.toLowerCase();
  return selectableTags.value.filter((t: ImageTag) => !isSelected(t) && (t.name.toLowerCase().includes(q) || tagLabel(t).toLowerCase().includes(q)));
});
// under an active filter, tags that would produce an empty result set disappear;
// with no filter every tag stays offered (zero-count tags are normal pre-event)
const visibleTags = computed(() => {
  if (!props.filterActive || !props.tagFacets) return filteredTags.value;
  return filteredTags.value.filter((t: ImageTag) => isSelected(t) || (props.tagFacets!.facets[t.id] ?? 0) > 0);
});
// images left when adding the tag as +/− filter; null while facets are loading
const includeCount = (tag: ImageTag): number | null => (props.tagFacets ? (props.tagFacets.facets[tag.id] ?? 0) : null);
const excludeCount = (tag: ImageTag): number | null => {
  const inc = includeCount(tag);
  return inc === null ? null : props.tagFacets!.total - inc;
};
const isSelected = (tag: ImageTag) => selectedTags.value.some((e) => e.tag.id === tag.id);
function addTagFilter(tag: ImageTag, exclude: boolean) {
  selectedTags.value = [...selectedTags.value, { tag, exclude }];
}
function removeTagFilter(entry: TagFilterEntry) {
  selectedTags.value = selectedTags.value.filter((e) => e.tag.id !== entry.tag.id);
}
function clearTags() {
  selectedTags.value = [];
}

// upload batch — the picker renders and emits; Images.vue maps the selection
// onto the route query (?upload=), mirroring the person filter.
// ponytail: one unpaginated fetch — revisit if a project ever exceeds 500 uploads
const uploads = ref<Upload[]>([]);
watch(
  () => activeProject.value?.id,
  async (projectId) => {
    uploads.value = [];
    if (!projectId) return;
    try {
      const items = (await api.uploads.list({ projectId, limit: 500 })).items;
      uploads.value = items.sort((a, b) => new Date(b.updatedAt ?? 0).getTime() - new Date(a.updatedAt ?? 0).getTime());
    } catch {
      // picker degrades to "All uploads"; deep links keep working without it
    }
  },
  { immediate: true },
);
const uploadOptions = computed(() => [{ value: null as string | null, label: "All uploads" }, ...uploads.value.map((u) => ({ value: u.id, label: u.name }))]);
const currentUploadLabel = computed(() => uploads.value.find((u) => u.id === props.uploadFilter)?.name ?? (props.uploadFilter ? "1 upload" : "Upload"));
function onUploadSelect(id: string | null) {
  emit("uploadFilter", id);
}

// kept for Images.vue: re-sync selected tags when toggling grid/detail
function setFilteredTags(include: ImageTag[], exclude: ImageTag[]) {
  selectedTags.value = [...include.map((tag) => ({ tag, exclude: false })), ...exclude.map((tag) => ({ tag, exclude: true }))];
}
defineExpose({ setFilteredTags });
</script>
<script lang="ts">
export { SORT_ORDER } from "./sortOrder";
</script>
