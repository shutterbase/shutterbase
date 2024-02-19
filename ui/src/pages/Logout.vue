<template></template>
<script setup lang="ts">
import { onMounted } from "vue";
import pb from "src/boot/pocketbase";
import { useRouter } from "vue-router";

const router = useRouter();

onMounted(() => {
  pb.authStore.clear();
  router.push("login");
});
</script>
