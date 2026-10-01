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
        :tabindex="disabled ? -1 : 0"
        aria-label="Range start"
        :aria-valuemin="minStep"
        :aria-valuemax="highStep"
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
        :tabindex="disabled ? -1 : 0"
        aria-label="Range end"
        :aria-valuemin="lowStep"
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
// Which thumb the pointer owns right now, if any. Declared with the rest of the
// gesture state because the props watcher below has to consult it.
const dragging = ref<"low" | "high" | null>(null);

// Assign both thumbs from the route, in the order the route states them. Does
// NOT touch `touched` — whether an inbound range ends the current gesture is
// the caller's decision (a pure min/max tick update must not).
//
// Deliberately NOT sorted: this used to swap an inverted incoming pair, so a
// link (or a stale history entry) carrying from=14:00&to=12:00 displayed as a
// perfectly plausible 12:00-14:00 slider while the filter stayed inverted and
// the backend answered 400 invalid_time_range. Keeping the pair as given makes
// the control agree with the URL, and commit() below repairs the order the
// moment the user touches anything.
function syncThumbsFromProps() {
  lowStep.value = toStep(props.from, minStep.value);
  highStep.value = toStep(props.to, maxStep.value);
}

watch(
  () => [props.min, props.max, props.from, props.to],
  (_next, prev) => {
    // A bounds/tick response can land mid-drag: the popover-open fetch is not
    // awaited before the pointer goes down, so re-syncing here snapped the
    // thumb out from under the pointer and the release then committed the
    // parent's values, losing the gesture outright.
    if (dragging.value !== null) return;
    const domainChanged = props.min !== prev[0] || props.max !== prev[1];
    const rangeChanged = props.from !== prev[2] || props.to !== prev[3];
    if (!domainChanged && !rangeChanged) return;
    syncThumbsFromProps();
    // A pure domain update that did not move either thumb must NOT clear
    // `touched`: doing so made the next commit fall back to the parent's values
    // and swallow the in-flight adjustment.
    if (rangeChanged) touched.value = { low: false, high: false };
  },
);

// The thumbs cannot pass through each other. Returns the side that had to be
// dragged along, so the caller can mark it touched: it DID move, and a commit
// that left it out went out straddled — from=14:00, to=12:00, which the backend
// rejects with 400 invalid_time_range, turning the whole grid into an error
// page.
//
// This runs where a thumb is MOVED, never on a props re-sync: a route that
// arrives inverted is shown as it is (see syncThumbsFromProps) and only the
// user's own gesture drags the other thumb.
function crossClamp(moved: "low" | "high"): "low" | "high" | null {
  if (moved === "low" && lowStep.value > highStep.value) {
    highStep.value = lowStep.value;
    return "high";
  }
  if (moved === "high" && highStep.value < lowStep.value) {
    lowStep.value = highStep.value;
    return "low";
  }
  return null;
}

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
  return ticks.map((iso, i) => {
    const ms = new Date(iso).getTime();
    const pct = ((ms / MINUTE - minStep.value) / domainSpan) * 100;
    // Indexed, not the bare ISO: two photos can share a capturedAtCorrected
    // (the sampler returns every row up to maxTicks), and a duplicate v-for key
    // drops a node and logs a Vue warning.
    return { key: `${iso}-${i}`, pct: Math.max(0, Math.min(100, pct)) };
  });
});

// --- pointer drag ---
// The active drag's target, its pointer, and the listener teardown. `detach`
// must NOT be the function that calls it — that is an infinite recursion.
let activeTarget: "low" | "high" | null = null;
// The pointer that started the gesture. A second finger on a phone is a
// separate pointer with its own move and up events: without this filter it
// dragged the range around and committed it on release.
let activePointerId: number | null = null;
let detach: (() => void) | null = null;

function stepFromClientX(clientX: number): number {
  const rect = trackRef.value?.getBoundingClientRect();
  if (!rect) return minStep.value;
  const usable = Math.max(rect.width - 2 * THUMB_RADIUS, 1);
  const pct = Math.max(0, Math.min(1, (clientX - rect.left - THUMB_RADIUS) / usable));
  return clampStep(Math.round(minStep.value + pct * (maxStep.value - minStep.value)));
}

