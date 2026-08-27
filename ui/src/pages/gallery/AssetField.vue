<template>
  <div>
    <span class="mb-1 block text-xs font-medium uppercase tracking-wide text-primary-500 dark:text-primary-400">{{ label }}</span>
    <div class="flex items-center gap-3">
      <span class="min-w-0 flex-1 truncate font-mono text-xs text-primary-500" :title="value">{{ value || "—" }}</span>
      <label
        class="inline-flex cursor-pointer items-center rounded-md border border-primary-200 bg-surface px-3 py-1.5 text-xs font-medium text-primary-700 hover:border-primary-300 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200"
      >
        {{ busy ? "Uploading…" : "Upload" }}
        <input type="file" class="hidden" :accept="accept" :disabled="busy" @change="onFile" />
      </label>
      <button v-if="value" type="button" class="text-xs text-primary-400 hover:text-error-500" @click="emit('change', '')">Clear</button>
    </div>
    <p v-if="error" class="mt-1 text-xs text-error-600">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{ label: string; value: string; accept: string; upload: (file: File) => Promise<string> }>();
const emit = defineEmits<{ change: [string] }>();
const busy = ref(false);
const error = ref("");

async function onFile(ev: Event) {
  const file = (ev.target as HTMLInputElement).files?.[0];
  if (!file) return;
  busy.value = true;
  error.value = "";
  try {
    emit("change", await props.upload(file));
  } catch (e: any) {
    error.value = e?.message ?? "upload failed";
  } finally {
    busy.value = false;
    (ev.target as HTMLInputElement).value = "";
  }
}
</script>
