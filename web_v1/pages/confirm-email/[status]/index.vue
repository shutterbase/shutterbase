<template>
  <div class="hero min-h-screen bg-base-200">
    <div class="hero-content text-center">
      <div class="max-w-md" v-if="status === 'ok'">
        <h1 class="text-5xl font-bold">Your email has been verified</h1>
        <p class="py-6"><Icon name="🚀" size="96" /></p>
        <progress class="progress progress-success w-56" :value="progress" max="100"></progress>
      </div>
      <div v-else-if="status === 'email_already_validated'">
        <h1 class="text-5xl font-bold">Your email has already been validated</h1>
        <p class="py-6"><Icon name="👍" size="96" /></p>
        <progress class="progress progress-success w-56" :value="progress" max="100"></progress>
      </div>
      <div v-else>
        <h1 class="text-5xl font-bold">There has been an error verifying your email address</h1>
        <p class="py-6"><Icon name="☠️" size="96" /></p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const status = computed(() => {
  return router.currentRoute.value.params.status;
});

const progress = ref(0);

if (status.value === "ok" || status.value === "email_already_validated") {
  const i = setInterval(() => {
    progress.value += 1;
    if (progress.value >= 100) {
      clearInterval(i);
      navigateTo("/login");
    }
  }, 20);
}
</script>

<style scoped></style>
