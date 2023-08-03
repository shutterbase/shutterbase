<template>
  <NuxtLayout name="logged-in">
    <ProjectImageDetail :projectId="projectId"></ProjectImageDetail>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const projectId = router.currentRoute.value.params.project;
</script>
