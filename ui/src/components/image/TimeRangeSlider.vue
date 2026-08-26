<template>
  <div class="select-none" :class="disabled ? 'pointer-events-none opacity-40' : ''">
    <div
      class="relative h-6 cursor-pointer"
      ref="trackRef"
      @pointerdown="onPointerDown"
    >
      <!-- track -->
      <div class="absolute top-1/2 h-1 w-full -translate-y-1/2 rounded-full bg-primary-200 dark:bg-primary-700"></div>
      <!-- selected segment -->
      <div
        class="absolute top-1/2 h-1 -translate-y-1/2 rounded-full bg-accent-500"
        :style="{ left: lowPct + '%', width: highPct - lowPct + '%' }"
      ></div>
      <!-- low thumb -->
      <div
        class="absolute top-1/2 z-10 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent-600 shadow-md transition-none"
        :class="dragging === 'low' ? 'ring-2 ring-accent-400/40' : ''"
        :style="{ left: lowPct + '%' }"
      ></div>
      <!-- high thumb -->
      <div
        class="absolute top-1/2 z-20 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-accent-600 shadow-md transition-none"
        :class="dragging === 'high' ? 'ring-2 ring-accent-400/40' : ''"
        :style="{ left: highPct + '%' }"
      ></div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch, onUnmounted } from "vue";

const props = defineProps<{
  min: string;
  max: string;
  from?: string | null;
  to?: string | null;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  change: [string | null, string | null];
  preview: [string, string];
}>();

const MINUTE = 60_000;
const domainMin = computed(() => new Date(props.min).getTime());
const domainMax = computed(() => new Date(props.max).getTime());
const minStep = computed(() => Math.round(domainMin.value / MINUTE));
const maxStep = computed(() => Math.round(domainMax.value / MINUTE));

const clamp = (ms: number) => Math.min(Math.max(ms, domainMin.value), domainMax.value);
const toStep = (iso: string | null | undefined, fallback: number) => {
  const ms = iso ? new Date(iso).getTime() : NaN;
  return Number.isNaN(ms) ? fallback : Math.round(clamp(ms) / MINUTE);
};

const lowStep = ref(toStep(props.from, minStep.value));
const highStep = ref(toStep(props.to, maxStep.value));

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
  },
);

watch(lowStep, (v) => {
  if (v > highStep.value) highStep.value = v;
});
watch(highStep, (v) => {
  if (v < lowStep.value) lowStep.value = v;
});

const lowPct = computed(() => ((lowStep.value - minStep.value) / Math.max(maxStep.value - minStep.value, 1)) * 100);
const highPct = computed(() => ((highStep.value - minStep.value) / Math.max(maxStep.value - minStep.value, 1)) * 100);

// --- manual pointer drag ---
const trackRef = ref<HTMLElement | null>(null);
const dragging = ref<"low" | "high" | null>(null);

function stepFromClientX(clientX: number): number {
  const el = trackRef.value;
  if (!el) return minStep.value;
  const rect = el.getBoundingClientRect();
  const pct = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
  return Math.round(minStep.value + pct * (maxStep.value - minStep.value));
}

function onPointerDown(e: PointerEvent) {
  if (props.disabled) return;
  const clickStep = stepFromClientX(e.clientX);
  // pick the closest thumb
  const distLow = Math.abs(clickStep - lowStep.value);
  const distHigh = Math.abs(clickStep - highStep.value);
  const target = distLow <= distHigh ? "low" : "high";
  dragging.value = target;

  if (target === "low") lowStep.value = clickStep;
  else highStep.value = clickStep;

  emit("preview", new Date(lowStep.value * MINUTE).toISOString(), new Date(highStep.value * MINUTE).toISOString());

  const onMove = (ev: PointerEvent) => {
    const step = stepFromClientX(ev.clientX);
    if (target === "low") lowStep.value = step;
    else highStep.value = step;
    emit("preview", new Date(lowStep.value * MINUTE).toISOString(), new Date(highStep.value * MINUTE).toISOString());
  };

  const onUp = () => {
    dragging.value = null;
    window.removeEventListener("pointermove", onMove);
    window.removeEventListener("pointerup", onUp);
    emit("change", new Date(lowStep.value * MINUTE).toISOString(), new Date(highStep.value * MINUTE).toISOString());
  };

  window.addEventListener("pointermove", onMove);
  window.addEventListener("pointerup", onUp, { once: true });
}
</script>
