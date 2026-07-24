<template>
  <div class="w-64 grid grid-cols-1 auto-rows-min justify-items-center gap-3">
    <div
      class="w-64 h-64 rounded-lg border border-primary-200 bg-surface-muted bg-contain bg-no-repeat bg-center dark:border-primary-800 dark:bg-surface-dark-muted"
      ref="qrCode"
    ></div>
    <div class="font-data text-sm text-primary-700 dark:text-primary-200">{{ timeString }}</div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import * as websocket from "src/util/websocket";

import init, { get_time_qr_code_image } from "image-wasm";

const timeString = ref<string>("");
const wasmInitialized = ref<boolean>(false);
const qrCode = ref<HTMLDivElement>();

// Server time is the reference (all display devices must agree, hence the ws
// tick). Each tick re-anchors; between ticks we advance the second off the
// monotonic clock, so the QR refreshes the instant a new second begins.
let anchorServerSecond = 0; // unix seconds from the last server tick
let anchorLocalMs = 0; // performance.now() when that tick arrived
let haveAnchor = false;
let lastRenderedSecond = -1;
let pollHandle: ReturnType<typeof setInterval> | undefined;

const websocketListenerId = websocket.on({ object: "time" }, ({ data }) => {
  anchorServerSecond = Number(data);
  anchorLocalMs = performance.now();
  haveAnchor = true;
});

async function renderSecond(second: number) {
  lastRenderedSecond = second; // set before await so the poll can't re-enter for the same second
  timeString.value = new Date(second * 1000).toLocaleString();
  const qrCodeResult = await get_time_qr_code_image(String(second));
  if (qrCode.value) {
    qrCode.value.style.backgroundImage = `url("data:image/png;base64,${qrCodeResult.base64}")`;
  }
}

onMounted(async () => {
  websocket.connect();
  await init();
  wasmInitialized.value = true;
  // ponytail: 200ms poll of the interpolated server second; regenerate only on
  // rollover. Latency bias is ~one network hop (tens of ms) — well under our
  // sub-second target; correct it with an RTT estimate only if that stops holding.
  pollHandle = setInterval(() => {
    if (!wasmInitialized.value || !haveAnchor) return;
    const currentSecond = anchorServerSecond + Math.floor((performance.now() - anchorLocalMs) / 1000);
    if (currentSecond !== lastRenderedSecond) void renderSecond(currentSecond);
  }, 200);
});

onUnmounted(() => {
  if (pollHandle) clearInterval(pollHandle);
  websocket.disconnect();
  websocket.off(websocketListenerId);
});
</script>
<style scoped></style>
