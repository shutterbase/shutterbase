<template>
  <span @click="removeTag" :class="[`inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset`, tagColor, removableClasses()]">{{
    tagAssignment.expand.imageTag.name
  }}</span>
</template>
<script lang="ts" setup>
import { ImageTagAssignmentType } from "src/types/custom";
import { computed } from "vue";

interface Props {
  tagAssignment: ImageTagAssignmentType;
}
const props = withDefaults(defineProps<Props>(), {});

const tagType = props.tagAssignment.expand.imageTag.type;

/*
export enum ImageTagsTypeOptions {
	"default" = "default",
	"manual" = "manual",
	"custom" = "custom",
}

export enum ImageTagAssignmentsTypeOptions {
	"manual" = "manual",
	"inferred" = "inferred",
	"default" = "default",
}
*/

// tag colors for light and dark mode
const tagColor = computed(() => {
  switch (tagType) {
    case "default":
      return "bg-blue-200 dark:bg-blue-800 text-gray-900 dark:text-gray-100 ring-blue-200 dark:ring-blue-700";
    case "manual":
      return "bg-green-200 dark:bg-green-800 text-gray-900 dark:text-gray-100 ring-green-200 dark:ring-green-700";
    case "custom":
      return "bg-yellow-200 dark:bg-yellow-800 text-gray-900 dark:text-gray-100 ring-yellow-200 dark:ring-yellow-700";
    // TODO: add inferred tag color
  }
});

function removable(): boolean {
  return tagType !== "default";
}

function removableClasses() {
  if (removable()) {
    return "cursor-pointer";
  } else {
    return "cursor-not-allowed";
  }
}

function removeTag() {
  if (!removable()) {
    return;
  }
  console.log(`remove tag ${props.tagAssignment.expand.imageTag.name} with assignment id ${props.tagAssignment.id}`);
}
</script>
<script lang="ts"></script>
