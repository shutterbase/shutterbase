<template>
  <section class="mt-8">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-lg font-semibold tracking-tight text-primary-900 dark:text-white">Tagging timeline</h2>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">
          Your schedule items land here automatically — drag the in/out points, stack extra tag lanes, then apply.
        </p>
      </div>

      <div class="flex items-center gap-2" v-if="!readonly">
        <!-- add lane -->
        <Menu as="div" class="relative">
          <MenuButton
            class="inline-flex h-9 cursor-pointer items-center gap-1.5 rounded-md border border-primary-200 bg-surface px-3 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white"
          >
            <PlusIcon class="h-4 w-4" />
            Add lane
          </MenuButton>
          <MenuItems
            class="absolute right-0 z-20 mt-1 max-h-80 w-64 overflow-y-auto rounded-md border border-primary-200 bg-surface py-1 shadow-panel focus:outline-none dark:border-primary-700 dark:bg-surface-dark dark:shadow-panel-dark"
          >
            <p v-if="addableItems.length" class="label-mono px-3 pb-1 pt-2 text-primary-400">Schedule items</p>
            <MenuItem v-for="entry in addableItems" :key="entry.id" v-slot="{ active }">
              <button
                type="button"
                :class="['flex w-full cursor-pointer items-center gap-2 px-3 py-1.5 text-left text-sm', active ? 'bg-accent-500/10 text-accent-700 dark:text-accent-200' : 'text-primary-700 dark:text-primary-200']"
                @click="addItemLane(entry)"
              >
                <CalendarDaysIcon class="h-4 w-4 flex-shrink-0 text-primary-400" />
                <span class="truncate">{{ entry.title }}</span>
              </button>
            </MenuItem>
            <p class="label-mono px-3 pb-1 pt-2 text-primary-400">Tags</p>
            <MenuItem v-for="tag in addableTags" :key="tag.id" v-slot="{ active }">
              <button
                type="button"
                :class="['flex w-full cursor-pointer items-center gap-2 px-3 py-1.5 text-left text-sm', active ? 'bg-accent-500/10 text-accent-700 dark:text-accent-200' : 'text-primary-700 dark:text-primary-200']"
                @click="addTagLane(tag)"
              >
                <TagIcon class="h-4 w-4 flex-shrink-0 text-primary-400" />
                <span class="truncate">{{ tag.name }}</span>
              </button>
            </MenuItem>
          </MenuItems>
        </Menu>

        <button
          type="button"
          :disabled="!dirty || applying || hasScheduleOverlap(tracks)"
          class="inline-flex h-9 cursor-pointer items-center gap-1.5 rounded-md bg-accent-600 px-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50"
          @click="apply"
        >
          <CheckIcon class="h-4 w-4" />
          {{ applying ? "Applying…" : "Apply tags" }}
        </button>
      </div>
    </div>

    <div v-if="tracks.length === 0" class="mt-4 rounded-md border border-dashed border-primary-200 px-4 py-6 text-center text-sm text-primary-400 dark:border-primary-700">
      No lanes yet — your assigned schedule items appear here once images with timestamps are in, or add a lane manually.
    </div>

    <div v-else class="mt-4 select-none rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark">
      <!-- image density strip -->
      <div class="relative ml-36 h-4">
        <span
          v-for="timed in timedImages"
          :key="timed.id"
          class="absolute top-1 h-2 w-px bg-primary-300 dark:bg-primary-600"
          :style="{ left: `${pct(timed.time)}%` }"
        ></span>
      </div>

      <!-- lanes -->
      <div
        v-for="tr in tracks"
        :key="tr.key"
        class="flex items-center gap-2 py-1.5"
        @click="selectedKey = tr.key"
      >
        <div class="flex w-34 flex-shrink-0 items-center gap-1.5 overflow-hidden" style="width: 8.5rem">
          <component :is="tr.scheduleItemId ? CalendarDaysIcon : TagIcon" class="h-3.5 w-3.5 flex-shrink-0 text-primary-400" />
          <span class="truncate text-xs font-medium text-primary-700 dark:text-primary-200" :title="tr.label">{{ tr.label }}</span>
        </div>
        <div class="relative h-9 flex-1 rounded bg-primary-100/60 dark:bg-primary-900/40" :ref="(el) => registerLane(tr.key, el as HTMLElement | null)">
          <!-- hour ticks -->
          <span v-for="tick in hourTicks" :key="tick.time" class="absolute inset-y-0 w-px bg-primary-200/70 dark:bg-primary-800/70" :style="{ left: `${tick.pct}%` }"></span>
          <!-- the bar -->
          <div
            :data-testid="`lane-${tr.key}`"
            :class="[
              'absolute inset-y-1 rounded-md border text-[10px] leading-none transition-shadow',
              tr.scheduleItemId
                ? 'border-accent-500 bg-accent-500/25 text-accent-800 dark:text-accent-100'
                : 'border-sky-500 bg-sky-500/20 text-sky-800 dark:text-sky-100',
              tr.enabled ? '' : 'opacity-35 saturate-0',
              selectedKey === tr.key ? 'shadow-md ring-1 ring-accent-500/70' : '',
              readonly ? '' : 'cursor-pointer',
            ]"
            :style="{ left: `${pct(tr.start)}%`, width: `${Math.max(pct(tr.end) - pct(tr.start), 0.5)}%` }"
            tabindex="0"
            @keydown="onKeydown(tr, $event)"
            @click.stop="selectedKey = tr.key"
          >
            <span class="pointer-events-none absolute inset-x-2 top-1/2 -translate-y-1/2 truncate font-medium">
              {{ formatTime(tr.start) }}–{{ formatTime(tr.end) }} · {{ coveredCount(tr) }} photos
            </span>
            <!-- in/out handles (transcript 22:31: like a Premiere/DaVinci clip) -->
            <template v-if="!readonly">
              <span
                class="absolute inset-y-0 -left-1 w-2.5 cursor-ew-resize rounded-l-md bg-current/0 hover:bg-current/20"
                @pointerdown.stop.prevent="startDrag(tr, 'start', $event)"
              ></span>
              <span
                class="absolute inset-y-0 -right-1 w-2.5 cursor-ew-resize rounded-r-md bg-current/0 hover:bg-current/20"
                @pointerdown.stop.prevent="startDrag(tr, 'end', $event)"
              ></span>
            </template>
          </div>
        </div>
      </div>

      <!-- hour labels (transcript 26:47: real clock underneath) -->
      <div class="relative ml-36 mt-1 h-4">
        <span
          v-for="tick in hourTicks"
          :key="tick.time"
          class="absolute -translate-x-1/2 text-[10px] tabular-nums text-primary-400"
          :style="{ left: `${tick.pct}%` }"
        >
          {{ tick.label }}
        </span>
      </div>

      <!-- selected lane controls + boundary previews -->
      <div v-if="selected" class="mt-4 border-t border-primary-100 pt-3 dark:border-primary-800">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium text-primary-700 dark:text-primary-200">{{ selected.label }}</span>
          <span class="text-xs tabular-nums text-primary-400">{{ formatTime(selected.start) }} – {{ formatTime(selected.end) }}</span>
          <span v-if="!selected.enabled" class="rounded-full bg-primary-100 px-2 py-0.5 text-[10px] font-medium text-primary-500 dark:bg-primary-800 dark:text-primary-300">disabled</span>
          <template v-if="!readonly">
            <button type="button" class="lane-btn" @click="toggleEnabled(selected)">
              {{ selected.enabled ? "Disable" : "Enable" }}
            </button>
            <button type="button" class="lane-btn" @click="expand(selected)">
              <ArrowsRightLeftIcon class="h-3.5 w-3.5" />
              Expand
            </button>
            <button v-if="selected.tagId" type="button" class="lane-btn text-red-600 dark:text-red-400" @click="removeLane(selected)">
              <TrashIcon class="h-3.5 w-3.5" />
              Remove lane
            </button>
            <span class="text-[10px] text-primary-400">⇧←/→ in-point · ←/→ out-point · ⌥ ×10</span>
          </template>
        </div>

        <!-- boundary previews (transcript 23:06): what falls in and out at the edges -->
        <div class="mt-3 flex items-end gap-3 overflow-x-auto">
          <figure v-for="entry in boundaryPreviews" :key="entry.role" class="w-24 flex-shrink-0 text-center">
            <img v-if="entry.src" :src="entry.src" :alt="entry.role" :class="['h-16 w-24 rounded-md object-cover', entry.inside ? 'ring-2 ring-accent-500' : 'opacity-50 grayscale']" />
            <div v-else class="flex h-16 w-24 items-center justify-center rounded-md border border-dashed border-primary-200 text-[10px] text-primary-300 dark:border-primary-700">—</div>
            <figcaption class="mt-1 text-[10px] text-primary-400">{{ entry.role }}<template v-if="entry.time"> · {{ formatTime(entry.time) }}</template></figcaption>
          </figure>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/vue";
