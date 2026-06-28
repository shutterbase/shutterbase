<!--
  DEV quick-actions panel. Rendered ONLY in dev builds (mounted behind
  import.meta.env.DEV in App.vue), so a production build never ships it. Every
  action calls /api/v1/dev/*, which the backend registers only when DEV=true.
  Unobtrusive: a corner toggle that expands a compact panel.
-->
<template>
  <div class="fixed bottom-3 right-3 z-[9999] font-mono text-xs">
    <button @click="open = !open" class="rounded-full bg-fuchsia-700 text-white w-10 h-10 shadow-lg hover:bg-fuchsia-600" title="Dev quick-actions">
      {{ open ? "×" : "DEV" }}
    </button>

    <div v-if="open" class="mt-2 w-72 rounded-lg bg-gray-900 text-gray-100 p-3 shadow-2xl space-y-3 max-h-[80vh] overflow-y-auto">
      <div class="font-bold text-fuchsia-300">Dev quick-actions</div>

      <!-- Quick login per role -->
      <div class="space-y-1">
        <label class="block text-gray-400">Login as role</label>
        <div class="flex gap-1">
          <select v-model="role" class="flex-1 bg-gray-800 rounded px-1 py-1">
            <option v-for="r in roles" :key="r" :value="r">{{ r }}</option>
          </select>
          <button class="btn" @click="run(() => dev.login({ role }))">go</button>
        </div>
      </div>

      <!-- Impersonate / role toggle -->
      <div class="flex gap-1">
        <input v-model="userId" placeholder="userId" class="flex-1 bg-gray-800 rounded px-1 py-1" />
        <button class="btn" @click="run(() => dev.impersonate(userId))">impersonate</button>
      </div>
      <button class="btn w-full" @click="run(() => dev.roleToggle())">toggle my role</button>

      <!-- Quick time-offset -->
      <div class="space-y-1">
        <label class="block text-gray-400">Time-offset</label>
        <input v-model="cameraId" placeholder="cameraId" class="w-full bg-gray-800 rounded px-1 py-1" />
        <div class="flex gap-1 items-center">
          <input v-model.number="driftSeconds" type="number" placeholder="drift s" class="flex-1 bg-gray-800 rounded px-1 py-1" />
          <label class="flex items-center gap-1"><input v-model="stale" type="checkbox" />stale</label>
          <button class="btn" @click="run(() => dev.timeOffset({ cameraId, driftSeconds, stale }))">set</button>
        </div>
      </div>

      <!-- Quick images -->
      <div class="space-y-1">
        <label class="block text-gray-400">Synthetic images</label>
        <div class="flex gap-1">
          <input v-model="uploadId" placeholder="uploadId" class="flex-1 bg-gray-800 rounded px-1 py-1" />
          <input v-model.number="imageCount" type="number" class="w-12 bg-gray-800 rounded px-1 py-1" />
          <button class="btn" @click="run(() => dev.images({ uploadId, count: imageCount }))">add</button>
        </div>
      </div>

      <!-- Trigger infer -->
      <div class="flex gap-1">
        <input v-model="imageId" placeholder="imageId" class="flex-1 bg-gray-800 rounded px-1 py-1" />
        <button class="btn" @click="run(() => dev.infer(imageId))">infer</button>
      </div>

      <!-- Freeze clock -->
      <div class="space-y-1">
        <label class="block text-gray-400">Clock</label>
        <div class="flex gap-1">
          <input v-model="clockAt" type="datetime-local" class="flex-1 bg-gray-800 rounded px-1 py-1" />
          <button class="btn" @click="run(() => dev.clock({ at: new Date(clockAt).toISOString() }))">freeze</button>
          <button class="btn" @click="run(() => dev.clock({ reset: true }))">live</button>
        </div>
      </div>

      <!-- Maintenance -->
      <div class="grid grid-cols-2 gap-1">
        <button class="btn" @click="run(() => dev.syncTags())">sync-tags</button>
        <button class="btn" @click="run(() => dev.reseed())">reseed</button>
        <button class="btn" @click="run(() => dev.apiKey())">api-key</button>
        <button class="btn" @click="run(() => dev.defaultTags(cameraId))">default-tags</button>
      </div>

      <pre v-if="result" class="bg-black/50 rounded p-2 whitespace-pre-wrap break-all">{{ result }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import * as dev from "src/api/dev";

const open = ref(false);
const roles = ["admin", "user", "projectAdmin", "projectEditor", "projectViewer"];
const role = ref("admin");
const userId = ref("");
const cameraId = ref("");
const driftSeconds = ref(37);
const stale = ref(false);
const uploadId = ref("");
const imageCount = ref(3);
const imageId = ref("");
const clockAt = ref("");
const result = ref("");

async function run(fn: () => Promise<unknown>) {
  try {
    result.value = JSON.stringify(await fn(), null, 2);
  } catch (e) {
    result.value = `error: ${(e as Error).message}`;
  }
}
</script>

<style scoped>
.btn {
  @apply rounded bg-gray-700 px-2 py-1 hover:bg-gray-600;
}
</style>
