<template>
  <Navbar />
  <Sidebar />
  <router-view />
</template>

<script setup lang="ts">
import { ref } from "vue";
import Navbar from "src/components/layout/Navbar.vue";
import Sidebar from "src/components/layout/Sidebar.vue";
</script>
