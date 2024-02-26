<template>
  <div class="mx-auto max-w-7xl lg:flex lg:gap-x-16 lg:px-8">
    <Navigation />
    <router-view />
  </div>
</template>

<script setup lang="ts">
import Navigation from "src/components/project/Navigation.vue";
</script>
