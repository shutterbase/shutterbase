<template>
  <div class="stats bg-primary text-primary-content">
    <div v-if="activeProjectId" class="stat">
      <div class="stat-title text-white">Upload</div>
      <div class="stat-value">Batch</div>
      <div class="stat-actions">
        <NuxtLink :to="`/dashboard/projects/${activeProjectId}/new-batch`">
          <button class="btn btn-sm btn-success">Upload new batch</button>
        </NuxtLink>
      </div>
      <div class="stat-figure text-secondary">
        <Icon name="mdi:cloud-upload" size="64" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
const store = useStore();
const { activeProjectId } = storeToRefs(store);
</script>
