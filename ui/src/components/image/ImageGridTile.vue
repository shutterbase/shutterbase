<template>
  <figure
    :id="`grid-tile-${image.id}`"
    @click="(e) => emit('select', image.id, e)"
    :class="[
      'group relative cursor-pointer select-none overflow-hidden bg-primary-100 dark:bg-primary-900 transition-shadow',
      density === 'gallery' ? 'mb-4 break-inside-avoid rounded-lg shadow-panel dark:shadow-panel-dark' : density === 'dense' ? 'rounded-none' : 'rounded-md',
      selected ? 'ring-2 ring-accent-500 ring-offset-2 ring-offset-primary-50 dark:ring-offset-primary-950' : 'ring-0',
    ]"
  >
    <img
      v-if="!failed"
      :src="src"
      @error="onError"
      :alt="image.computedFileName"
      loading="lazy"
      :class="['w-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.03]', density === 'gallery' ? 'h-auto' : 'aspect-square']"
    />
    <div v-else :class="['flex w-full items-center justify-center bg-primary-100 dark:bg-primary-900', density === 'gallery' ? 'aspect-[4/3]' : 'aspect-square']">
      <PhotoIcon class="h-8 w-8 text-primary-300 dark:text-primary-700" />
    </div>

    <!-- selection check -->
    <div v-if="selected" class="absolute right-2 top-2 rounded-full bg-accent-600 p-1 shadow-sm">
      <CheckIcon class="h-3.5 w-3.5 text-white" />
    </div>

    <!-- caption: below the image in gallery mode, hover scrim otherwise -->
    <figcaption v-if="density === 'gallery'" class="px-3 py-2.5">
      <p class="truncate text-sm font-medium text-primary-800 dark:text-primary-100">{{ image.computedFileName }}</p>
      <p class="mt-0.5 truncate font-data text-xs text-primary-500 dark:text-primary-400">{{ capturedAt }}</p>
    </figcaption>
    <figcaption
      v-else
      class="pointer-events-none absolute inset-x-0 bottom-0 bg-primary-950/75 px-2 py-1.5 opacity-0 transition-opacity duration-200 group-hover:opacity-100"
    >
      <p class="truncate text-xs font-medium text-white">{{ image.computedFileName }}</p>
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { ImageWithTagsType } from "src/types/custom";
import { computed, ref } from "vue";
import { CheckIcon, PhotoIcon } from "@heroicons/vue/24/solid";

type Density = "gallery" | "comfortable" | "dense";

interface Props {
  image: ImageWithTagsType;
  selected?: boolean;
  density?: Density;
}
const props = withDefaults(defineProps<Props>(), {
  selected: false,
  density: "comfortable",
});

const emit = defineEmits<{
  select: [string, MouseEvent];
}>();

const capturedAt = computed(() => dateTimeUtil.dateTimeFromBackend(props.image.capturedAtCorrected));

// In dev there is no S3, so the presigned thumbnail URLs 404. Fall back to a
// deterministic placeholder photo so the gallery layout is reviewable; in prod
// a genuinely missing thumbnail shows the neutral placeholder instead.
// ponytail: dev-only picsum fallback, drop the whole block once S3 dev fixtures exist.
const failed = ref(false);
const src = ref<string>(props.image.downloadUrls?.[`256`] ?? "");

function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
  return Math.abs(h);
}

function onError() {
  if (import.meta.env.DEV && !src.value.includes("picsum.photos")) {
    const ratios = [
      [4, 3],
      [3, 4],
      [1, 1],
      [3, 2],
      [2, 3],
    ];
    const [w, h] = ratios[hash(props.image.id) % ratios.length];
    src.value = `https://picsum.photos/seed/${encodeURIComponent(props.image.id)}/${w * 220}/${h * 220}`;
  } else {
    failed.value = true;
  }
}
</script>
