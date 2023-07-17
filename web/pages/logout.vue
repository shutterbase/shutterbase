<template>
  <div class="hero min-h-screen bg-base-200">
    <div class="hero-content text-center">
      <div class="max-w-md">
        <h1 class="text-5xl font-bold">You've been logged out</h1>
        <p class="py-6"><Icon name="👋" size="96" /></p>
        <progress class="progress progress-success w-56" :value="progress" max="100"></progress>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { logout } from "~/api/authorization";

const progress = ref(0);
await logout();

const i = setInterval(() => {
  progress.value += 1;
  if (progress.value >= 100) {
    clearInterval(i);
    navigateTo("/login");
  }
}, 20);
</script>

<style scoped></style>
