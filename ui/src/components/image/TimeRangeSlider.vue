<template>
  <div class="select-none" :class="disabled ? 'pointer-events-none opacity-40' : ''">
    <div class="relative h-6">
      <!-- track -->
      <div class="absolute top-1/2 h-1 w-full -translate-y-1/2 rounded-full bg-primary-200 dark:bg-primary-700"></div>
      <!-- selected segment -->
      <div
        class="absolute top-1/2 h-1 -translate-y-1/2 rounded-full bg-accent-500"
        :style="{ left: lowPct + '%', width: highPct - lowPct + '%' }"
      ></div>
      <input
        aria-label="Range start"
        type="range"
        class="time-thumb absolute top-0 h-6 w-full appearance-none bg-transparent"
        :min="minStep"
        :max="maxStep"
        step="1"
        v-model.number="lowStep"
        @change="commit"
      />
      <input
        aria-label="Range end"
        type="range"
        class="time-thumb absolute top-0 h-6 w-full appearance-none bg-transparent"
        :min="minStep"
        :max="maxStep"
        step="1"
        v-model.number="highStep"
        @change="commit"
      />
    </div>
    <div class="label-mono-sm mt-1 flex justify-between text-primary-400 dark:text-primary-500">
      <span>{{ fmt(domainMin) }}</span>
      <span>{{ fmt(domainMax) }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
// Two overlapped native range inputs = a dual-thumb time-range slider.
// Keyboard-accessible, no component framework needed. Thumbs work in whole
// minutes across the domain; dragging updates locally, release (`change`)
// commits ISO strings — one request per gesture, never per pixel.
import { computed, ref, watch } from "vue";

const props = defineProps<{
  min: string;
  max: string;
  from?: string | null;
  to?: string | null;
  disabled?: boolean;
}>();

const emit = defineEmits<{ change: [string | null, string | null] }>();

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

// inbound sync (props are the source of truth); keep thumbs ordered
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

// keep low<=high while dragging by nudging the opposite thumb
watch(lowStep, (v) => {
  if (v > highStep.value) highStep.value = v;
});
watch(highStep, (v) => {
  if (v < lowStep.value) lowStep.value = v;
});

const commit = () => {
  emit("change", new Date(lowStep.value * MINUTE).toISOString(), new Date(highStep.value * MINUTE).toISOString());
};

const lowPct = computed(() => ((lowStep.value - minStep.value) / Math.max(maxStep.value - minStep.value, 1)) * 100);
const highPct = computed(() => ((highStep.value - minStep.value) / Math.max(maxStep.value - minStep.value, 1)) * 100);

const fmt = (ms: number) =>
  new Date(ms).toLocaleString([], { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
</script>

<style scoped>
/* both inputs cover the track; only their thumbs receive pointer events */
.time-thumb {
  pointer-events: none;
  -webkit-appearance: none;
  appearance: none;
}
.time-thumb::-webkit-slider-thumb {
  pointer-events: auto;
  -webkit-appearance: none;
  appearance: none;
  height: 14px;
  width: 14px;
  border-radius: 9999px;
  background: var(--color-accent-600, #2563eb);
  border: 2px solid white;
  cursor: pointer;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.3);
}
.time-thumb::-moz-range-thumb {
  pointer-events: auto;
  height: 14px;
  width: 14px;
  border-radius: 9999px;
  background: #2563eb;
  border: 2px solid white;
  cursor: pointer;
}
.time-thumb::-webkit-slider-runnable-track,
.time-thumb::-moz-range-track {
  background: transparent;
}
</style>
