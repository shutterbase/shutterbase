<template>
  <div class="select-none" :class="disabled ? 'pointer-events-none opacity-40' : ''">
    <!-- touch-none is required: without it the browser claims a horizontal
         drag for page scrolling and cancels the gesture, so the slider never
         moves on a phone. -->
    <!-- The track div is ALWAYS rendered — it is the element the ref points at
         and the ResizeObserver watches, so gating it on `measured` would make
         measurement impossible (the element that must be measured is the one
         waiting to be measured). Only the VISUALS wait: usableWidth falls back
         to 1px while unmeasured, which would pile the thumbs, ticks and
         selected segment onto x=7 and map the whole track to one endpoint. -->
    <div ref="trackRef" class="relative h-6 cursor-pointer touch-none" @pointerdown="onPointerDown">
      <template v-if="measured">
      <!-- track: inset by the thumb radius so the end thumbs stay inside the panel -->
      <div
        class="absolute top-1/2 h-1 -translate-y-1/2 rounded-full bg-primary-200 dark:bg-primary-700"
        :style="{ left: THUMB_RADIUS + 'px', width: usableWidth + 'px' }"
      ></div>
      <!-- image ticks -->
      <span
        v-for="tick in tickPositions"
        :key="tick.key"
        class="absolute top-1 h-2 w-px bg-primary-300 dark:bg-primary-600"
        :style="{ left: tickPx(tick.pct) + 'px' }"
      ></span>
      <!-- selected segment -->
      <div
        class="absolute top-1/2 z-[1] h-1 -translate-y-1/2 rounded-full bg-accent-500"
        :style="{ left: tickPx(lowPct) + 'px', width: tickPx(highPct) - tickPx(lowPct) + 'px' }"
      ></div>
      <!-- low thumb -->
      <div
        role="slider"
        tabindex="0"
        aria-label="Range start"
        :aria-valuemin="minStep"
        :aria-valuemax="maxStep"
        :aria-valuenow="lowStep"
        :aria-valuetext="describeStep(lowStep)"
        :aria-disabled="disabled ? 'true' : undefined"
        class="absolute top-1/2 z-10 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent-600 shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-400"
        :class="dragging === 'low' ? 'ring-2 ring-accent-400/40' : ''"
        :style="{ left: tickPx(lowPct) + 'px' }"
        data-testid="range-start-thumb"
        @keydown="onKeyDown('low', $event)"
      ></div>
      <!-- high thumb -->
      <div
        role="slider"
        tabindex="0"
        aria-label="Range end"
        :aria-valuemin="minStep"
        :aria-valuemax="maxStep"
        :aria-valuenow="highStep"
        :aria-valuetext="describeStep(highStep)"
        :aria-disabled="disabled ? 'true' : undefined"
        class="absolute top-1/2 z-20 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent-600 shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-400"
        :class="dragging === 'high' ? 'ring-2 ring-accent-400/40' : ''"
        :style="{ left: tickPx(highPct) + 'px' }"
        data-testid="range-end-thumb"
        @keydown="onKeyDown('high', $event)"
      ></div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onUnmounted, ref, watch } from "vue";

const props = defineProps<{
  min: string;
  max: string;
  from?: string | null;
  to?: string | null;
  disabled?: boolean;
  ticks?: string[] | null;
}>();

const emit = defineEmits<{
  // Either side is null only when that side is genuinely unbounded. A thumb the
  // user did not touch comes back out as the parent's own value, so a one-thumb
  // gesture never deletes the other bound.
  change: [string | null, string | null, { replace?: boolean }?];
  preview: [string, string];
  // A gesture ended without committing (pointercancel, window blur): re-sync the
  // parent's inputs to the restored values.
  restore: [string, string];
}>();

const MINUTE = 60_000;
// Half a 14px thumb: the track and the thumb CENTRES both start this far in,
// so the end thumbs sit fully inside the popover panel.
const THUMB_RADIUS = 7;

const domainMin = computed(() => new Date(props.min).getTime());
const domainMax = computed(() => new Date(props.max).getTime());
// The domain is FLOORED to whole minutes, matching the minute granularity the
// thumbs emit, so a thumb can never land outside the density strip the server
// drew. Flooring (not rounding) is what makes the control round-trip stable:
// the committed `to` bound is widened to :59.999 of its minute, and rounding
// that back up would move the thumb a minute further right on every commit —
// pressing ArrowLeft would visibly jump the wrong way.
const minStep = computed(() => Math.floor(domainMin.value / MINUTE));
const maxStep = computed(() => Math.floor(domainMax.value / MINUTE));
const span = computed(() => Math.max(maxStep.value - minStep.value, 1));

