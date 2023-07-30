<template>
  <NuxtLayout name="logged-in">
    <ProjectTags :projectId="projectId"></ProjectTags>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const projectId = router.currentRoute.value.params.project;
</script>

<style scoped></style>
