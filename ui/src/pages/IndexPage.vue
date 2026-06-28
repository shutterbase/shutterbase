<template>
  <main class="mx-auto w-full max-w-7xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
    <p class="label-mono text-accent-600 dark:text-accent-400">Welcome back</p>
    <h1 class="display mt-3 text-3xl text-primary-900 dark:text-white sm:text-5xl">Your shared photo library.</h1>
    <p class="mt-5 max-w-xl text-base leading-7 text-primary-600 dark:text-primary-300">Jump back into a project, browse the latest frames, or set up a camera to start syncing your timeline with the team.</p>

    <div class="mt-10 grid gap-4 sm:grid-cols-3">
      <router-link
        :to="{ name: 'projects' }"
        class="group flex cursor-pointer flex-col gap-3 rounded-lg border border-primary-200 bg-surface p-6 shadow-panel transition-colors hover:border-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark dark:hover:border-accent-500"
      >
        <FolderIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">Projects</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Open a shared project and its library.</span>
      </router-link>

      <router-link
        :to="{ name: 'images' }"
        class="group flex cursor-pointer flex-col gap-3 rounded-lg border border-primary-200 bg-surface p-6 shadow-panel transition-colors hover:border-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark dark:hover:border-accent-500"
      >
        <PhotoIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">Images</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Browse, search and tag the latest frames.</span>
      </router-link>

      <router-link
        :to="`/users/${userStore.user?.id || ''}/cameras`"
        class="group flex cursor-pointer flex-col gap-3 rounded-lg border border-primary-200 bg-surface p-6 shadow-panel transition-colors hover:border-accent-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark dark:hover:border-accent-500"
      >
        <CameraIcon class="h-6 w-6 text-accent-600 dark:text-accent-400" />
        <span class="display text-lg text-primary-900 dark:text-white">My Cameras</span>
        <span class="text-sm text-primary-500 dark:text-primary-400">Add a camera and set its time offset.</span>
      </router-link>
    </div>
  </main>
</template>

<script setup lang="ts">
import { useUserStore } from "src/stores/user-store";
import { CameraIcon, FolderIcon, PhotoIcon } from "@heroicons/vue/24/solid";

const userStore = useUserStore();
</script>