// Move a thumb, clamp it against the other one, record what moved and announce
// the live pair. Every gesture funnels through here (pointer and keyboard), so
// the cross-clamp cannot be bypassed by one of them.
function applyStep(target: "low" | "high", step: number) {
  if (target === "low") lowStep.value = step;
  else highStep.value = step;
  const dragged = crossClamp(target);
  // A thumb the cross-clamp carried is touched too — it is part of the gesture,
  // and dropping it is what used to emit the straddled pair.
  touched.value = dragged ? { low: true, high: true } : { ...touched.value, [target]: true };
  emit("preview", isoForStep(lowStep.value), isoForStep(highStep.value));
}

// A side the user never touched must go back out EXACTLY as it came in.
// Emitting null for it would mean "no bound on that side": dragging or arrowing
// ONE thumb of an existing 10:00-12:00 range would delete the other bound from
// the route and silently widen the filter to an open end. applyStep keeps that
// true — it only widens `touched` when the clamp actually carried a thumb.
function commit(opts?: { replace?: boolean }) {
  const fromTouched = touched.value.low;
  const toTouched = touched.value.high;
  let from = fromTouched ? isoForStep(lowStep.value) : (props.from ?? null);
  let to = toTouched ? isoForStep(highStep.value) : (props.to ?? null);
  // Backstop for a pair the thumbs cannot explain: the untouched side comes
  // straight back out of props, and props are live (the route can change under
  // an in-flight drag), so a value that lands before the dragged thumb would go
  // out straddled. A TOUCHED side is the user's decision and wins — the other
  // side is pulled onto it, the same rule the header applies to an inverted
  // range it types by hand. An already-ordered pair passes through untouched, so
  // an untouched bound is never rewritten.
  if (from && to && new Date(to).getTime() < new Date(from).getTime()) {
    if (fromTouched === toTouched) {
      // no gesture to defer to: just order the pair
      const mid = from;
      from = to;
      to = mid;
    } else if (fromTouched) {
      to = from;
    } else {
      from = to;
    }
  }
  emit("change", from, to, opts);
}

const onDragMove = (ev: PointerEvent) => {
  if (activeTarget && ev.pointerId === activePointerId) applyStep(activeTarget, stepFromClientX(ev.clientX));
};
const onDragEnd = (ev: PointerEvent) => {
  if (ev.pointerId !== activePointerId) return;
  detach?.();
  commit();
};
// A cancelled gesture (browser takes the pointer for a scroll, a system
// gesture, a lost capture) must tear the drag down exactly like a release —
// otherwise `dragging` stays pinned to a thumb and the listeners leak. It must
// also UNDO: `preview` has already moved the thumbs and, through the header, the
// datetime inputs, so tearing down without restoring leaves the panel showing a
// range the route never applied.
const onDragCancel = (ev: PointerEvent) => {
  if (ev.pointerId !== activePointerId) return;
  detach?.();
  restoreFromProps();
};
// Losing window focus (Alt-Tab, a native menu) is not a pointer event at all, so
// it carries no pointerId and must be handled on its own.
const onWindowBlur = () => {
  detach?.();
  restoreFromProps();
};

// Put the thumbs back where the route says they are, and tell the parent to
// re-sync its inputs to match.
//
// The emit carries the route's OWN bounds, not the thumb positions: an
// open-ended range has no thumb on one side, and `toStep(null, minStep)` put the
// DOMAIN edge in the parent's From/To input. The panel then showed a bound the
// route does not carry, and the "Clear time range" button appeared for a range
// that was never applied.
function restoreFromProps() {
  syncThumbsFromProps();
  touched.value = { low: false, high: false };
  emit("restore", props.from ?? "", props.to ?? "");
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
  activePointerId = e.pointerId;
  dragging.value = activeTarget;
  applyStep(activeTarget, clickStep);

  detach = () => {
    window.removeEventListener("pointermove", onDragMove);
    window.removeEventListener("pointerup", onDragEnd);
    window.removeEventListener("pointercancel", onDragCancel);
    window.removeEventListener("blur", onWindowBlur);
    activeTarget = null;
    activePointerId = null;
    dragging.value = null;
    detach = null;
  };
  window.addEventListener("pointermove", onDragMove);
  window.addEventListener("pointerup", onDragEnd);
  window.addEventListener("pointercancel", onDragCancel);
  window.addEventListener("blur", onWindowBlur);
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
  // applyStep, not a bare assignment: the keyboard commits in the SAME handler,
  // so a cross-clamp that only ran on the next tick would emit the straddled
  // pair before it ever got the chance.
  applyStep(target, clampStep(next));
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
  activePointerId = null;
  dragging.value = null;
  observer?.disconnect();
  observer = null;
});
</script>
