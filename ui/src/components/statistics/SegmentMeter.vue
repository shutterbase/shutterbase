<template>
  <div>
    <!-- flex-nowrap is load-bearing: Quasar's .flex sets flex-wrap:wrap, which
         would push the last segment onto a clipped second row -->
    <div class="flex h-3 flex-nowrap gap-[2px] overflow-hidden rounded-full" :class="total === 0 ? 'bg-primary-100 dark:bg-primary-800' : ''">
      <div
        v-for="segment in visibleSegments"
        :key="segment.key"
        class="h-full first:rounded-l-full last:rounded-r-full"
        :style="{ width: `${(segment.value / total) * 100}%`, backgroundColor: segment.color }"
      ></div>
    </div>
    <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1">
      <div v-for="segment in segments" :key="segment.key" class="flex items-center gap-1.5">
        <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: segment.color }"></span>
        <span class="text-xs text-primary-600 dark:text-primary-300">{{ segment.label }}</span>
        <span class="font-data text-xs text-primary-900 dark:text-white">{{ segment.value }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

export interface MeterSegment {
  key: string;
  label: string;
  value: number;
  color: string;
}

const props = defineProps<{ segments: MeterSegment[] }>();

const total = computed(() => props.segments.reduce((sum, s) => sum + s.value, 0));
const visibleSegments = computed(() => props.segments.filter((s) => s.value > 0));
</script>
