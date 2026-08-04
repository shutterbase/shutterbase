<template>
  <main class="flex min-h-screen items-center justify-center bg-surface dark:bg-primary-950">
    <p class="label-mono text-primary-500 dark:text-primary-400">Signing out…</p>
  </main>
</template>
<script setup lang="ts">
import { onMounted } from "vue";
import { useUserStore } from "src/stores/user-store";
import { useRouter } from "vue-router";

const router = useRouter();
const userStore = useUserStore();

onMounted(async () => {
  await userStore.logout();
  router.push({ name: "login" });
});
</script>
