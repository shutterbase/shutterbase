<template></template>
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
