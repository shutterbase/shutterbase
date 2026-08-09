<template>
  <div class="fixed inset-0 z-50 bg-black" data-testid="slideshow-overlay" @mousemove="showControls" @click="showControls">
    <!-- setup phase: configure, then start -->
    <div v-if="phase === 'setup'" class="flex h-full items-center justify-center px-4">
      <div class="w-full max-w-md rounded-xl border border-primary-800 bg-primary-950 p-6" data-testid="slideshow-setup" @click.stop>
        <h2 class="display text-xl text-white">Slideshow</h2>
        <p class="mt-1 text-sm text-primary-400">{{ totalCount.toLocaleString() }} images in the current view</p>

        <div class="mt-5 space-y-4">
          <label class="block">
            <span class="text-xs font-medium uppercase tracking-wide text-primary-400">Show time (seconds per image)</span>
            <input
              v-model.number="config.showSeconds"
              type="number"
              min="1"
              max="120"
              step="0.5"
              data-testid="slideshow-show-seconds"
              class="mt-1 w-full rounded-md border border-primary-700 bg-primary-900 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
            />
          </label>
          <label class="block">
            <span class="text-xs font-medium uppercase tracking-wide text-primary-400">Transition time (seconds)</span>
            <input
              v-model.number="config.transitionSeconds"
              type="number"
              min="0"
              max="10"
              step="0.25"
              class="mt-1 w-full rounded-md border border-primary-700 bg-primary-900 px-3 py-2 text-sm text-white focus:border-accent-500 focus:outline-none"
            />
          </label>
          <div>
            <span class="text-xs font-medium uppercase tracking-wide text-primary-400">Pan &amp; zoom</span>
            <div class="mt-1 flex rounded-lg border border-primary-700 p-0.5">
              <button
                v-for="option in kenBurnsOptions"
                :key="option.value"
                type="button"
                :class="[
                  'flex-1 rounded-md px-3 py-1.5 text-sm transition-colors',
                  config.kenBurns === option.value ? 'bg-accent-500/20 text-accent-200' : 'text-primary-400 hover:text-primary-200',
                ]"
                @click="config.kenBurns = option.value"
              >
                {{ option.label }}
              </button>
            </div>
          </div>
          <label class="flex items-center gap-2 text-sm text-primary-200">
            <input v-model="config.loop" type="checkbox" class="h-4 w-4 rounded border-primary-700 bg-primary-900 accent-accent-500" />
            Loop endlessly
          </label>
        </div>

        <div class="mt-6 flex justify-end gap-3">
          <button type="button" class="rounded-md px-4 py-2 text-sm text-primary-300 hover:text-white" @click="emit('close')">Cancel</button>
          <button
            type="button"
            data-testid="slideshow-start"
            class="inline-flex items-center gap-2 rounded-md bg-accent-500 px-4 py-2 text-sm font-semibold text-primary-950 hover:bg-accent-400"
            @click="start"
          >
            <PlayIcon class="h-4 w-4" />
            Start
          </button>
        </div>
      </div>
    </div>

    <!-- playing phase: stacked slides, crossfade + optional Ken Burns -->
    <template v-else>
      <div
        v-for="slide in visibleSlides"
        :key="slide.image.id"
        class="absolute inset-0 flex items-center justify-center transition-opacity ease-linear"
        :style="{ opacity: slide.index === current ? 1 : 0, transitionDuration: `${config.transitionSeconds}s` }"
      >
        <!-- kb class stays on the slide through its fade-out as "previous" —
             stripping it while visible cancels the animation and snaps the
             transform back to identity mid-crossfade -->
        <img
          :src="slideSrc(slide.image)"
          :alt="slide.image.computedFileName"
          class="h-full w-full object-contain"
          :class="config.kenBurns !== 'off' ? `kb kb-${kenBurnsVariant(slide.index)}` : ''"
          :style="kenBurnsStyle"
          @error="onSlideError(slide.image)"
        />
      </div>

      <div v-if="waiting" class="absolute inset-x-0 top-4 flex justify-center" data-testid="slideshow-buffering">
        <span class="rounded-full bg-black/60 px-3 py-1 text-xs text-primary-300">buffering…</span>
      </div>

      <!-- controls: fade with the cursor -->
      <div
        :class="['absolute inset-x-0 bottom-0 flex items-center gap-4 bg-black/70 px-4 py-2 transition-opacity', controlsVisible ? 'opacity-100' : 'pointer-events-none opacity-0']"
        data-testid="slideshow-controls"
        @click.stop
      >
        <button type="button" class="text-primary-300 hover:text-white" title="Previous" @click="goPrevious">
          <BackwardIcon class="h-5 w-5" />
        </button>
        <button type="button" class="text-primary-300 hover:text-white" :title="paused ? 'Play' : 'Pause'" data-testid="slideshow-play-pause" @click="togglePause">
          <PlayIcon v-if="paused" class="h-6 w-6" />
          <PauseIcon v-else class="h-6 w-6" />
        </button>
        <button type="button" class="text-primary-300 hover:text-white" title="Next" @click="goNext">
          <ForwardIcon class="h-5 w-5" />
        </button>
        <span class="min-w-0 flex-1 truncate text-center font-data text-sm text-primary-200">{{ images[current]?.computedFileName }}</span>
        <span class="label-mono-sm shrink-0 text-primary-400" data-testid="slideshow-position">{{ current + 1 }} / {{ totalCount.toLocaleString() }}</span>
        <button type="button" class="shrink-0 text-primary-300 hover:text-white" title="Exit slideshow" data-testid="slideshow-exit" @click="emit('close')">
          <XMarkIcon class="h-6 w-6" />
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useStorage } from "@vueuse/core";
import { PlayIcon, PauseIcon, ForwardIcon, BackwardIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { ImageWithTagsType } from "src/types/custom";
import { devPlaceholder } from "src/util/devPlaceholder";
import { DEFAULT_SLIDESHOW_CONFIG, SlideshowConfig, kenBurnsVariant, nextSlideIndex, previousSlideIndex, preloadIndices, shouldFetchMore } from "src/util/slideshow";

const props = defineProps<{
  images: ImageWithTagsType[];
  totalCount: number;
}>();
const emit = defineEmits<{ close: []; needMore: [] }>();

const config = useStorage<SlideshowConfig>("slideshow-config", { ...DEFAULT_SLIDESHOW_CONFIG }, undefined, { mergeDefaults: true });
const kenBurnsOptions = [
  { value: "off", label: "Off" },
  { value: "subtle", label: "Subtle" },
  { value: "strong", label: "Strong" },
] as const;

const phase = ref<"setup" | "playing">("setup");
const current = ref(0);
const paused = ref(false);
const waiting = ref(false);

// Current + previous slide stay mounted so the crossfade has both layers.
const previous = ref<number | null>(null);
const visibleSlides = computed(() => {
  const slides: { index: number; image: ImageWithTagsType }[] = [];
  if (previous.value !== null && previous.value !== current.value && props.images[previous.value]) {
    slides.push({ index: previous.value, image: props.images[previous.value] });
  }
  if (props.images[current.value]) {
    slides.push({ index: current.value, image: props.images[current.value] });
  }
  return slides;
});

// Ken Burns runs slightly longer than the slide is on screen so it never halts visibly.
const kenBurnsStyle = computed(() => ({
  "--kb-duration": `${config.value.showSeconds + 2 * config.value.transitionSeconds}s`,
  "--kb-scale": config.value.kenBurns === "strong" ? "1.14" : "1.06",
}));

// --- preloading -------------------------------------------------------------
// A slide only goes on screen once its 2048 rendition is decoded: `loaded`
// tracks finished URLs, `pending` the in-flight Image() handles. Broken
// renditions fall back to the dev placeholder and count as loaded — the show
// must go on either way.

const fallbacks = reactive<Record<string, string>>({});
const loaded = new Set<string>();
const pending = new Map<string, Promise<void>>();

function slideSrc(image: ImageWithTagsType): string {
  return fallbacks[image.id] ?? image.downloadUrls?.["2048"] ?? "";
}

function onSlideError(image: ImageWithTagsType) {
  const placeholder = devPlaceholder(image.id);
  if (placeholder && fallbacks[image.id] !== placeholder) fallbacks[image.id] = placeholder;
}

function preload(image: ImageWithTagsType | undefined): Promise<void> {
  if (!image) return Promise.resolve();
  const src = slideSrc(image);
  if (!src || loaded.has(src)) return Promise.resolve();
  const inFlight = pending.get(src);
  if (inFlight) return inFlight;
  const promise = new Promise<void>((resolve) => {
    const img = new Image();
    img.onload = () => {
      loaded.add(src);
      resolve();
    };
    img.onerror = () => {
      onSlideError(image);
      loaded.add(src); // fallback renders instead; never stall the show
      resolve();
    };
    img.src = src;
  }).finally(() => pending.delete(src));
  pending.set(src, promise);
  return promise;
}

function fillPreloadWindow() {
  for (const index of preloadIndices(current.value, props.images.length)) {
    void preload(props.images[index]);
  }
  if (shouldFetchMore(current.value, props.images.length, props.totalCount)) {
    emit("needMore");
  }
}

// --- playback ---------------------------------------------------------------

let timer: ReturnType<typeof setTimeout> | null = null;

function clearTimer() {
  if (timer) {
    clearTimeout(timer);
    timer = null;
  }
}

function scheduleAdvance() {
  clearTimer();
  if (paused.value) return;
  timer = setTimeout(advance, (config.value.showSeconds + config.value.transitionSeconds) * 1000);
}

// navToken cancels an in-flight decode-gated navigation: pause and any newer
// navigation bump it, so a superseded advance can never land afterwards.
let navToken = 0;

// Every path onto the screen funnels through goTo: decode-gate the target,
// then show it — auto-advance, manual next AND previous all get the same
// "never show an undecoded slide" guarantee.
async function goTo(index: number | null) {
  if (index === null) {
    emit("close");
    return;
  }
  if (index === current.value) {
    // single-image show (loop wraps onto itself): nothing to change, keep the
    // timer alive. The Ken Burns run stays static here — the DOM node never
    // remounts — which is acceptable for a one-image slideshow.
    scheduleAdvance();
    return;
  }
  const token = ++navToken;
  const target = props.images[index];
  const src = target ? slideSrc(target) : "";
  if (src && !loaded.has(src)) {
    waiting.value = true;
    await preload(target);
    if (token === navToken) waiting.value = false;
  }
  if (token !== navToken) return; // superseded by pause or a newer navigation
  show(index);
}

function advance() {
  void goTo(nextSlideIndex(current.value, props.images.length, config.value.loop));
}

function show(index: number) {
  previous.value = current.value;
  current.value = index;
  fillPreloadWindow();
  scheduleAdvance();
}

async function start() {
  // The number inputs can yield NaN/0/negatives — clamped here, a bad show time
  // would otherwise turn the advance timer into a busy loop.
  const clamp = (v: unknown, min: number, max: number, fallback: number) => {
    const n = Number(v);
    return Number.isFinite(n) ? Math.min(max, Math.max(min, n)) : fallback;
  };
  config.value.showSeconds = clamp(config.value.showSeconds, 1, 120, DEFAULT_SLIDESHOW_CONFIG.showSeconds);
  config.value.transitionSeconds = clamp(config.value.transitionSeconds, 0, 10, DEFAULT_SLIDESHOW_CONFIG.transitionSeconds);
  phase.value = "playing";
  // Best effort: a browser may reject fullscreen without a fresh user gesture.
  document.documentElement.requestFullscreen?.().catch(() => undefined);
  // The first slide is decode-gated like every later one; the buffering chip
  // covers the wait.
  waiting.value = true;
  await preload(props.images[0]);
  waiting.value = false;
  fillPreloadWindow();
  scheduleAdvance();
}

function togglePause() {
  paused.value = !paused.value;
  if (paused.value) {
    clearTimer();
    navToken++; // cancel an advance that is already past its timer, mid-preload
    waiting.value = false; // the cancelled advance can no longer clear its own chip
  } else {
    scheduleAdvance();
  }
}

function goNext() {
  clearTimer();
  advance();
}

function goPrevious() {
  clearTimer();
  void goTo(previousSlideIndex(current.value, props.images.length, config.value.loop));
}

// more images may have arrived while we were at the end of the loaded list —
// and the list can also shrink (image deleted from the grid underneath).
watch(
  () => props.images.length,
  (length) => {
    if (current.value >= length && length > 0) current.value = length - 1;
    fillPreloadWindow();
  },
);

// --- controls / keyboard ----------------------------------------------------

const controlsVisible = ref(true);
let controlsTimer: ReturnType<typeof setTimeout> | null = null;
function showControls() {
  controlsVisible.value = true;
  if (controlsTimer) clearTimeout(controlsTimer);
  controlsTimer = setTimeout(() => (controlsVisible.value = false), 2500);
}

function onKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case "Escape":
      emit("close");
      break;
    case " ":
      event.preventDefault();
      if (phase.value === "playing") togglePause();
      break;
    case "ArrowRight":
      if (phase.value === "playing") goNext();
      break;
    case "ArrowLeft":
      if (phase.value === "playing") goPrevious();
      break;
  }
}

