<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<script setup lang="ts">
import { emitter } from "~/boot/mitt";
import { useRouter } from "vue-router";

onMounted(async () => {
  const store = useStore();
  await store.startIntervals();
});

const router = useRouter();
emitter.on("pushToLogin", () => {
  router.push("/login");
});
emitter.on("pushToHome", () => {
  router.push("/");
});
</script>
