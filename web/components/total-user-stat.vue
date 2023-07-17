<template>
  <div class="stats bg-primary text-primary-content">
    <div class="stat">
      <div class="stat-title text-white">Registered Users</div>
      <div class="stat-value">{{ total }}</div>
      <div class="stat-actions">
        <NuxtLink to="/dashboard/users">
          <button class="btn btn-sm btn-success">Manage users</button>
        </NuxtLink>
      </div>
      <div class="stat-figure text-secondary">
        <Icon name="mdi:account-multiple" size="64" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { requestList } from "~/api/common";
const { total } = await requestList("/users", { limit: 1 });
</script>
