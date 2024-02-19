<template>
  <router-view />
</template>

<script setup lang="ts">
import { useUserStore } from "./stores/user-store";
import { useRouter } from "vue-router";
import { emitter } from "./boot/mitt";
import { useLoginStore } from "./stores/login-store";
import { onMounted } from "@vue/runtime-core";

// TODO: add blocking page if user account is not active

onMounted(async () => {
  const userStore = useUserStore();
  const loginStore = useLoginStore();
  await loginStore.startTokenRefresh();
  if (loginStore.isLoggedIn) {
    userStore.refreshUser();
    userStore.startUserRefresh();
  }
});

const router = useRouter();
emitter.on("pushToLogin", () => {
  router.push("/login");
});
emitter.on("pushToHome", () => {
  router.push("/");
});
</script>
