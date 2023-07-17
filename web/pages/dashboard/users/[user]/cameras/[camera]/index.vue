<template>
  <NuxtLayout name="logged-in">
    <CameraDetail :id="cameraId" :userId="userId"></CameraDetail>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const userId = router.currentRoute.value.params.user;
const cameraId = router.currentRoute.value.params.camera;
</script>

<style scoped></style>
