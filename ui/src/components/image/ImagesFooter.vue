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

    <!-- end / empty states — suppressed during the initial load so the empty-project art never flashes -->
    <div v-else-if="!loading || currentImageCount > 0" class="flex flex-col items-center px-6 pt-14 text-center">
      <div class="relative">
        <CornerMarks />
        <img :src="art.src" :alt="art.headline" class="h-52 w-52 rounded-xl object-cover sm:h-60 sm:w-60" />
      </div>
      <p class="label-mono mt-8 text-primary-500 dark:text-primary-400">{{ art.kicker }}</p>
      <h2 class="display mt-2.5 text-2xl text-primary-900 dark:text-white">{{ art.headline }}</h2>
      <p v-if="art.sub" class="mt-2 max-w-md text-sm text-primary-500 dark:text-primary-400">{{ art.sub }}</p>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useUserStore } from "src/stores/user-store";
import { ArrowDownIcon } from "@heroicons/vue/24/outline";
import CornerMarks from "src/components/CornerMarks.vue";

interface Props {
  totalImageCount: number;
  currentImageCount: number;
  filtered: boolean;
  loading?: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  totalImageCount: 0,
  currentImageCount: 0,
  filtered: false,
  loading: false,
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
      return {
        src: new URL("../../assets/img/search-potato.webp", import.meta.url).href,
        kicker: "No matches",
        headline: "Nothing fits that filter",
        sub: "No frames match the current filters. Try clearing a filter or adjusting your search.",
      };
    }
    return {
      src: new URL("../../assets/img/search-potato.webp", import.meta.url).href,
      kicker: "End of results",
      headline: `All ${props.totalImageCount} matching frames`,
      sub: "",
    };
  }
  if (props.totalImageCount === 0) {
    return {
      src: new URL("../../assets/img/ghost.webp", import.meta.url).href,
      kicker: "Empty project",
      headline: "Nothing here yet",
      sub: `There are no frames in ${activeProject.value.name} yet. Upload some to get started.`,
    };
  }
  return { src: new URL("../../assets/img/potato.webp", import.meta.url).href, kicker: "End of gallery", headline: `That's all ${props.totalImageCount} frames`, sub: "" };
});
</script>
