<template>
  <section class="mt-8">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold tracking-tight text-primary-900 dark:text-white">Images</h2>
        <p class="mt-1 text-sm text-primary-500 dark:text-primary-400">{{ images.length }} in this upload — progress shows right on the tile.</p>
      </div>
    </div>

    <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
      <figure
        v-for="image in images"
        :key="image.id || image.originalFileName"
        class="group relative overflow-hidden rounded-lg border border-primary-200 bg-primary-100/50 dark:border-primary-800 dark:bg-primary-900/40"
      >
        <div class="aspect-[3/2] w-full">
          <img v-if="tileSrc(image)" :src="tileSrc(image)" :alt="image.originalFileName" class="h-full w-full object-cover" loading="lazy" />
          <div v-else class="flex h-full w-full items-center justify-center">
            <PhotoIcon class="h-8 w-8 text-primary-300 dark:text-primary-600" />
          </div>

          <!-- progress overlay while the pipeline runs (transcript 19:22:
               "sieht auf dem Image Preview, dass der grad hochgeladen wird") -->
          <div v-if="image.status !== ImageStatus.DONE" class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-primary-950/60 backdrop-blur-[1px]">
            <ExclamationTriangleIcon v-if="image.status === ImageStatus.ERROR" class="h-6 w-6 text-error-400" />
            <template v-else>
              <span class="h-6 w-6 animate-spin rounded-full border-2 border-white/30 border-t-white"></span>
              <div class="h-1 w-2/3 overflow-hidden rounded-full bg-white/20">
                <div class="h-full rounded-full bg-white transition-all" :style="{ width: `${image.progress}%` }"></div>
              </div>
            </template>
            <span v-if="image.status === ImageStatus.ERROR && image.errorMessage" class="px-2 text-center text-[10px] font-medium leading-tight text-white/90">
              {{ image.errorMessage }}
            </span>
            <span v-else class="text-[10px] font-medium uppercase tracking-wide text-white/90">{{ image.status }}</span>
          </div>

          <!-- remove, on hover, only once persisted -->
          <button
            v-if="allowEdit && image.status === ImageStatus.DONE"
            type="button"
            class="absolute right-1.5 top-1.5 hidden h-7 w-7 cursor-pointer items-center justify-center rounded-md bg-primary-950/70 text-white transition-colors hover:bg-error-600 group-hover:flex"
            :aria-label="`Remove ${image.originalFileName}`"
            @click="emit('remove', image)"
          >
            <TrashIcon class="h-4 w-4" />
          </button>
        </div>
        <figcaption class="space-y-0.5 px-2 py-1.5">
          <p class="truncate text-xs font-medium text-primary-800 dark:text-primary-100" :title="fileName(image)">{{ fileName(image) }}</p>
          <p class="truncate text-[10px] tabular-nums text-primary-400">{{ caption(image) }}</p>
        </figcaption>
      </figure>
    </div>

    <p v-if="images.length === 0" class="mt-4 rounded-md border border-dashed border-primary-200 px-4 py-8 text-center text-sm text-primary-400 dark:border-primary-700">
      No images yet — drop files above to start uploading.
    </p>
  </section>
</template>

<script setup lang="ts">
import { ExclamationTriangleIcon, PhotoIcon, TrashIcon } from "@heroicons/vue/24/outline";
import { Image, ImageStatus } from "src/util/fileProcessor";
import { fileSize } from "src/util/fileUtil";

defineProps<{ images: Image[]; allowEdit: boolean }>();
const emit = defineEmits<{ remove: [Image] }>();

function tileSrc(image: Image): string | undefined {
  if (image.thumbnail) return `data:image/jpeg;base64, ${image.thumbnail}`;
  return image.downloadUrls?.["256"];
}

const fileName = (image: Image) => image.computedFileName || image.originalFileName;

function caption(image: Image): string {
  const parts: string[] = [];
  if (image.correctedTime) parts.push(image.correctedTime.toFormat("HH:mm:ss"));
  if (image.size) parts.push(fileSize(image.size));
  return parts.join(" · ") || "—";
}
</script>
