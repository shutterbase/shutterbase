<template>
  <div class="fixed inset-x-0 bottom-0 z-10 border-t border-primary-200 bg-primary-50/90 backdrop-blur dark:border-primary-800 dark:bg-primary-950/90">
    <div class="relative h-24 overflow-hidden" role="listbox" aria-label="Film strip">
      <!-- The track sits at the horizontal center; translating it by the current
           slot offset centers the active frame without any width measurement.
           Transform-only, so navigation animates on the compositor. -->
      <div
        class="absolute inset-y-0 left-1/2 transition-transform duration-200 ease-out motion-reduce:transition-none"
        :style="{ transform: `translateX(${-(currentIndex + 0.5) * SLOT}px)` }"
      >
        <button
          v-for="entry in windowed"
          :key="entry.image.id"
          type="button"
          role="option"
          :aria-selected="entry.index === currentIndex"
          :title="entry.image.computedFileName"
          :style="{ left: `${entry.index * SLOT}px`, width: `${SLOT}px` }"
          class="absolute inset-y-0 flex items-center justify-center focus:outline-none"
          @click="emit('select', entry.index)"
        >
          <img
            :src="srcFor(entry.image)"
            :alt="entry.image.computedFileName"
            loading="lazy"
            draggable="false"
            @error="onThumbError(entry.image)"
            :class="[
              'h-16 w-[92px] rounded-sm bg-primary-100 object-cover transition-opacity duration-200 dark:bg-primary-900',
              entry.index === currentIndex
                ? 'ring-2 ring-accent-500 ring-offset-2 ring-offset-primary-50 dark:ring-offset-primary-950'
                : 'opacity-50 hover:opacity-100',
            ]"
          />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { ImageWithTagsType } from "src/types/custom";
import { devPlaceholder } from "src/util/devPlaceholder";

const SLOT = 100; // px per frame: 92px thumb + 8px gap; keeps the offset math integer

interface Props {
  images: ImageWithTagsType[];
  currentIndex: number;
}
const props = defineProps<Props>();
const emit = defineEmits<{
  select: [number];
}>();

// Only frames near the current one are mounted; anything farther than RADIUS
// slots is off-screen on any viewport, so long sessions with thousands of
// loaded frames stay cheap.
const RADIUS = 30;
const windowed = computed(() => {
  const start = Math.max(0, props.currentIndex - RADIUS);
  const end = Math.min(props.images.length, props.currentIndex + RADIUS + 1);
  return props.images.slice(start, end).map((image, i) => ({ image, index: start + i }));
});

const srcOverrides = reactive<Record<string, string>>({});
function srcFor(image: ImageWithTagsType): string {
  return srcOverrides[image.id] ?? image.downloadUrls?.["256"] ?? "";
}
function onThumbError(image: ImageWithTagsType) {
  const placeholder = devPlaceholder(image.id);
  if (placeholder && srcOverrides[image.id] !== placeholder) {
    srcOverrides[image.id] = placeholder;
  }
}
</script>
