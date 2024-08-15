<template>
  <div :id="`grid-tile-${image.id}`" @click="(e) => emit('select', image.id, e)" :class="[`cursor-pointer group relative p-4 sm:p-6`, selectedClasses]">
    <div class="aspect-h-1 aspect-w-1 overflow-hidden rounded-lg group-hover:opacity-75">
      <img :src="image.downloadUrls[`256`]" :alt="image.computedFileName" class="max-h-44 w-full object-cover object-center" />
    </div>
    <div class="pt-6 text-center">
      <h3 class="text-sm font-medium text-gray-900 dark:text-gray-100">
        <div>
          <span aria-hidden="true" class="absolute inset-0"></span>
          {{ image.computedFileName }}
        </div>
      </h3>
      <div class="mt-3 flex flex-col items-center">
        <div class="flex items-center text-gray-600 dark:text-gray-400">{{ getTagsList(image) }}</div>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{{ dateTimeUtil.dateTimeFromBackend(image.capturedAtCorrected) }}</p>
      </div>
      <!-- <p class="mt-4 text-base font-medium text-gray-900 dark:text-gray-100">$149</p> -->
    </div>
  </div>
</template>

<script setup lang="ts">
import * as dateTimeUtil from "src/util/dateTimeUtil";
import { ImageWithTagsType, ImageTagAssignmentType } from "src/types/custom";
import { computed } from "vue";

interface Props {
  image: ImageWithTagsType;
  selected?: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  selected: false,
});

const selectedClasses = computed(() => {
  const defaultBorder = "border-b border-t border-r border-gray-200 dark:border-gray-600";
  const selectedBorder = "border-b border-t border-r border-gray-200 dark:border-gray-600 bg-gray-300 dark:bg-gray-700";
  return props.selected ? selectedBorder : defaultBorder;
});

const emit = defineEmits<{
  select: [string, MouseEvent];
}>();

function getTagsList(image: ImageWithTagsType) {
  const imageTagAssignments = image.expand?.image_tag_assignments_via_image || [];
  return imageTagAssignments.map((imageTagAssignment: ImageTagAssignmentType) => imageTagAssignment.expand.imageTag.name).join(", ");
}
</script>
