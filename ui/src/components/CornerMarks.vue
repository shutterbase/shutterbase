<template>
  <div class="pointer-events-none absolute inset-0" aria-hidden="true">
    <span
      v-for="c in corners"
      :key="c"
      :class="['absolute border-accent-500/70', sizeClass, c]"
    />
  </div>
</template>

<script setup lang="ts">
// Viewfinder corner-marks — four L-brackets that frame the parent like a camera's
// focus guides. The parent must be `relative`. A small, recurring brand device.
interface Props {
  size?: "sm" | "md";
}
const props = withDefaults(defineProps<Props>(), { size: "md" });

const sizeClass = props.size === "sm" ? "h-2.5 w-2.5" : "h-3.5 w-3.5";
const corners = [
  "left-0 top-0 border-l-[1.5px] border-t-[1.5px]",
  "right-0 top-0 border-r-[1.5px] border-t-[1.5px]",
  "bottom-0 left-0 border-b-[1.5px] border-l-[1.5px]",
  "bottom-0 right-0 border-b-[1.5px] border-r-[1.5px]",
];
</script>
