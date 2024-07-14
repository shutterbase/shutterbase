<template>
  <div
    v-if="item"
    class="max-2xl:hidden w-80 top-16 fixed inset-y-0 left-0 bg-gray-50 dark:bg-primary-900 text-gray-900 dark:text-gray-200 shadow-lg z-10 overflow-y-scroll no-scrollbar"
  >
    <div class="p-5">
      <h3 class="text-lg font-medium pb-6 border-b dark:border-primary-400">Image Details</h3>
      <div class="border-b py-6 dark:border-primary-400">
        <div class="pb-2">
          <p class="text-sm font-medium">Name</p>
          <p class="text-sm">{{ item.computedFileName }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Original file name</p>
          <p class="text-sm">{{ item.fileName }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Corrected capture time</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.capturedAtCorrected) }}</p>
        </div>
        <div class="pb-2">
          <p class="text-sm font-medium">Original capture time</p>
          <p class="text-sm">{{ dateTimeFromBackend(item.capturedAt) }}</p>
        </div>
      </div>

      <div class="border-b pb-6 dark:border-primary-400">
        <h3 class="text-lg font-medium py-6">Image Tags</h3>
        <div class="flex">
          <ImageTagBadge class="mr-2 mb-2" v-for="tagAssignment in tagAssignments" :key="tagAssignment.id" :tagAssignment="tagAssignment" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ImagesResponse } from "src/types/pocketbase";
import { ImageWithTagsType } from "src/types/custom";
import { dateTimeFromBackend } from "src/util/dateTimeUtil";
import ImageTagBadge from "src/components/image/ImageTagBadge.vue";
import { computed } from "vue";
interface Props {
  item: ImageWithTagsType | null;
}
const props = withDefaults(defineProps<Props>(), {});

const tagAssignments = computed(() => {
  return props.item?.expand.image_tag_assignments_via_image || [];
});
</script>
<style>
/* Hide scrollbar for Chrome, Safari and Opera */
.no-scrollbar::-webkit-scrollbar {
  display: none;
}

/* Hide scrollbar for IE, Edge and Firefox */
.no-scrollbar {
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
}
</style>
