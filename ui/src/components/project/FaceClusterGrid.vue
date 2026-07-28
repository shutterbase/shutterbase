<template>
  <div :class="['grid gap-2', cols === 4 ? 'grid-cols-2 sm:grid-cols-4' : cols === 1 ? 'grid-cols-1' : 'grid-cols-2']">
    <div v-for="(item, i) in items" :key="i" class="relative overflow-hidden rounded-sm bg-primary-100 dark:bg-primary-900">
      <img :src="item.image.downloadUrls?.['512']" :alt="item.image.computedFileName" loading="lazy" class="aspect-square w-full object-cover" />
      <div :style="coverBoxStyle(item, item.image)" class="absolute rounded-sm border-2 border-accent-400/90 shadow-[0_0_0_1px_rgba(0,0,0,0.4)]"></div>
    </div>
    <div v-if="items.length === 0" class="col-span-full py-8 text-center text-sm text-primary-500 dark:text-primary-400">No appearances in this project</div>
  </div>
</template>

<script setup lang="ts">
import { AiPersonImage } from "src/api/ai";
import { Image } from "src/types/api";
import { faceBoxStyle } from "src/util/aiDetection";

withDefaults(defineProps<{ items: AiPersonImage[]; cols?: 1 | 2 | 4 }>(), { cols: 2 });

// The tiles are square (object-cover), so the relative bbox needs the
// cover-crop correction before faceBoxStyle's percentage mapping.
function coverBoxStyle(face: AiPersonImage, image: Image) {
  const w = image.width ?? 0;
  const h = image.height ?? 0;
  if (!w || !h || w === h) return faceBoxStyle(face);
  const scale = Math.min(w, h);
  const cropX = (w - scale) / 2 / w;
  const cropY = (h - scale) / 2 / h;
  return faceBoxStyle({
    x: ((face.x - cropX) * w) / scale,
    y: ((face.y - cropY) * h) / scale,
    w: (face.w * w) / scale,
    h: (face.h * h) / scale,
  });
}
</script>
