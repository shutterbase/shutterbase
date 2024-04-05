<template>
  <main class="grid min-h-full place-items-center px-6 py-24 sm:py-32 lg:px-8">
    <div class="text-center">
      <p class="text-base font-semibold text-primary-600 dark:text-primary-300">Hi</p>
      <h1 class="mt-4 text-3xl font-bold tracking-tight text-primary-700 dark:text-primary-200 sm:text-5xl">Welcome to Shutterbase</h1>
      <p class="mt-6 text-base leading-7 text-primary-600 dark:text-primary-300">Start by adding a camera</p>
      <div class="mt-10 flex items-center justify-center gap-x-6">
        <router-link
          :to="`/users/${pb.authStore.model?.id || ''}/cameras`"
          class="bg-secondary-600 hover:bg-secondary-700 inline-flex w-full justify-center rounded-md px-3 py-2 text-sm font-semibold text-white shadow-sm sm:ml-3 sm:w-auto"
          ><CameraIcon class="mr-2 w-5 h-5 text-white" /> My Cameras</router-link
        >
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import pb from "src/boot/pocketbase";
import { CameraIcon } from "@heroicons/vue/24/solid";
</script>
