<template>
  <main class="flex min-h-screen flex-col items-center justify-center bg-surface px-6 py-24 text-center dark:bg-primary-950">
    <div class="relative">
      <CornerMarks />
      <img src="~assets/img/ghost.webp" alt="Page not found" class="h-52 w-52 rounded-xl object-cover sm:h-60 sm:w-60" />
    </div>
    <p class="label-mono mt-8 text-accent-600 dark:text-accent-400">Error 404</p>
    <h1 class="display mt-2.5 text-3xl text-primary-900 dark:text-white sm:text-4xl">Nothing here...</h1>
    <p class="mt-2 max-w-md text-sm text-primary-500 dark:text-primary-400">The page you were looking for doesn't exist or has moved.</p>
    <router-link
      to="/"
      class="mt-8 inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950"
    >
      Go home
    </router-link>
  </main>
</template>

<script setup lang="ts">
import CornerMarks from "src/components/CornerMarks.vue";
</script>
