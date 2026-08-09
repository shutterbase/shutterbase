<template>
  <div class="mx-auto w-full max-w-screen-2xl">
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="min-w-0">
        <h1 class="display text-3xl text-primary-900 dark:text-white">EXIF Viewer</h1>
        <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
          Inspect all metadata of a local image. The file is read in memory only — nothing is uploaded to the photo catalog or stored.
        </p>
      </div>

      <!-- drop zone -->
      <div
        class="mt-6 flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed px-6 py-10 text-center transition-colors"
        :class="dragging ? 'border-accent-500 bg-accent-500/10' : 'border-primary-300 bg-surface hover:border-accent-500 dark:border-primary-700 dark:bg-surface-dark'"
        data-testid="exif-drop-zone"
        @click="fileInput?.click()"
        @dragover.prevent="dragging = true"
        @dragleave.prevent="dragging = false"
        @drop.prevent="onDrop"
      >
        <PhotoIcon class="h-10 w-10 text-primary-400" />
        <p class="mt-3 text-sm font-medium text-primary-900 dark:text-white">Drop an image here or click to browse</p>
        <p class="mt-1 text-xs text-primary-500 dark:text-primary-400">JPEG, RAW, PNG, TIFF … anything exiftool can read</p>
        <input ref="fileInput" type="file" class="hidden" @change="onPick" />
      </div>

      <div v-if="loading" class="mt-6 flex items-center gap-3 text-sm text-primary-500 dark:text-primary-400">
        <ArrowPathIcon class="h-5 w-5 animate-spin" />
        Reading metadata of {{ fileName }}…
      </div>

      <div v-if="error" class="mt-6 rounded-lg border border-red-300 bg-red-500/10 px-4 py-3 text-sm text-red-800 dark:border-red-700 dark:text-red-200">
        {{ error }}
      </div>

      <template v-if="meta && !loading">
        <!-- headline: preview + capture times -->
        <div class="mt-6 flex flex-wrap gap-6">
          <img v-if="previewUrl" :src="previewUrl" :alt="fileName" class="max-h-64 rounded-lg border border-primary-200 object-contain dark:border-primary-800" />
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-semibold text-primary-900 dark:text-white" data-testid="exif-file-name">{{ fileName }}</p>
            <div class="mt-3 grid gap-4 sm:grid-cols-2">
              <div class="rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark">
                <p class="text-xs font-medium uppercase tracking-wide text-primary-500 dark:text-primary-400">Original capture time</p>
                <p class="mt-1 text-lg font-semibold tabular-nums text-primary-900 dark:text-white">{{ facts.originalCaptureTime ?? "—" }}</p>
              </div>
              <div class="rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark">
                <p class="text-xs font-medium uppercase tracking-wide text-primary-500 dark:text-primary-400">Corrected capture time</p>
                <p class="mt-1 text-lg font-semibold tabular-nums text-primary-900 dark:text-white">{{ facts.correctedCaptureTime ?? "—" }}</p>
                <span
                  v-if="facts.timeShift"
                  class="mt-2 inline-flex rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-semibold tabular-nums text-amber-700 dark:text-amber-300"
                  data-testid="exif-time-shift"
                >
                  {{ facts.timeShift }} shift applied
                </span>
                <span v-else class="mt-2 inline-flex rounded-full bg-primary-500/10 px-2 py-0.5 text-xs text-primary-600 dark:text-primary-300">no shift</span>
              </div>
            </div>
          </div>
        </div>

        <!-- key characteristics -->
        <div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4" data-testid="exif-key-facts">
          <div v-for="fact in factCards" :key="fact.label" class="rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark">
            <div class="flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-primary-500 dark:text-primary-400">
              <component :is="fact.icon" class="h-4 w-4" />
              {{ fact.label }}
            </div>
            <p class="mt-1 truncate text-base font-semibold text-primary-900 dark:text-white" :title="fact.value ?? ''">{{ fact.value ?? "—" }}</p>
          </div>
        </div>

        <div v-if="facts.keywords.length" class="mt-4 flex flex-wrap gap-1.5">
          <span v-for="keyword in facts.keywords" :key="keyword" class="rounded-full bg-primary-500/10 px-2.5 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-200">
            {{ keyword }}
          </span>
        </div>

        <!-- all raw data, one collapsed group per exiftool family-1 group -->
        <h2 class="display mt-8 text-xl text-primary-900 dark:text-white">All metadata</h2>
        <div class="mb-10 mt-3 space-y-2" data-testid="exif-raw-groups">
          <details v-for="(group, groupName) in meta" :key="groupName" class="rounded-lg border border-primary-200 bg-surface dark:border-primary-800 dark:bg-surface-dark">
            <summary class="cursor-pointer select-none px-4 py-3 text-sm font-semibold text-primary-900 marker:text-primary-400 dark:text-white">
              {{ groupName }}
              <span class="ml-2 text-xs font-normal text-primary-500 dark:text-primary-400">{{ Object.keys(group).length }} tags</span>
            </summary>
            <div class="overflow-x-auto border-t border-primary-200 dark:border-primary-800">
              <table class="w-full text-left text-sm">
                <tbody>
                  <tr v-for="(value, tagName) in group" :key="tagName" class="border-b border-primary-100 last:border-b-0 dark:border-primary-800/50">
                    <td class="whitespace-nowrap px-4 py-1.5 align-top font-medium text-primary-700 dark:text-primary-200">{{ tagName }}</td>
                    <td class="break-all px-4 py-1.5 tabular-nums text-primary-900 dark:text-white">{{ formatRaw(value) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </details>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, type Component } from "vue";
import { ArrowPathIcon, CameraIcon, Squares2X2Icon, EyeIcon, BoltIcon, SunIcon, ViewfinderCircleIcon, PhotoIcon, UserIcon, IdentificationIcon } from "@heroicons/vue/24/outline";
import { api } from "src/api";
import { extractKeyFacts, type ExifGroups } from "src/util/exifInspect";

const fileInput = ref<HTMLInputElement | null>(null);
const dragging = ref(false);
const loading = ref(false);
const error = ref<string | null>(null);
const meta = ref<ExifGroups | null>(null);
const fileName = ref("");
const previewUrl = ref<string | null>(null);

const facts = computed(() => extractKeyFacts(meta.value ?? {}));

const factCards = computed((): { label: string; value?: string; icon: Component }[] => [
  { label: "Camera", value: facts.value.camera, icon: CameraIcon },
  { label: "Body serial", value: facts.value.bodySerial, icon: IdentificationIcon },
  { label: "Lens", value: facts.value.lens, icon: EyeIcon },
  { label: "Dimensions", value: facts.value.dimensions, icon: Squares2X2Icon },
  { label: "Shutter", value: facts.value.exposureTime, icon: BoltIcon },
  { label: "Aperture", value: facts.value.aperture ? `ƒ/${facts.value.aperture}` : undefined, icon: ViewfinderCircleIcon },
  { label: "ISO", value: facts.value.iso, icon: SunIcon },
  { label: "Focal length", value: facts.value.focalLength, icon: PhotoIcon },
  { label: "Artist", value: facts.value.artist, icon: UserIcon },
  { label: "Copyright", value: facts.value.copyright, icon: IdentificationIcon },
]);

function onPick(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (file) void inspect(file);
}

function onDrop(event: DragEvent) {
  dragging.value = false;
  const file = event.dataTransfer?.files?.[0];
  if (file) void inspect(file);
}

// latest-wins guard: dropping a second file mid-flight must never pair the
// first file's metadata with the second file's name/preview.
let inspectToken = 0;

async function inspect(file: File) {
  const token = ++inspectToken;
  loading.value = true;
  error.value = null;
  meta.value = null;
  fileName.value = file.name;
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
  // RAW files won't render in <img>; the browser just shows nothing — fine.
  previewUrl.value = URL.createObjectURL(file);
  try {
    const result = await api.exif.inspect(file);
    if (token !== inspectToken) return; // superseded by a newer drop
    meta.value = result;
  } catch (e: any) {
    if (token !== inspectToken) return;
    error.value = e?.response?.data?.message ?? "Could not read metadata from this file.";
  } finally {
    if (token === inspectToken) {
      loading.value = false;
      if (fileInput.value) fileInput.value.value = "";
    }
  }
}

function formatRaw(value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (Array.isArray(value)) return value.join(", ");
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

onBeforeUnmount(() => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
});
</script>
