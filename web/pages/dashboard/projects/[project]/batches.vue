<template>
  <NuxtLayout name="logged-in">
    <ProjectBatches :projectId="projectId"></ProjectBatches>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const projectId = `${router.currentRoute.value.params.project}`;
</script>

<style scoped></style>