import { ArrowsRightLeftIcon, CalendarDaysIcon, CheckIcon, PlusIcon, TagIcon, TrashIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { computed, ref, watch } from "vue";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import { ImageTag, ScheduleItem, Upload } from "src/types/api";
import { Image } from "src/util/fileProcessor";
import {
  EditorTrack,
  TimedImage,
  addScheduleTrack,
  addTagTrack,
  boundaryImages,
  expandTrack,
  hasScheduleOverlap,
  imagesInTrack,
  initialTracks,
  moveEdge,
  setEnabled,
  timelineWindow,
  toApiTracks,
} from "src/util/uploadTimeline";

const props = defineProps<{
  upload: Upload;
  images: Image[];
  readonly: boolean;
}>();

const emit = defineEmits<{ applied: [Upload] }>();

// --- context data ------------------------------------------------------------

const myItems = ref<ScheduleItem[]>([]);
const allItems = ref<ScheduleItem[]>([]);
const projectTags = ref<ImageTag[]>([]);

async function loadContext() {
  try {
    const projectId = props.upload.project.id;
    const [mine, all, tags] = await Promise.all([
      api.scheduleItems.list({ projectId, mine: true, limit: 500, sort: "start", order: "asc" }),
      api.scheduleItems.list({ projectId, limit: 500, sort: "start", order: "asc" }),
      api.imageTags.list({ projectId, limit: 500, sort: "name", order: "asc" }),
    ]);
    myItems.value = mine.items;
    allItems.value = all.items;
    projectTags.value = tags.items;
    seedTracks();
  } catch {
    // The timeline is an enhancement on this page — the upload itself must
    // keep working without it, so context errors stay silent.
  }
}

// --- track state ---------------------------------------------------------------

const tracks = ref<EditorTrack[]>([]);
const selectedKey = ref<string | null>(null);
const dirty = ref(false);
const applying = ref(false);
let seeded = false;

const timedImages = computed<TimedImage[]>(() =>
  props.images.filter((i) => i.id && i.correctedTime).map((i) => ({ id: i.id as string, time: (i.correctedTime as DateTime).toMillis() })),
);

const imageById = computed(() => {
  const map = new Map<string, Image>();
  props.images.forEach((i) => i.id && map.set(i.id, i));
  return map;
});

const labels = {
  scheduleItem: (id: string) => allItems.value.find((i) => i.id === id)?.title ?? "Schedule item",
  tag: (id: string) => projectTags.value.find((t) => t.id === id)?.name ?? "Tag",
};

// Seed once: persisted timeline wins; otherwise pre-populate my items when the
// first timed images exist. Re-seeds when images arrive later (fresh upload).
function seedTracks() {
  if (seeded && tracks.value.length > 0) return;
  const initial = initialTracks(props.upload.timeline, myItems.value, timedImages.value, labels);
  if (initial.length > 0) {
    tracks.value = initial;
    seeded = true;
  }
}
watch(timedImages, seedTracks);
watch(() => props.upload.id, loadContext, { immediate: true });

const selected = computed(() => tracks.value.find((t) => t.key === selectedKey.value) ?? null);

const window_ = computed(() => timelineWindow(timedImages.value, tracks.value));
const span = computed(() => window_.value.end - window_.value.start);
const pct = (time: number) => ((time - window_.value.start) / span.value) * 100;
const formatTime = (ms: number) => DateTime.fromMillis(ms).toFormat("HH:mm");

const hourTicks = computed(() => {
  const ticks: { time: number; pct: number; label: string }[] = [];
  const stepMs = span.value > 20 * 3_600_000 ? 4 * 3_600_000 : span.value > 8 * 3_600_000 ? 2 * 3_600_000 : 3_600_000;
  let t = Math.ceil(window_.value.start / stepMs) * stepMs;
  for (; t < window_.value.end; t += stepMs) {
    ticks.push({ time: t, pct: pct(t), label: DateTime.fromMillis(t).toFormat("HH:mm") });
  }
  return ticks;
});

const coveredCount = (track: EditorTrack) => imagesInTrack(timedImages.value, track).length;

// --- add / remove lanes ----------------------------------------------------------

const addableItems = computed(() => allItems.value.filter((i) => !tracks.value.some((t) => t.scheduleItemId === i.id)));
const addableTags = computed(() => projectTags.value.filter((t) => t.type !== "template"));

function markDirty() {
  dirty.value = true;
}

function addItemLane(item: ScheduleItem) {
  tracks.value = addScheduleTrack(tracks.value, item);
  selectedKey.value = `s${item.id}`;
  markDirty();
}

function addTagLane(tag: ImageTag) {
  tracks.value = addTagTrack(tracks.value, tag.id, tag.name, timedImages.value, window_.value);
  selectedKey.value = tracks.value[tracks.value.length - 1].key;
  markDirty();
}

function removeLane(track: EditorTrack) {
  tracks.value = tracks.value.filter((t) => t.key !== track.key);
  if (selectedKey.value === track.key) selectedKey.value = null;
  markDirty();
}

function replaceTrack(next: EditorTrack) {
  tracks.value = tracks.value.map((t) => (t.key === next.key ? next : t));
  markDirty();
}

function toggleEnabled(track: EditorTrack) {
  const next = setEnabled(track, !track.enabled, tracks.value);
  if (!next) {
    showNotificationToast({ headline: "Overlaps another schedule lane — move it first", type: "error" });
    return;
  }
  replaceTrack(next);
}

function expand(track: EditorTrack) {
  replaceTrack(expandTrack(track, tracks.value, window_.value));
}

// --- drag + keyboard --------------------------------------------------------------

const laneEls = new Map<string, HTMLElement>();
function registerLane(key: string, el: HTMLElement | null) {
  if (el) laneEls.set(key, el);
  else laneEls.delete(key);
}

function startDrag(track: EditorTrack, edge: "start" | "end", event: PointerEvent) {
  if (props.readonly) return;
  selectedKey.value = track.key;
  const lane = laneEls.get(track.key);
  if (!lane) return;
  const rect = lane.getBoundingClientRect();
  const onMove = (e: PointerEvent) => {
    const ratio = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
    const to = window_.value.start + ratio * span.value;
    const current = tracks.value.find((t) => t.key === track.key);
    if (current) replaceTrack(moveEdge(current, edge, to, tracks.value, window_.value));
  };
  const onUp = () => {
    document.removeEventListener("pointermove", onMove);
    document.removeEventListener("pointerup", onUp);
  };
  document.addEventListener("pointermove", onMove);
  document.addEventListener("pointerup", onUp);
}

// Keyboard nudging (transcript 26:15): arrows = out-point, shift = in-point,
// alt = coarse steps. Handled on the focused lane bar.
function onKeydown(track: EditorTrack, event: KeyboardEvent) {
  if (props.readonly) return;
  if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
  event.preventDefault();
  const step = (event.altKey ? 10 : 1) * 60_000 * (event.key === "ArrowLeft" ? -1 : 1);
  const edge = event.shiftKey ? "start" : "end";
  const to = (edge === "start" ? track.start : track.end) + step;
  replaceTrack(moveEdge(track, edge, to, tracks.value, window_.value));
}

// --- boundary previews ---------------------------------------------------------------

const boundaryPreviews = computed(() => {
  if (!selected.value) return [];
  const bounds = boundaryImages(timedImages.value, selected.value);
  const src = (timed?: TimedImage) => {
    if (!timed) return undefined;
    const image = imageById.value.get(timed.id);
    if (!image) return undefined;
    return image.thumbnail ? `data:image/jpeg;base64, ${image.thumbnail}` : image.downloadUrls?.["256"];
  };
  return [
    { role: "before", src: src(bounds.before), time: bounds.before?.time, inside: false },
    { role: "first in", src: src(bounds.first), time: bounds.first?.time, inside: true },
    { role: "last in", src: src(bounds.last), time: bounds.last?.time, inside: true },
    { role: "after", src: src(bounds.after), time: bounds.after?.time, inside: false },
  ];
});

// --- apply --------------------------------------------------------------------------

async function apply() {
  applying.value = true;
  try {
    const result = await api.uploads.applyTimeline(props.upload.id, toApiTracks(tracks.value));
    dirty.value = false;
    showNotificationToast({
      headline: `Tags applied — ${result.applied?.created ?? 0} added, ${result.applied?.deleted ?? 0} removed`,
      type: "success",
    });
    emit("applied", result);
  } catch (error: any) {
    const code = error?.response?.data?.code;
    showNotificationToast({
      headline: code === "schedule_track_overlap" ? "Schedule lanes overlap — fix the timeline first" : "Applying the timeline failed",
      type: "error",
    });
  } finally {
    applying.value = false;
  }
}
</script>

<style scoped>
.lane-btn {
  @apply inline-flex cursor-pointer items-center gap-1 rounded-md border border-primary-200 bg-surface px-2 py-1 text-xs font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white;
}
</style>
