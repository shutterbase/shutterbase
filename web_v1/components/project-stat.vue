<template>
  <div class="stats bg-primary text-primary-content">
    <div class="stat">
      <div class="stat-title text-white">Projects</div>
      <div class="stat-value">{{ total }}</div>
      <div class="stat-actions">
        <NuxtLink to="/dashboard/projects">
          <button class="btn btn-sm btn-success">Manage Projects</button>
        </NuxtLink>
      </div>
      <div class="stat-figure text-secondary">
        <Icon name="mdi:view-gallery" size="64" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { requestList } from "~/api/common";
const { total } = await requestList("/projects", { limit: 1 });
</script>
