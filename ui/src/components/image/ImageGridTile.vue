<template>
  <div class="group relative border-b border-t border-r border-gray-200 dark:border-gray-600 p-4 sm:p-6">
    <div class="aspect-h-1 aspect-w-1 overflow-hidden rounded-lg group-hover:opacity-75">
      <img :src="image.downloadUrls[`256`]" :alt="image.computedFileName" class="max-h-44 w-full object-cover object-center" />
    </div>
    <div class="pt-6 text-center">
      <h3 class="text-sm font-medium text-gray-900 dark:text-gray-100">
        <a href="#">
          <span aria-hidden="true" class="absolute inset-0"></span>
          {{ image.computedFileName }}
        </a>
      </h3>
      <div class="mt-3 flex flex-col items-center">
        <div class="flex items-center"></div>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{{ dateTimeUtil.dateTimeFromBackend(image.capturedAtCorrected) }}</p>
      </div>
      <!-- <p class="mt-4 text-base font-medium text-gray-900 dark:text-gray-100">$149</p> -->
    </div>
  </div>
</template>

<script setup lang="ts">
import { ImagesResponse } from "src/types/pocketbase";
import * as dateTimeUtil from "src/util/dateTimeUtil";

interface Props {
  image: ImagesResponse;
}
const props = withDefaults(defineProps<Props>(), {});
</script>
