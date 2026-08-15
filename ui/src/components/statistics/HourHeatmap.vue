<template>
  <div v-if="max === 0" class="flex h-24 items-center justify-center text-sm text-primary-500 dark:text-primary-400">No data yet</div>
  <div v-else>
    <div class="flex items-center gap-[2px]">
      <div class="w-16 shrink-0"></div>
      <div v-for="hour in 24" :key="hour" class="label-mono-sm min-w-0 flex-1 text-center text-primary-400 dark:text-primary-500">
        <span v-if="(hour - 1) % 6 === 0">{{ String(hour - 1).padStart(2, "0") }}</span>
      </div>
    </div>
    <div v-for="day in days" :key="day.date" class="mt-[2px] flex items-center gap-[2px]">
      <div class="label-mono-sm w-16 shrink-0 text-primary-500 dark:text-primary-400">{{ shortDayLabel(day.date) }}</div>
      <div
        v-for="(count, hour) in day.byHour"
        :key="hour"
        class="h-4 min-w-0 flex-1 rounded-[2px]"
        :class="count === 0 ? 'bg-primary-100/70 dark:bg-primary-800/50' : ''"
        :style="count > 0 ? { backgroundColor: heatColor(count, max) } : {}"
        :title="`${shortDayLabel(day.date)} ${String(hour).padStart(2, '0')}:00 — ${count} ${count === 1 ? 'image' : 'images'}`"
      ></div>
    </div>
    <div class="mt-2 flex items-center justify-end gap-1.5">
      <span class="label-mono-sm text-primary-400 dark:text-primary-500">1</span>
      <span v-for="step in 5" :key="step" class="h-2.5 w-5 rounded-[2px]" :style="{ backgroundColor: `var(--chart-heat-${step})` }"></span>
      <span class="label-mono-sm text-primary-400 dark:text-primary-500">{{ max }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { StatDay } from "src/api/statistics";
import { heatColor } from "src/util/chartColors";
import { shortDayLabel } from "src/util/dateTimeUtil";

const props = defineProps<{ days: StatDay[] }>();

const max = computed(() => Math.max(0, ...props.days.flatMap((d) => d.byHour)));
</script>
