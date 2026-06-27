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
  // load the effective user first so the active project is known, then tags
  if (!userStore.isAuthenticated) {
    await userStore.loadUser();
  }
  userStore.startUserFetching();
  await userStore.loadProjectTags();
  userStore.startProjectTagFetching();
});
onUnmounted(() => {
  document.removeEventListener("keydown", keyEventHandler);
  userStore.stopProjectTagFetching();
  userStore.stopUserFetching();
});
</script>
