<template>
  <div class="w-64 grid grid-cols-1 auto-rows-min justify-items-center">
    <div class="w-64 h-64 bg-gray-50 bg-contain bg-no-repeat bg-center" ref="qrCode"></div>
    <div class="text-base font-semibold leading-7 text-gray-900 dark:text-primary-200">{{ timeString }}</div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { encode, decode, parse, stringify } from "urlencode";
import * as websocket from "src/util/websocket";

import init, { get_time_qr_code_image } from "image-wasm";

const time = ref<string>("");
const timeString = ref<string>("");
const wasmInitialized = ref<boolean>(false);
const qrCodeBackground = ref<string>("");

const qrCode = ref<HTMLDivElement>();

const websocketListenerId = websocket.on({ object: "time" }, async ({ data }) => {
  if (wasmInitialized.value && data !== time.value) {
    time.value = data;
    timeString.value = new Date(parseInt(data) * 1000).toLocaleString();
    const qrCodeResult = await get_time_qr_code_image(data);
    // qrCodeBackground.value = `bg-[url("data:image/svg+xml;charset=UTF-8,${encode(qrCodeResult.svg)}")] bg-center bg-cover`;
    qrCode.value.style.backgroundImage = `url("data:image/png;base64,${qrCodeResult.base64}")`;
  }
});

onMounted(websocket.connect);
onUnmounted(websocket.disconnect);
onUnmounted(() => websocket.off(websocketListenerId));

onMounted(async () => {
  await init();
  wasmInitialized.value = true;
});
</script>
<style scoped></style>