onMounted(() => {
  window.addEventListener("keydown", onKeydown);
  showControls();
});

onBeforeUnmount(() => {
  clearTimer();
  if (controlsTimer) clearTimeout(controlsTimer);
  window.removeEventListener("keydown", onKeydown);
  if (document.fullscreenElement) document.exitFullscreen?.().catch(() => undefined);
});
</script>

<style scoped>
.kb {
  animation-duration: var(--kb-duration);
  animation-timing-function: ease-in-out;
  animation-fill-mode: forwards;
}
.kb-0 {
  animation-name: kb-zoom-in;
}
.kb-1 {
  animation-name: kb-pan-right;
}
.kb-2 {
  animation-name: kb-zoom-out;
}
.kb-3 {
  animation-name: kb-pan-left;
}
@keyframes kb-zoom-in {
  from {
    transform: scale(1);
  }
  to {
    transform: scale(var(--kb-scale));
  }
}
@keyframes kb-zoom-out {
  from {
    transform: scale(var(--kb-scale));
  }
  to {
    transform: scale(1);
  }
}
@keyframes kb-pan-right {
  from {
    transform: scale(var(--kb-scale)) translateX(-1.5%);
  }
  to {
    transform: scale(var(--kb-scale)) translateX(1.5%);
  }
}
@keyframes kb-pan-left {
  from {
    transform: scale(var(--kb-scale)) translateX(1.5%);
  }
  to {
    transform: scale(var(--kb-scale)) translateX(-1.5%);
  }
}
</style>
