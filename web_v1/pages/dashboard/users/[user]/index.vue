<template>
  <NuxtLayout name="logged-in">
    <UserDetail :id="userId"></UserDetail>
  </NuxtLayout>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";

const router = useRouter();

const userId = `${router.currentRoute.value.params.user}`;
console.log(userId);
</script>

<style scoped></style>
