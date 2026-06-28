<template>
  <router-view />
  <!-- DEV quick-actions panel: dev builds only. The async import sits in a dead
       branch in production (import.meta.env.DEV === false), so Rollup drops the
       chunk entirely — the panel code never ships to prod. -->
  <component :is="DevPanel" v-if="DevPanel" />
</template>

<script setup lang="ts">
import { keyEventHandler } from "src/util/keyEvents";
import { defineAsyncComponent, onMounted, onUnmounted } from "vue";
import { useUserStore } from "src/stores/user-store";

const DevPanel = import.meta.env.DEV ? defineAsyncComponent(() => import("src/components/DevPanel.vue")) : null;
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
