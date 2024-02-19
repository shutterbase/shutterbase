<template>
  <div></div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

interface Props {
  title: string;
  todos?: string[];
  active: boolean;
}
const props = withDefaults(defineProps<Props>(), {
  todos: () => [],
});
</script>
