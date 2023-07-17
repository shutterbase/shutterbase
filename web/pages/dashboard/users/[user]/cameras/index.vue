<template>
  <NuxtLayout name="logged-in">
    <CameraList :userId="userId"></CameraList>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
const router = useRouter();

const userId = router.currentRoute.value.params.user;
</script>

<style scoped></style>
