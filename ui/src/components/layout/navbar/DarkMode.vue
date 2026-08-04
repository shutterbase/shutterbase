<template>
  <button
    type="button"
    @click="toggle"
    :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
    class="inline-flex items-center justify-center rounded-md p-2 text-primary-500 dark:text-primary-300 hover:bg-primary-100 dark:hover:bg-primary-800 hover:text-primary-900 dark:hover:text-white transition-colors"
  >
    <span class="sr-only">Toggle dark mode</span>
    <SunIcon v-if="isDark" class="h-5 w-5" aria-hidden="true" />
    <MoonIcon v-else class="h-5 w-5" aria-hidden="true" />
  </button>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { SunIcon, MoonIcon } from "@heroicons/vue/24/outline";

const isDark = ref(true);

onMounted(() => {
  isDark.value = document.documentElement.classList.contains("dark");
});

function toggle() {
  const root = document.documentElement;
  const next = !root.classList.contains("dark");
  root.classList.toggle("dark", next);
  localStorage.setItem("color-theme", next ? "dark" : "light");
  isDark.value = next;
}
</script>
