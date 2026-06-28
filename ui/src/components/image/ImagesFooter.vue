<template>
  <div class="mb-16 mt-2">
    <!-- load more: only when there are more to fetch -->
    <div v-if="hasMore" class="flex justify-center pt-10">
      <button
        type="button"
        @click="() => emit('loadMore')"
        class="inline-flex items-center gap-2 rounded-md border border-primary-200 bg-surface px-5 py-2.5 text-sm font-medium text-primary-700 shadow-panel transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:shadow-panel-dark dark:hover:border-primary-600 dark:hover:text-white"
      >
        <ArrowDownIcon class="h-4 w-4" />
        Load more
      </button>
    </div>

    <!-- end / empty states -->
    <div v-else class="flex flex-col items-center px-6 pt-12 text-center">
      <img :src="art.src" :alt="art.headline" class="h-56 w-56 rounded-2xl object-cover sm:h-64 sm:w-64" />
      <h2 class="mt-7 text-lg font-semibold tracking-tight text-primary-900 dark:text-white">{{ art.headline }}</h2>
      <p v-if="art.sub" class="mt-1.5 max-w-md text-sm text-primary-500 dark:text-primary-400">{{ art.sub }}</p>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { ArrowDownIcon } from "@heroicons/vue/24/outline";

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

const hasMore = computed(() => props.totalImageCount > 0 && props.currentImageCount < props.totalImageCount);

// The funny potato + ghost art is intentional brand personality — kept, just framed cleanly.
const art = computed(() => {
  if (props.filtered) {
    if (props.totalImageCount === 0) {
      return { src: new URL("../../assets/img/search-potato.webp", import.meta.url).href, headline: "No matches", sub: "No images match the current filters. Try clearing a filter or adjusting your search." };
    }
    return { src: new URL("../../assets/img/search-potato.webp", import.meta.url).href, headline: `All ${props.totalImageCount} matching images shown`, sub: "" };
  }
  if (props.totalImageCount === 0) {
    return { src: new URL("../../assets/img/ghost.webp", import.meta.url).href, headline: "Nothing here yet", sub: `There are no images in ${activeProject.value.name} yet. Upload some to get started.` };
  }
  return { src: new URL("../../assets/img/potato.webp", import.meta.url).href, headline: `That's all ${props.totalImageCount} images`, sub: "" };
});
</script>
