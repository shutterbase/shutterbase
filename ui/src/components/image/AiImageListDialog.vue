<template>
  <div v-show="shown" class="relative z-10" role="dialog" aria-modal="true">
    <div v-show="shown" class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity" @click="emit('close')"></div>

    <div v-show="shown" class="fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-16">
      <div
        class="mx-auto max-w-4xl transform overflow-hidden rounded-xl border border-primary-200 bg-surface shadow-2xl transition-all dark:border-primary-800 dark:bg-surface-dark"
      >
        <div class="flex items-center justify-between border-b border-primary-200 px-5 py-4 dark:border-primary-800">
          <h3 class="display text-lg text-primary-900 dark:text-white">
            <template v-if="query"
              >Search: <span class="font-normal italic">“{{ query }}”</span></template
            >
            <template v-else>Similar images</template>
          </h3>
          <XMarkIcon class="h-5 w-5 cursor-pointer text-primary-400 hover:text-primary-600 dark:hover:text-primary-200" @click="emit('close')" />
        </div>

        <div class="max-h-[65vh] overflow-y-auto p-5">
          <div v-if="items.length" class="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5">
            <div v-for="entry in items" :key="entry.image.id" class="relative">
              <ImageGridTile :image="entry.image" density="dense" @select="(id) => emit('select', id)" />
              <span v-if="entry.similarity !== undefined" class="label-mono-sm pointer-events-none absolute bottom-1 right-1 rounded bg-primary-950/70 px-1 text-accent-300"
                >{{ Math.round(entry.similarity * 100) }}%</span
              >
            </div>
          </div>
          <p v-else-if="!loading" class="py-8 text-center text-sm text-primary-500 dark:text-primary-400">
            {{ notAnalyzed ? "This image has not been analyzed yet." : query ? "Nothing matches that description." : "Nothing found." }}
          </p>
          <p v-if="loading" class="py-4 text-center text-sm text-primary-500 dark:text-primary-400">Loading…</p>
        </div>

        <div class="flex items-center justify-between border-t border-primary-200 px-5 py-3 dark:border-primary-800">
          <button
            class="text-sm font-medium text-accent-600 underline disabled:cursor-default disabled:text-primary-400 disabled:no-underline dark:text-accent-400 dark:disabled:text-primary-600"
            :disabled="page === 0 || loading"
            @click="page--"
          >
            previous
          </button>
          <span class="label-mono-sm text-primary-500 dark:text-primary-400">page {{ page + 1 }}</span>
          <button
            class="text-sm font-medium text-accent-600 underline disabled:cursor-default disabled:text-primary-400 disabled:no-underline dark:text-accent-400 dark:disabled:text-primary-600"
            :disabled="!hasMore || loading"
            @click="page++"
          >
            next
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Paginated ranked result grid: "similar images" for the current viewer image
// (imageId), or semantic text search over the project's AI descriptions
// (query). Selecting a result hands the image id back. (Person browsing lives
// in the normal grid via the implicit person filter, not here.)
import { ref, watch } from "vue";
import { XMarkIcon } from "@heroicons/vue/24/outline";
import ImageGridTile from "src/components/image/ImageGridTile.vue";
import { api } from "src/api";
import { Image } from "src/types/api";

const PAGE_SIZE = 20;

interface Entry {
  image: Image;
  similarity?: number;
}

interface Props {
  shown: boolean;
  imageId?: string;
  // free-text semantic search; takes precedence over imageId
  query?: string;
  projectId?: string;
}
const props = defineProps<Props>();
const emit = defineEmits<{ close: []; select: [string] }>();

const items = ref<Entry[]>([]);
const page = ref(0);
const hasMore = ref(false);
const loading = ref(false);
const notAnalyzed = ref(false);

watch(
  () => [props.shown, props.imageId, props.query],
  () => {
    if (!props.shown) return;
    page.value = 0;
    load();
  },
);
watch(page, load);

async function load() {
  if (!props.shown) return;
  const semantic = !!props.query && !!props.projectId;
  if (!semantic && !props.imageId) return;
  loading.value = true;
  notAnalyzed.value = false;
  items.value = [];
  try {
    const result = semantic ? await api.ai.search(props.projectId!, props.query!, page.value, PAGE_SIZE) : await api.ai.similar(props.imageId!, page.value, PAGE_SIZE);
    items.value = result.items.map((i) => ({ image: i.image, similarity: i.similarity }));
    hasMore.value = result.hasMore;
  } catch (error: any) {
    // 404 = not analyzed yet
    notAnalyzed.value = error?.response?.status === 404;
    hasMore.value = false;
  } finally {
    loading.value = false;
  }
}
</script>
