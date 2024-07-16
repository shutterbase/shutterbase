<template>
  <router-view />
</template>

<script setup lang="ts">
import { keyEventHandler } from "src/util/keyEvents";
import { onMounted, onUnmounted } from "vue";
import { useUserStore } from "src/stores/user-store";

const userStore = useUserStore();

onMounted(async () => {
  document.addEventListener("keydown", keyEventHandler);
  await userStore.loadProjectTags();
  userStore.startProjectTagFetching();
});
onUnmounted(() => {
  document.removeEventListener("keydown", keyEventHandler);
  userStore.stopProjectTagFetching();
});
</script>
