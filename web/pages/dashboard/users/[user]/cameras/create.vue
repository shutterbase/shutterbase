<template>
  <NuxtLayout name="logged-in">
    <CameraCreate :userId="userId"></CameraCreate>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
const router = useRouter();

const userId = router.currentRoute.value.params.user;
</script>

<style scoped></style>
