<template>
  <!-- Unzoomed: w-fit hugs the fitted img, so the clip stays at the image's
       bounds. Zoomed with an expandClass: the container becomes that stage
       (e.g. fixed between header bar and film strip) and the image can extend
       across it, clipped only by the stage. -->
  <div
    ref="container"
    class="overflow-hidden"
    :class="[stageActive ? expandClass : 'relative w-fit max-w-full', zoomed ? (panning ? 'cursor-grabbing touch-none select-none' : 'cursor-grab touch-none select-none') : '']"
    @wheel.prevent="onWheel"
    @dblclick="onDblClick"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @click.capture="onClickCapture"
  >
    <!-- relative + w-fit so slotted overlays (face boxes) keep their percentage
         positions relative to the img and zoom along with it -->
    <div class="relative w-fit origin-top-left" :style="{ transform: `translate(${zoom.tx}px, ${zoom.ty}px) scale(${zoom.scale})` }">
      <img ref="imgEl" :src="src" :alt="alt" :class="imgClass" :style="freezeStyle" draggable="false" @load="onImgLoad" @error="emit('error')" />
      <!-- progressive hi-res: mounted (= fetching) on first zoom, revealed only
           once fully loaded and only while zoomed, exactly covering the base
           img — the base stays underneath, so the swap never flickers -->
      <img
        v-if="hiresSrc && (hires === 'loading' || hires === 'ready')"
        v-show="hires === 'ready' && zoomed"
        :src="hiresSrc"
        alt=""
        aria-hidden="true"
        class="absolute inset-0 h-full w-full"
        draggable="false"
        @load="hires = 'ready'"
        @error="hires = 'failed'"
      />
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { ZOOM_RESET, panBy, zoomAt } from "src/util/zoom";

interface Props {
  src: string;
  alt?: string;
  imgClass?: string;
  // full-resolution rendition, lazily fetched on first zoom (src is enough
  // for fitted display, but pixelates when zoomed in)
  hiresSrc?: string;
  // classes the clip container takes while zoomed (e.g. "fixed inset-x-0
  // top-16 bottom-24 z-0"); without it the zoom stays clipped to the fitted
  // image bounds
  expandClass?: string;
}
const props = defineProps<Props>();
const emit = defineEmits<{ error: [] }>();

const container = ref<HTMLElement | null>(null);
const imgEl = ref<HTMLImageElement | null>(null);
const zoom = ref(ZOOM_RESET);
const zoomed = computed(() => zoom.value.scale > 1);
const panning = ref(false);

const hires = ref<"idle" | "loading" | "ready" | "failed">("idle");
watch(zoomed, (z) => {
  if (z && hires.value === "idle" && props.hiresSrc) hires.value = "loading";
});

// Ctrl/Cmd held at the moment the src swaps decides: keep the zoom across the
// switch (held) or return to the fitted view (not held). Tracked on window so
// it covers hotkey and film-strip navigation alike.
let modifierHeld = false;
const trackModifier = (e: KeyboardEvent) => (modifierHeld = e.ctrlKey || e.metaKey);
const clearModifier = () => (modifierHeld = false);
onMounted(() => {
  window.addEventListener("keydown", trackModifier);
  window.addEventListener("keyup", trackModifier);
  window.addEventListener("blur", clearModifier);
});
onUnmounted(() => {
  window.removeEventListener("keydown", trackModifier);
  window.removeEventListener("keyup", trackModifier);
  window.removeEventListener("blur", clearModifier);
});

let pendingReset = false;
watch(
  () => props.src,
  () => {
    if (zoomed.value && modifierHeld) {
      // keep zoom and stage — the next image swaps in-place. Unfreeze so it
      // takes its own stage-fitted size (aspect may differ); re-clamped on load.
      freezeStyle.value = undefined;
      hires.value = props.hiresSrc ? "loading" : "idle";
      return;
    }
    if (zoomed.value) {
      // reset only once the next image is ready — never visibly un-zoom the
      // old one first (that was the jitter)
      pendingReset = true;
      hires.value = "idle";
      return;
    }
    zoom.value = ZOOM_RESET;
    hires.value = "idle";
  },
);

