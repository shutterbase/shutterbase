<template>
  <div class="lg:flex lg:items-center lg:justify-between mb-16">
    <div v-if="filtered" class="flex-1">
      <div v-if="totalImageCount === 0" class="min-w-0 flex-1">
        <div class="mt-8">
          <img src="~assets/img/search-potato.webp" alt="No images" class="mx-auto h-96 w-96 rounded-xl object-center" />
        </div>
        <h2 class="mt-8 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight">No images match the search criteria</h2>
      </div>
      <div v-else-if="totalImageCount === currentImageCount" class="min-w-0 flex-1">
        <div class="mt-8">
          <img src="~assets/img/search-potato.webp" alt="No images" class="mx-auto h-96 w-96 rounded-xl object-center" />
        </div>
        <h2 class="mt-8 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight">
          All {{ totalImageCount }} images matching the search criteria are shown
        </h2>
      </div>
      <div v-else class="min-w-0 flex-1 text-center">
        <button @click="() => emit('loadMore')">
          <h2 class="mt-6 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight hover:underline">Load more</h2>
        </button>
      </div>
    </div>
    <div v-else class="flex-1">
      <div v-if="totalImageCount === 0" class="min-w-0 flex-1">
        <div class="mt-8">
          <img src="~assets/img/ghost.webp" alt="No images" class="mx-auto h-96 w-96 rounded-xl object-center" />
        </div>
        <h2 class="mt-8 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight">
          There are no images in project {{ activeProject.name }} yet
        </h2>
      </div>
      <div v-else-if="totalImageCount === currentImageCount" class="min-w-0 flex-1">
        <div class="mt-8">
          <img src="~assets/img/potato.webp" alt="No more images" class="mx-auto h-96 w-96 rounded-xl object-center" />
        </div>
        <h2 class="mt-6 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight">
          All {{ totalImageCount }} images in project {{ activeProject.name }} are shown
        </h2>
      </div>
      <div v-else class="min-w-0 flex-1 text-center">
        <button @click="() => emit('loadMore')">
          <h2 class="mt-6 text-center text-xl font-bold leading-7 text-gray-900 dark:text-gray-100 sm:truncate sm:text-3xl sm:tracking-tight hover:underline">Load more</h2>
        </button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";

interface Props {
  totalImageCount: number;
  currentImageCount: number;
  filtered: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  totalImageCount: 0,
  currentImageCount: 0,
  filtered: false,
});

const { activeProject } = storeToRefs(useUserStore());

const emit = defineEmits<{
  loadMore: [];
}>();
</script>
