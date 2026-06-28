<template>
  <div class="w-full">
    <Navigation />
    <div class="pt-8 pb-12">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import Navigation from "src/components/project/Navigation.vue";
</script>
