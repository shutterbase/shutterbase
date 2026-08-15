<template>
  <dl class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
    <div
      v-for="tile in tiles"
      :key="tile.label"
      class="rounded-lg border border-primary-200 bg-surface p-4 shadow-panel dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark"
    >
      <dt class="label-mono text-[0.6rem] text-primary-500 dark:text-primary-400">{{ tile.label }}</dt>
      <dd class="font-data mt-1 text-xl text-primary-900 dark:text-white">{{ tile.value }}</dd>
      <p v-if="tile.sub" class="mt-0.5 truncate text-xs text-primary-500 dark:text-primary-400">{{ tile.sub }}</p>
    </div>
  </dl>
</template>

<script setup lang="ts">
export interface StatTile {
  label: string;
  value: string;
  sub?: string;
}

defineProps<{ tiles: StatTile[] }>();
</script>
