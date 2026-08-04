<template>
  <div class="mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8">
    <Navigation />
    <router-view />
  </div>
</template>

<script setup lang="ts">
import Navigation from "src/components/user/Navigation.vue";
</script>
