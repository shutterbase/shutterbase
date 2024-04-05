<template>
  <div>
    <!-- <Table class="mx-auto max-w-7xl"></Table> -->
    <button @click="doStuff('success')">Success</button>
    <br />
    <button @click="doStuff('info')">Info</button>
    <br />
    <button @click="doStuff('error')">Error</button>
    <br />
    <button @click="doStuff('warning')">Warning</button>
  </div>
</template>

<script setup lang="ts">
import { Ref, ref } from "vue";

import pb from "src/boot/pocketbase";
import { showNotificationToast } from "src/boot/mitt";

const counter = ref(0);

async function doStuff(type: "success" | "error" | "warning" | "info") {
  showNotificationToast({
    headline: `Attention ${++counter.value}`,
    message: "Doing stuff",
    type,
  });
}
</script>
