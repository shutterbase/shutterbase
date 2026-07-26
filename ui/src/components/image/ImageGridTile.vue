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

    <!-- AI detection status -->
    <div v-if="image.aiStatus" :title="aiTitle" class="absolute left-2 top-2 flex items-center gap-1 rounded-full bg-primary-950/70 px-1.5 py-1 shadow-sm">
      <SparklesIcon v-if="image.aiStatus === 'done'" class="h-3.5 w-3.5 text-accent-300" />
      <ArrowPathIcon v-else-if="image.aiStatus === 'processing'" class="h-3.5 w-3.5 animate-spin text-accent-300" />
      <ClockIcon v-else-if="image.aiStatus === 'pending'" class="h-3.5 w-3.5 text-primary-200" />
      <ExclamationTriangleIcon v-else class="h-3.5 w-3.5 text-red-400" />
      <span v-if="aiLabel" class="label-mono-sm text-primary-100">{{ aiLabel }}</span>
    </div>

    <!-- caption: below the image in gallery mode, EXIF-style hover readout otherwise -->
    <figcaption v-if="density === 'gallery'" class="px-3 py-2.5">
      <p class="truncate text-sm font-medium text-primary-800 dark:text-primary-100">{{ image.computedFileName }}</p>
      <p class="label-mono-sm mt-1 truncate text-primary-500 dark:text-primary-400">{{ capturedAt }}</p>
    </figcaption>
    <figcaption v-else class="pointer-events-none absolute inset-x-0 bottom-0 bg-primary-950/80 px-2.5 py-2 opacity-0 transition-opacity duration-200 group-hover:opacity-100">
      <p class="truncate text-xs font-medium text-white">{{ image.computedFileName }}</p>
      <p class="label-mono-sm mt-0.5 truncate text-accent-300">{{ capturedAt }}</p>
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { aiBadgeLabel, aiBadgeTitle } from "src/util/aiDetection";
import { ImageWithTagsType } from "src/types/custom";
import { devPlaceholder } from "src/util/devPlaceholder";
import { computed, ref } from "vue";
import { ArrowPathIcon, CheckIcon, ClockIcon, ExclamationTriangleIcon, PhotoIcon, SparklesIcon } from "@heroicons/vue/24/solid";

type Density = "gallery" | "comfortable" | "dense";

interface Props {
  image: ImageWithTagsType;
  selected?: boolean;
  density?: Density;
  // global queue position for pending images (from api.ai.queueStatus)
  aiPosition?: number;
}
const props = withDefaults(defineProps<Props>(), {
  selected: false,
  density: "comfortable",
  aiPosition: 0,
});

const emit = defineEmits<{
  select: [string, MouseEvent];
}>();

const capturedAt = computed(() => dateTimeUtil.dateTimeFromBackend(props.image.capturedAtCorrected));
const aiLabel = computed(() => aiBadgeLabel(props.image.aiStatus, props.aiPosition));
const aiTitle = computed(() => aiBadgeTitle(props.image.aiStatus, props.aiPosition, props.image.aiError));

// Missing thumbnail: dev builds swap in the deterministic local placeholder
// (see devPlaceholder.ts); in prod the neutral icon tile shows instead.
const failed = ref(false);
const src = ref<string>(props.image.downloadUrls?.[`256`] ?? "");

function onError() {
  const placeholder = devPlaceholder(props.image.id);
  if (placeholder && src.value !== placeholder) {
    src.value = placeholder;
  } else {
    failed.value = true;
  }
}
</script>
