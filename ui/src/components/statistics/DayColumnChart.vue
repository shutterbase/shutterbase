<template>
  <div v-if="maxTotal === 0" class="flex h-40 items-center justify-center text-sm text-primary-500 dark:text-primary-400">No data yet</div>
  <div v-else>
    <div class="flex gap-2">
      <!-- y axis: max / mid / 0, recessive -->
      <div class="font-data flex h-40 w-9 shrink-0 flex-col items-end justify-between text-[10px] text-primary-400 dark:text-primary-500">
        <span>{{ maxTotal }}</span>
        <span>{{ Math.round(maxTotal / 2) }}</span>
        <span>0</span>
      </div>
      <div class="relative h-40 flex-1">
        <!-- recessive gridlines at 0 / 50 / 100% -->
        <div class="pointer-events-none absolute inset-x-0 top-0 border-t border-primary-100 dark:border-primary-800"></div>
        <div class="pointer-events-none absolute inset-x-0 top-1/2 border-t border-primary-100 dark:border-primary-800"></div>
        <div class="pointer-events-none absolute inset-x-0 bottom-0 border-t border-primary-200 dark:border-primary-700"></div>
        <div class="absolute inset-0 flex items-end gap-[2px]">
          <!-- the whole column is the hover target, larger than the marks -->
          <div v-for="day in days" :key="day.date" class="group relative h-full min-w-0 flex-1 rounded-sm hover:bg-primary-100/60 dark:hover:bg-primary-800/40">
            <!-- full-height stack so segment %-heights resolve and align with the
                 gridlines; the 2px surface gap between segments lives inside each
                 segment's height as a transparent border (border-box + clip-padding) -->
            <div class="absolute inset-0 flex flex-col-reverse px-px">
              <div
                v-for="segment in day.segments"
                :key="segment.key"
                class="w-full border-b-2 border-transparent bg-clip-padding first:border-b-0 last:rounded-t-[4px]"
                :style="{ backgroundColor: segment.color, height: `${(segment.value / maxTotal) * 100}%` }"
              ></div>
            </div>
            <!-- tooltip -->
            <div
              v-if="day.total > 0"
              class="pointer-events-none absolute bottom-full left-1/2 z-10 hidden -translate-x-1/2 whitespace-nowrap rounded border border-primary-200 bg-surface p-2 text-xs shadow-panel group-hover:block dark:border-primary-700 dark:bg-surface-dark dark:shadow-panel-dark"
            >
              <p class="label-mono-sm text-primary-500 dark:text-primary-400">{{ day.label }}</p>
              <div v-for="segment in day.segments" :key="segment.key" class="mt-1 flex items-center gap-1.5">
                <span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: segment.color }"></span>
                <span class="text-primary-700 dark:text-primary-200">{{ segment.label }}</span>
                <span class="font-data ml-auto pl-3 text-primary-900 dark:text-white">{{ segment.value }}</span>
              </div>
              <div v-if="day.segments.length > 1" class="mt-1 border-t border-primary-100 pt-1 text-right dark:border-primary-800">
                <span class="font-data text-primary-900 dark:text-white">{{ day.total }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- x labels, thinned to ~10 -->
    <div class="ml-11 flex gap-[2px]">
      <div v-for="(day, i) in days" :key="day.date" class="label-mono-sm min-w-0 flex-1 truncate pt-1 text-center text-primary-500 dark:text-primary-400">
        <span v-if="i % labelStep === 0">{{ day.label }}</span>
      </div>
    </div>
    <!-- legend: only for real multi-series charts -->
    <div v-if="legend && legend.length > 1" class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1">
      <div v-for="entry in legend" :key="entry.key" class="flex items-center gap-1.5">
        <span class="h-2 w-2 rounded-full" :style="{ backgroundColor: entry.color }"></span>
        <span class="text-xs text-primary-600 dark:text-primary-300">{{ entry.label }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ChartDay, SeriesEntry } from "src/util/statisticsView";

const props = defineProps<{
  days: ChartDay[];
  legend?: SeriesEntry[];
}>();

const maxTotal = computed(() => Math.max(0, ...props.days.map((d) => d.total)));
const labelStep = computed(() => Math.max(1, Math.ceil(props.days.length / 10)));
</script>