const clampStep = (step: number) => Math.min(Math.max(step, minStep.value), maxStep.value);
const toStep = (iso: string | null | undefined, fallback: number) => {
  const ms = iso ? new Date(iso).getTime() : NaN;
  return Number.isNaN(ms) ? fallback : clampStep(Math.floor(ms / MINUTE));
};
const isoForStep = (step: number) => new Date(step * MINUTE).toISOString();
// Screen readers announce aria-valuetext verbatim, so a raw ISO instant would be
// read as "2026-08-11T10:00:00.000Z". Say it the way the inputs do.
const describeStep = (step: number) =>
  new Date(step * MINUTE).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });

const lowStep = ref(toStep(props.from, minStep.value));
const highStep = ref(toStep(props.to, maxStep.value));
// Which thumbs the user has actually moved. Drives the null-vs-ISO decision on
// commit, so an untouched thumb leaves its side of the range open.
const touched = ref({ low: false, high: false });

watch(
  () => [props.min, props.max, props.from, props.to],
  () => {
    lowStep.value = toStep(props.from, minStep.value);
    highStep.value = toStep(props.to, maxStep.value);
    if (lowStep.value > highStep.value) {
      const mid = lowStep.value;
      lowStep.value = highStep.value;
      highStep.value = mid;
    }
    touched.value = { low: false, high: false };
  },
);

watch(lowStep, (v) => {
  if (v > highStep.value) highStep.value = v;
});
watch(highStep, (v) => {
  if (v < lowStep.value) lowStep.value = v;
});

const lowPct = computed(() => ((lowStep.value - minStep.value) / span.value) * 100);
const highPct = computed(() => ((highStep.value - minStep.value) / span.value) * 100);

// --- geometry ---
const trackRef = ref<HTMLElement | null>(null);
const trackWidth = ref(0);
const measured = computed(() => trackWidth.value > 2 * THUMB_RADIUS);
const usableWidth = computed(() => Math.max(trackWidth.value - 2 * THUMB_RADIUS, 1));
/** Percentage on the track -> pixel offset for the thumb/tick centre. */
const tickPx = (pct: number) => THUMB_RADIUS + (pct / 100) * usableWidth.value;

// Image ticks: map each sampled ISO timestamp to a percentage position on
// the slider track. Matches the UploadTimeline.vue density strip pattern.
interface TickPosition { key: string; pct: number }
const tickPositions = computed<TickPosition[]>(() => {
  const ticks = props.ticks;
  if (!ticks || ticks.length === 0) return [];
  const domainSpan = maxStep.value - minStep.value;
  if (domainSpan <= 0) return [];
  return ticks.map((iso) => {
    const ms = new Date(iso).getTime();
    const pct = ((ms / MINUTE - minStep.value) / domainSpan) * 100;
    return { key: iso, pct: Math.max(0, Math.min(100, pct)) };
  });
});

// --- pointer drag ---
const dragging = ref<"low" | "high" | null>(null);
// The active drag's target, and its listener teardown. `detach` must NOT be the
// function that calls it — that is an infinite recursion.
let activeTarget: "low" | "high" | null = null;
let detach: (() => void) | null = null;

function stepFromClientX(clientX: number): number {
  const rect = trackRef.value?.getBoundingClientRect();
  if (!rect) return minStep.value;
  const usable = Math.max(rect.width - 2 * THUMB_RADIUS, 1);
  const pct = Math.max(0, Math.min(1, (clientX - rect.left - THUMB_RADIUS) / usable));
  return clampStep(Math.round(minStep.value + pct * (maxStep.value - minStep.value)));
}

function applyStep(target: "low" | "high", step: number) {
  if (target === "low") lowStep.value = step;
  else highStep.value = step;
  touched.value = { ...touched.value, [target]: true };
  emit("preview", isoForStep(lowStep.value), isoForStep(highStep.value));
}

// A side the user never touched must go back out EXACTLY as it came in.
// Emitting null for it would mean "no bound on that side": dragging or arrowing
// ONE thumb of an existing 10:00-12:00 range would delete the other bound from
// the route and silently widen the filter to an open end.
//
// This matters for the cross-clamp too: watch(lowStep)/watch(highStep) move the
// other thumb when they cross, but do not mark it touched — so the untouched
// side is still the parent's value, which is what we want.
function commit(opts?: { replace?: boolean }) {
  const from = touched.value.low ? isoForStep(lowStep.value) : (props.from ?? null);
  const to = touched.value.high ? isoForStep(highStep.value) : (props.to ?? null);
  emit("change", from, to, opts);
}

