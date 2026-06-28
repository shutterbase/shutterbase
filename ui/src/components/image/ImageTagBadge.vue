<template>
  <span @click="emit('remove', tagAssignment)" :class="[`inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset`, tagColor, removableClasses()]">{{
    tagAssignment.tag.name
  }}</span>
</template>
<script lang="ts" setup>
import { ImageTagAssignmentType } from "src/types/custom";
import { computed } from "vue";

interface Props {
  tagAssignment: ImageTagAssignmentType;
  removable: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  removable: false,
});

const tagType = props.tagAssignment.tag.type;

const emit = defineEmits<{
  remove: [ImageTagAssignmentType];
}>();

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

// tag colours mapped onto the design tokens (default → accent, manual → success,
// custom → warning). The tag name is always rendered, so colour is never the sole signal.
const tagColor = computed(() => {
  switch (tagType) {
    case "default":
      return "bg-accent-500/10 text-accent-700 ring-accent-500/30 dark:bg-accent-500/15 dark:text-accent-200 dark:ring-accent-400/30";
    case "manual":
      return "bg-success-500/10 text-success-700 ring-success-500/30 dark:bg-success-500/15 dark:text-success-300 dark:ring-success-400/30";
    case "custom":
      return "bg-warning-500/15 text-warning-800 ring-warning-500/40 dark:bg-warning-500/15 dark:text-warning-300 dark:ring-warning-400/30";
    // TODO: add inferred tag color
  }
});

function removableClasses() {
  if (props.removable) {
    return "cursor-pointer";
  } else {
    return "cursor-not-allowed";
  }
}
</script>
<script lang="ts"></script>
