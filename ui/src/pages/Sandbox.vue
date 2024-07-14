<template>
  <div>
    <!-- <Table class="mx-auto max-w-7xl w-full"></Table> -->
    <button @click="doStuff('success')">Success</button>
    <br />
    <button @click="doStuff('info')">Info</button>
    <br />
    <button @click="doStuff('error')">Error</button>
    <br />
    <button @click="doStuff('warning')">Warning</button>
  </div>
  <TaggingPalette :shown="showTaggingPalette" />
</template>

<script setup lang="ts">
import { Ref, ref } from "vue";

import pb from "src/boot/pocketbase";
import TaggingPalette from "src/components/TaggingPalette.vue";
import { emitter, showNotificationToast } from "src/boot/mitt";

const counter = ref(0);

const showTaggingPalette = ref(false);

emitter.on("key-t", () => {
  showTaggingPalette.value = true;
});

emitter.on("key-Escape", () => {
  showTaggingPalette.value = false;
});

async function doStuff(type: "success" | "error" | "warning" | "info") {
  showNotificationToast({
    headline: `Attention ${++counter.value}`,
    message: "Doing stuff",
    type,
  });
}
</script>