const onDragMove = (ev: PointerEvent) => {
  if (activeTarget) applyStep(activeTarget, stepFromClientX(ev.clientX));
};
const onDragEnd = () => {
  detach?.();
  commit();
};
// A cancelled gesture (browser takes the pointer for a scroll, a system
// gesture, a lost capture) must tear the drag down exactly like a release —
// otherwise `dragging` stays pinned to a thumb and the listeners leak. It must
// also UNDO: `preview` has already moved the thumbs and, through the header, the
// datetime inputs, so tearing down without restoring leaves the panel showing a
// range the route never applied.
const onDragCancel = () => {
  detach?.();
  restoreFromProps();
};

// Put the thumbs back where the route says they are, and tell the parent to
// re-sync its inputs to match.
function restoreFromProps() {
  lowStep.value = toStep(props.from, minStep.value);
  highStep.value = toStep(props.to, maxStep.value);
  if (lowStep.value > highStep.value) {
    const mid = lowStep.value;
    lowStep.value = highStep.value;
    highStep.value = mid;
  }
  touched.value = { low: false, high: false };
  emit("restore", isoForStep(lowStep.value), isoForStep(highStep.value));
}

function onPointerDown(e: PointerEvent) {
  if (props.disabled) return;
  detach?.(); // defensive: never stack two drag sessions
  trackRef.value?.setPointerCapture?.(e.pointerId);

  const clickStep = stepFromClientX(e.clientX);
  // pick the closest thumb
  const distLow = Math.abs(clickStep - lowStep.value);
  const distHigh = Math.abs(clickStep - highStep.value);
  activeTarget = distLow <= distHigh ? "low" : "high";
  dragging.value = activeTarget;
  applyStep(activeTarget, clickStep);

  detach = () => {
    window.removeEventListener("pointermove", onDragMove);
    window.removeEventListener("pointerup", onDragEnd);
    window.removeEventListener("pointercancel", onDragCancel);
    window.removeEventListener("blur", onDragCancel);
    activeTarget = null;
    dragging.value = null;
    detach = null;
  };
  window.addEventListener("pointermove", onDragMove);
  window.addEventListener("pointerup", onDragEnd);
  window.addEventListener("pointercancel", onDragCancel);
  window.addEventListener("blur", onDragCancel);
}

function onKeyDown(target: "low" | "high", e: KeyboardEvent) {
  if (props.disabled) return;
  const current = target === "low" ? lowStep.value : highStep.value;
  let next: number;
  switch (e.key) {
    case "ArrowLeft":
    case "ArrowDown":
      next = current - 1;
      break;
    case "ArrowRight":
    case "ArrowUp":
      next = current + 1;
      break;
    case "PageDown":
      next = current - 5;
      break;
    case "PageUp":
      next = current + 5;
      break;
    case "Home":
      next = minStep.value;
      break;
    case "End":
      next = maxStep.value;
      break;
    default:
      return;
  }
  e.preventDefault();
  e.stopPropagation();
  next = clampStep(next);
  if (target === "low") lowStep.value = next;
  else highStep.value = next;
  touched.value = { ...touched.value, [target]: true };
  emit("preview", isoForStep(lowStep.value), isoForStep(highStep.value));
  commit({ replace: true });
}

// Track geometry is measured, not hard-coded: the popover sizes itself, so the
// width is only known after layout. A ResizeObserver (not a window resize
// listener) because the popover opening and the window resizing are the same
// event as far as this component is concerned.
let observer: ResizeObserver | null = null;
const measure = () => {
  trackWidth.value = trackRef.value?.clientWidth ?? 0;
};
watch(
  trackRef,
  (el) => {
    observer?.disconnect();
    observer = null;
    if (!el) return;
    measure();
    observer = new ResizeObserver(measure);
    observer.observe(el);
  },
  { immediate: true, flush: "post" },
);

// The window listeners above outlive the component unless the unmount hook
// removes them: closing the popover mid-drag used to leak both of them and
// fire `change` after unmount.
onUnmounted(() => {
  detach?.();
  activeTarget = null;
  dragging.value = null;
  observer?.disconnect();
  observer = null;
});
</script>