function onImgLoad() {
  if (pendingReset) {
    pendingReset = false;
    freezeStyle.value = undefined;
    zoom.value = ZOOM_RESET;
    return;
  }
  // a kept zoom: pull the offsets back into the new image's pan range once its
  // real fitted size is known
  const el = container.value;
  if (!el || !zoomed.value) return;
  const content = contentSize(el);
  zoom.value = panBy(zoom.value, 0, 0, el.clientWidth, el.clientHeight, content.w, content.h);
}

// Freeze the img at its fitted size while the stage is active — the stage's
// width would otherwise re-fit the image mid-zoom (its max-* constraints
// resolve against the wider container) and make it jump.
const freezeStyle = ref<{ width: string; height: string } | undefined>();
const stageActive = computed(() => zoomed.value && !!props.expandClass);
watch(stageActive, async (active) => {
  // pre-flush: rects and sizes here are still the old (flow) layout
  const before = container.value?.getBoundingClientRect();
  if (active && imgEl.value) {
    freezeStyle.value = { width: `${imgEl.value.offsetWidth}px`, height: `${imgEl.value.offsetHeight}px` };
  } else {
    freezeStyle.value = undefined;
  }
  await nextTick();
  // keep the image visually in place across the flow → stage re-anchor
  const after = container.value?.getBoundingClientRect();
  if (active && before && after) {
    zoom.value = { ...zoom.value, tx: zoom.value.tx + before.left - after.left, ty: zoom.value.ty + before.top - after.top };
  }
});

// content = the fitted image; equals the container in flow mode (w-fit), the
// distinction matters once the stage is larger than the image
function contentSize(el: HTMLElement): { w: number; h: number } {
  const img = imgEl.value;
  if (img) return { w: img.offsetWidth, h: img.offsetHeight };
  return { w: el.clientWidth, h: el.clientHeight };
}

function zoomTo(clientX: number, clientY: number, targetScale: number) {
  const el = container.value;
  if (!el) return;
  const rect = el.getBoundingClientRect();
  const content = contentSize(el);
  zoom.value = zoomAt(zoom.value, clientX - rect.left, clientY - rect.top, targetScale, rect.width, rect.height, content.w, content.h);
}

function onWheel(event: WheelEvent) {
  zoomTo(event.clientX, event.clientY, zoom.value.scale * Math.exp(-event.deltaY * 0.002));
}

function onDblClick(event: MouseEvent) {
  if (zoomed.value) zoom.value = ZOOM_RESET;
  else zoomTo(event.clientX, event.clientY, 2.5);
}

// drag-pan while zoomed; a pan must not fall through as a click on slotted
// overlays (face boxes), so the capture handler swallows the click after a drag
let panPointer: number | null = null;
let lastX = 0;
let lastY = 0;
let dragged = false;

function onPointerDown(event: PointerEvent) {
  if (!zoomed.value || event.button !== 0) return;
  panPointer = event.pointerId;
  lastX = event.clientX;
  lastY = event.clientY;
  dragged = false;
  panning.value = true;
  container.value?.setPointerCapture(event.pointerId);
}

function onPointerMove(event: PointerEvent) {
  if (panPointer !== event.pointerId) return;
  const dx = event.clientX - lastX;
  const dy = event.clientY - lastY;
  if (Math.abs(dx) + Math.abs(dy) > 2) dragged = true;
  lastX = event.clientX;
  lastY = event.clientY;
  const el = container.value;
  if (el) {
    const content = contentSize(el);
    zoom.value = panBy(zoom.value, dx, dy, el.clientWidth, el.clientHeight, content.w, content.h);
  }
}

function onPointerUp(event: PointerEvent) {
  if (panPointer !== event.pointerId) return;
  panPointer = null;
  panning.value = false;
}

function onClickCapture(event: MouseEvent) {
  if (!dragged) return;
  dragged = false;
  event.stopPropagation();
  event.preventDefault();
}
</script>
