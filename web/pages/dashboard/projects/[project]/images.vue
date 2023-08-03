<template>
  <NuxtLayout name="logged-in">
    <ProjectImages :projectId="projectId"></ProjectImages>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const projectId = router.currentRoute.value.params.project;
</script>
