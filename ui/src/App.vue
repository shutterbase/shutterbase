<template>
  <router-view />
</template>

<script setup lang="ts">
import { keyEventHandler } from "src/util/keyEvents";
import { onMounted, onUnmounted } from "vue";

onMounted(() => {
  document.addEventListener("keydown", keyEventHandler);
});
onUnmounted(() => {
  document.removeEventListener("keydown", keyEventHandler);
});
</script>
