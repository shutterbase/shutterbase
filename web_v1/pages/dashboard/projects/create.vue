<template>
  <NuxtLayout name="logged-in">
    <ProjectCreate></ProjectCreate>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
const router = useRouter();
</script>

<style scoped></style>
