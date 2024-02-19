<template>
  <NuxtLayout name="logged-in">
    <BatchUpload :projectId="projectId"></BatchUpload>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const projectId = `${router.currentRoute.value.params.project}`;
</script>
