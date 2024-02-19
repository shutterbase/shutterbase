<script setup lang="ts">
import { getCreatedByString, getUpdatedByString, getDateTimeString } from "~/api/common";
const props = defineProps({
  item: {
    type: Object,
    required: true,
  },
});

const id = computed(() => props.item?.id);
</script>

<template>
  <div class="text-xs font-mono text-gray-500" v-if="item">
    ID: {{ id }} | Created by <span class="text-bold">{{ getCreatedByString(item) }}</span> on {{ getDateTimeString(item.createdAt) }} | Updated by
    <span class="text-bold">{{ getUpdatedByString(item) }}</span> on
    {{ getDateTimeString(item.updatedAt) }}
  </div>
  <div v-if="item.capturedAtCorrected && item.capturedAt" class="text-xs font-mono text-gray-500">
    Captured on {{ getDateTimeString(item.capturedAt) }} | Corrected capture time: {{ getDateTimeString(item.capturedAtCorrected) }} | Computed file name:
    {{ item.computedFileName }}
  </div>
</template>
