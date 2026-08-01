<template>
  <div>
    <div class="relative aspect-square w-full overflow-hidden rounded-sm bg-primary-100 dark:bg-primary-900">
      <template v-if="current && crop">
        <img :src="src(current)" :alt="current.image.computedFileName" :style="crop.img" class="absolute max-w-none" loading="lazy" />
        <div :style="crop.box" class="absolute rounded-sm border-2 border-accent-400/90 shadow-[0_0_0_1px_rgba(0,0,0,0.4)]"></div>
      </template>
      <img v-else-if="current" :src="src(current)" :alt="current.image.computedFileName" class="h-full w-full object-cover" loading="lazy" />
      <div v-else class="flex h-full w-full items-center justify-center">
        <span class="label-mono-sm text-primary-400 dark:text-primary-600">{{ loading ? "…" : "no sample" }}</span>
      </div>
    </div>
    <div v-if="total > 1" class="mt-1.5 flex items-center justify-center gap-1">
      <button
        @click.stop.prevent="step(-1)"
        :disabled="index === 0 || loading"
        title="Previous photo"
        class="cursor-pointer rounded p-1 text-primary-500 transition-colors hover:text-primary-900 disabled:pointer-events-none disabled:opacity-30 dark:text-primary-400 dark:hover:text-white"
      >
        <ChevronLeftIcon class="h-4 w-4" />
      </button>
      <span class="label-mono-sm min-w-12 text-center text-primary-500 dark:text-primary-400">{{ index + 1 }} / {{ total }}</span>
      <button
        @click.stop.prevent="step(1)"
        :disabled="atEnd || loading"
        title="Next photo"
        class="cursor-pointer rounded p-1 text-primary-500 transition-colors hover:text-primary-900 disabled:pointer-events-none disabled:opacity-30 dark:text-primary-400 dark:hover:text-white"
      >
        <ChevronRightIcon class="h-4 w-4" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { ChevronLeftIcon, ChevronRightIcon } from "@heroicons/vue/24/solid";
import { api } from "src/api";
import { AiPersonImage, AiPersonImagesPage } from "src/api/ai";
import { faceCropStyle } from "src/util/aiDetection";

// One face of a person cluster, cropped with generous margin, with arrows to
// flip through the cluster's appearances in place. Seed it with a single
// `sample` (People cards) or a prefetched `page0` (merge review); with no seed
// it fetches page 0 itself (merged-cluster inspection). Further pages load on
// demand via the global person-images endpoint. Seeds are captured on setup —
// key the component when the person or seed changes.
const props = withDefaults(
  defineProps<{
    personRef: string;
    sample?: AiPersonImage;
    page0?: AiPersonImagesPage;
    total?: number; // known appearance count when only a sample is given
    raw?: boolean; // skip merge-group resolution (inspect one cluster's own faces)
  }>(),
  { raw: false },
);

const PAGE_SIZE = 8;

const items = ref<AiPersonImage[]>(props.page0?.items ?? (props.sample ? [props.sample] : []));
const index = ref(0);
const page = ref(props.page0 ? 0 : -1);
const hasMore = ref(props.page0 ? props.page0.hasMore : true);
const total = ref(props.page0?.total ?? props.total ?? items.value.length);
const loading = ref(false);

const current = computed(() => items.value[index.value]);
const atEnd = computed(() => !hasMore.value && index.value >= items.value.length - 1);
const crop = computed(() => (current.value ? faceCropStyle(current.value, current.value.image.width ?? 0, current.value.image.height ?? 0) : null));

function src(item: AiPersonImage) {
  return item.image.downloadUrls?.["512"] ?? "";
}

// fetchNext loads the next page and leaves `index` on the face the user asked
// for (the one after the current position).
async function fetchNext() {
  loading.value = true;
  try {
    const firstPage = page.value < 0;
    const resp = await api.ai.personImagesGlobal(props.personRef, page.value + 1, PAGE_SIZE, props.raw);
    page.value++;
    hasMore.value = resp.hasMore;
    if (resp.total) total.value = resp.total;
    if (resp.items.length === 0) {
      hasMore.value = false;
      return;
    }
    if (firstPage && props.sample) {
      // the single-sample seed is replaced by the real first page; continue
      // right after the sample when it is part of that page
      const at = resp.items.findIndex((it) => it.image.id === props.sample?.image.id && it.x === props.sample?.x);
      items.value = resp.items;
      index.value = Math.min(at + 1, resp.items.length - 1);
    } else {
      const wasEmpty = items.value.length === 0;
      items.value = [...items.value, ...resp.items];
      index.value = wasEmpty ? 0 : Math.min(index.value + 1, items.value.length - 1);
    }
  } catch {
    hasMore.value = false;
  } finally {
    loading.value = false;
  }
}

async function step(dir: 1 | -1) {
  const next = index.value + dir;
  if (next < 0) return;
  if (next < items.value.length) {
    index.value = next;
    return;
  }
  if (hasMore.value && !loading.value) await fetchNext();
}

// warm the next face's thumbnail so flipping feels instant
watch(
  [index, items],
  () => {
    const next = items.value[index.value + 1];
    if (next) new Image().src = src(next);
  },
  { immediate: true },
);

onMounted(() => {
  if (items.value.length === 0) fetchNext();
});
</script>
