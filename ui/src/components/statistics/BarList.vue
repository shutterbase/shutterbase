<template>
  <div v-if="items.length === 0" class="flex h-16 items-center justify-center text-sm text-primary-500 dark:text-primary-400">No data yet</div>
  <div v-else class="space-y-2">
    <div v-for="item in items" :key="item.key" class="grid grid-cols-[minmax(0,11rem)_1fr_auto] items-center gap-x-3">
      <div class="min-w-0">
        <span class="block truncate text-sm text-primary-800 dark:text-primary-100" :title="item.label">{{ item.label }}</span>
        <span v-if="item.sub" class="label-mono-sm block truncate text-primary-400 dark:text-primary-500">{{ item.sub }}</span>
      </div>
      <div class="h-2.5 rounded-full bg-primary-100 dark:bg-primary-800">
        <div class="h-full rounded-full" :style="{ width: `${max > 0 ? (item.value / max) * 100 : 0}%`, backgroundColor: item.color ?? 'var(--chart-cat-1)' }"></div>
      </div>
      <span class="font-data text-xs text-primary-900 dark:text-white">{{ item.value }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

export interface BarListItem {
  key: string;
  label: string;
  sub?: string;
  value: number;
  color?: string;
}

const props = defineProps<{ items: BarListItem[] }>();

const max = computed(() => Math.max(0, ...props.items.map((i) => i.value)));
</script>
