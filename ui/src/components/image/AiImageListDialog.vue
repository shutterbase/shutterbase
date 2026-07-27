<template>
  <div v-show="shown" class="relative z-10" role="dialog" aria-modal="true">
    <div v-show="shown" class="fixed inset-0 bg-primary-950/60 backdrop-blur-sm transition-opacity" @click="emit('close')"></div>

    <div v-show="shown" class="fixed inset-0 z-10 w-screen overflow-y-auto p-4 sm:p-6 md:p-16">
      <div
        class="mx-auto max-w-4xl transform overflow-hidden rounded-xl border border-primary-200 bg-surface shadow-2xl transition-all dark:border-primary-800 dark:bg-surface-dark"
      >
        <div class="flex items-center justify-between border-b border-primary-200 px-5 py-4 dark:border-primary-800">
          <div>
            <h3 class="display text-lg text-primary-900 dark:text-white">{{ title }}</h3>
            <p class="label-mono-sm mt-0.5 text-primary-500 dark:text-primary-400">{{ subtitle }}</p>
          </div>
          <div class="flex items-center gap-4">
            <label v-if="mode === 'person'" class="flex cursor-pointer items-center gap-1.5 text-sm text-primary-600 dark:text-primary-300">
              <input v-model="crossProject" type="checkbox" class="accent-accent-600" />
              all my projects
            </label>
            <XMarkIcon class="h-5 w-5 cursor-pointer text-primary-400 hover:text-primary-600 dark:hover:text-primary-200" @click="emit('close')" />
          </div>
        </div>

        <div class="max-h-[65vh] overflow-y-auto p-5">
          <div v-if="items.length" class="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5">
            <div v-for="entry in items" :key="entry.image.id" class="relative">
              <ImageGridTile :image="entry.image" density="dense" @select="(id) => emit('select', id)" />
              <span
                v-if="mode === 'similar' && entry.similarity !== undefined"
                class="label-mono-sm pointer-events-none absolute bottom-1 right-1 rounded bg-primary-950/70 px-1 text-accent-300"
                >{{ Math.round(entry.similarity * 100) }}%</span
              >
              <span
                v-if="entry.image.project.id !== projectId"
                class="label-mono-sm pointer-events-none absolute bottom-1 right-1 max-w-full truncate rounded bg-primary-950/70 px-1 text-accent-300"
                >{{ entry.image.project.name }}</span
              >
            </div>
          </div>
          <p v-else-if="!loading" class="py-8 text-center text-sm text-primary-500 dark:text-primary-400">
            {{ notAnalyzed ? "This image has not been analyzed yet." : "Nothing found." }}
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
          <span class="label-mono-sm text-primary-500 dark:text-primary-400">{{ pageLabel }}</span>
          <button
            class="text-sm font-medium text-accent-600 underline disabled:cursor-default disabled:text-primary-400 disabled:no-underline dark:text-accent-400 dark:disabled:text-primary-600"
            :disabled="!hasNext || loading"
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
// Paginated result grid for the two AI lookups: "photos of this person"
// (mode=person, from a clicked face box) and "similar images" (mode=similar).
// Selecting a result hands the image id back to the viewer.
import { computed, ref, watch } from "vue";
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
  mode: "person" | "similar";
  projectId: string;
  // person mode: the AI server's person handle; similar mode: the query image
  personRef?: string;
  imageId?: string;
}
const props = defineProps<Props>();
const emit = defineEmits<{ close: []; select: [string] }>();

const items = ref<Entry[]>([]);
const page = ref(0);
const total = ref(0);
const hasMore = ref(false);
const loading = ref(false);
const notAnalyzed = ref(false);
// person mode: also query the user's other projects (person ids are global)
const crossProject = ref(false);

const title = computed(() => (props.mode === "person" ? "Photos of this person" : "Similar images"));
const subtitle = computed(() => {
  if (props.mode === "person" && total.value > 0) return `${total.value} photo${total.value === 1 ? "" : "s"} ${crossProject.value ? "across your projects" : "in this project"}`;
  return "";
});
const hasNext = computed(() => (props.mode === "person" && !crossProject.value ? (page.value + 1) * PAGE_SIZE < total.value : hasMore.value));
const pageLabel = computed(() => {
  // cross-project pages are per-project slices, so total/PAGE_SIZE math doesn't apply
  if (props.mode === "person" && !crossProject.value && total.value > 0) return `page ${page.value + 1} / ${Math.max(1, Math.ceil(total.value / PAGE_SIZE))}`;
  return `page ${page.value + 1}`;
});

watch(
  () => [props.shown, props.mode, props.personRef, props.imageId, crossProject.value],
  () => {
    if (!props.shown) return;
    page.value = 0;
    load();
  },
);
watch(page, load);

async function load() {
  if (!props.shown) return;
  loading.value = true;
  notAnalyzed.value = false;
  items.value = [];
  try {
    if (props.mode === "person" && props.personRef) {
      const result = await api.ai.personImages(props.projectId, props.personRef, page.value, PAGE_SIZE, crossProject.value);
      items.value = result.items.map((i) => ({ image: i.image }));
      total.value = result.total;
      hasMore.value = result.hasMore;
    } else if (props.mode === "similar" && props.imageId) {
      const result = await api.ai.similar(props.imageId, page.value, PAGE_SIZE);
      items.value = result.items.map((i) => ({ image: i.image, similarity: i.similarity }));
      hasMore.value = result.hasMore;
    }
  } catch (error: any) {
    // 404 = not analyzed yet (or a stale person ref after re-clustering)
    notAnalyzed.value = error?.response?.status === 404;
    total.value = 0;
    hasMore.value = false;
  } finally {
    loading.value = false;
  }
}
</script>
