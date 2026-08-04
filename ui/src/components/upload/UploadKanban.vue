<template>
  <div class="mt-7 grid gap-4 lg:grid-cols-3">
    <section
      v-for="state in UPLOAD_STATES"
      :key="state"
      :class="[
        'flex flex-col rounded-lg border bg-surface-muted/60 transition-colors dark:bg-surface-dark-muted/40',
        dropTarget === state ? 'border-accent-500 ring-1 ring-accent-500' : 'border-primary-200 dark:border-primary-800',
      ]"
      @dragover="onDragOver($event, state)"
      @dragleave="onDragLeave(state)"
      @drop="onDrop($event, state)"
    >
      <header class="flex items-baseline justify-between border-b border-primary-200 px-4 py-3 dark:border-primary-800">
        <div>
          <p class="label-mono text-primary-500 dark:text-primary-400">{{ UPLOAD_STATE_LABEL[state] }}</p>
          <p class="mt-0.5 text-xs text-primary-500 dark:text-primary-400">{{ UPLOAD_STATE_HINT[state] }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <span class="font-data text-sm text-primary-700 dark:text-primary-200">{{ column(state).length }}</span>
          <!-- A new upload can only ever start in "open", so the shortcut lives
               in that lane's header. -->
          <button
            v-if="canCreate && state === 'open'"
            type="button"
            title="New upload"
            @click="emit('create')"
            class="inline-flex h-7 w-7 cursor-pointer items-center justify-center rounded-md border border-primary-200 bg-surface text-primary-500 transition-colors hover:border-accent-500 hover:text-accent-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-400 dark:hover:border-accent-500 dark:hover:text-accent-400"
          >
            <PlusIcon class="h-4 w-4" />
            <span class="sr-only">New upload</span>
          </button>
        </div>
      </header>

      <div class="flex flex-1 flex-col gap-3 p-3">
        <div v-if="column(state).length === 0" class="px-1 py-6 text-center">
          <p class="text-sm text-primary-400 dark:text-primary-500">Nothing here</p>
          <button
            v-if="canCreate && state === 'open'"
            type="button"
            @click="emit('create')"
            class="mt-2 inline-flex cursor-pointer items-center gap-1.5 text-sm font-medium text-accent-600 hover:text-accent-500 dark:text-accent-400"
          >
            <PlusIcon class="h-4 w-4" />
            New upload
          </button>
        </div>

        <article
          v-for="upload in column(state)"
          :key="upload.id"
          :draggable="transitionsFor(upload).length > 0"
          @dragstart="onDragStart($event, upload)"
          @dragend="dragged = null"
          :class="[
            'rounded-md border border-primary-200 bg-surface p-3 shadow-panel transition-colors dark:border-primary-700 dark:bg-surface-dark dark:shadow-panel-dark',
            transitionsFor(upload).length > 0 ? 'cursor-grab active:cursor-grabbing' : '',
            dragged?.id === upload.id ? 'opacity-50' : '',
          ]"
        >
          <div class="flex items-center justify-between gap-2">
            <a
              href="#"
              @click.prevent="emit('open', upload)"
              class="block truncate font-medium text-primary-900 hover:text-accent-600 dark:text-white dark:hover:text-accent-400"
              >{{ upload.name }}</a
            >
            <router-link
              :to="{ name: 'images', query: { upload: upload.id } }"
              title="Show this upload's images"
              class="shrink-0 text-primary-400 transition-colors hover:text-accent-600 dark:text-primary-500 dark:hover:text-accent-400"
            >
              <PhotoIcon class="h-4 w-4" />
              <span class="sr-only">Show this upload's images</span>
            </router-link>
          </div>
          <p class="mt-0.5 truncate text-sm text-primary-500 dark:text-primary-400">{{ upload.user?.firstName }} {{ upload.user?.lastName }}</p>

          <dl class="mt-3 grid grid-cols-2 gap-x-3 gap-y-1.5">
            <div v-for="metric in metricsFor(upload)" :key="metric.label" :class="metric.wide ? 'col-span-2' : ''">
              <dt class="label-mono text-[0.6rem] text-primary-500 dark:text-primary-400">{{ metric.label }}</dt>
              <dd :class="['font-data text-xs', metric.alert ? 'text-error-600 dark:text-error-400' : 'text-primary-800 dark:text-primary-100']">{{ metric.value }}</dd>
            </div>
          </dl>

          <div v-if="transitionsFor(upload).length" class="mt-3 flex flex-wrap gap-2 border-t border-primary-100 pt-3 dark:border-primary-800/70">
            <button
              v-for="next in transitionsFor(upload)"
              :key="next"
              type="button"
              @click="emit('move', upload, next)"
              :class="[actionBase, next === 'open' ? actionQuiet : actionAccent]"
            >
              {{ TRANSITION_LABEL[next] }}
            </button>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Upload, UploadState } from "src/types/api";
import { PhotoIcon, PlusIcon } from "@heroicons/vue/24/outline";
import { UPLOAD_STATES, UPLOAD_STATE_LABEL, UPLOAD_STATE_HINT, TRANSITION_LABEL, allowedTransitions, formatDuration, formatTaggingRate } from "src/util/uploadReview";

interface Props {
  uploads: Upload[];
  isReviewer: boolean;
  currentUserId?: string;
  canCreate?: boolean;
}
const props = defineProps<Props>();

const emit = defineEmits<{
  move: [upload: Upload, state: UploadState];
  open: [upload: Upload];
  create: [];
}>();

const actionBase =
  "inline-flex items-center justify-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 cursor-pointer";
const actionAccent = "bg-accent-600 text-white hover:bg-accent-500 active:bg-accent-700";
const actionQuiet =
  "border border-primary-200 bg-surface text-primary-700 hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:text-white";

// Most recently touched on top. updatedAt moves on every rename, state change
// AND tagging action (the metric accumulators write the row), so an upload
// somebody is actively working floats up. Sorted here rather than relying on the
// fetch order so a card jumps to the top the moment it is moved, without a reload.
function column(state: UploadState): Upload[] {
  return props.uploads.filter((u) => (u.state ?? "open") === state).sort((a, b) => new Date(b.updatedAt ?? 0).getTime() - new Date(a.updatedAt ?? 0).getTime());
}

function transitionsFor(upload: Upload): UploadState[] {
  return allowedTransitions(upload.state ?? "open", {
    isReviewer: props.isReviewer,
    isOwner: upload.user?.id === props.currentUserId,
  });
}

type Metric = { label: string; value: string; wide?: boolean; alert?: boolean };

function metricsFor(upload: Upload): Metric[] {
  const m = upload.metrics;
  if (!m) return [];
  const out: Metric[] = [
    { label: "Images", value: `${m.imageCount}` },
    { label: "Tags / image", value: m.tagsPerImage ? m.tagsPerImage.toFixed(1) : "–" },
    { label: "Tagging time", value: formatDuration(m.taggingSeconds) },
    { label: "To ready", value: formatDuration(m.timeToReadySeconds) },
  ];
  if (m.imagesPerSecond) {
    out.push({ label: "Pace", value: formatTaggingRate(m.imagesPerSecond), wide: true });
  }
  if (m.reviewCycles > 1) {
    out.push({ label: "Review cycles", value: `${m.reviewCycles}` });
  }
  if (m.errorCount > 0) {
    out.push({ label: "Tagging errors", value: `${m.errorCount}`, alert: true });
  }
  const aiTotal = (m.aiDone ?? 0) + (m.aiInFlight ?? 0) + (m.aiError ?? 0);
  if (aiTotal > 0) {
    out.push({
      label: "AI detection",
      value: `${m.aiDone}/${aiTotal}${m.aiError ? ` (${m.aiError} failed)` : ""}`,
      alert: m.aiError > 0,
    });
  }
  return out;
}

// Native HTML5 drag & drop — no dependency. The per-card buttons remain the
// keyboard-accessible path; dragging is the enhancement.
const dragged = ref<Upload | null>(null);
const dropTarget = ref<UploadState | null>(null);

function onDragStart(event: DragEvent, upload: Upload) {
  dragged.value = upload;
  event.dataTransfer?.setData("text/plain", upload.id);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = "move";
}

function onDragOver(event: DragEvent, state: UploadState) {
  if (!dragged.value || !transitionsFor(dragged.value).includes(state)) return;
  event.preventDefault(); // only a legal transition accepts the drop
  dropTarget.value = state;
}

function onDragLeave(state: UploadState) {
  if (dropTarget.value === state) dropTarget.value = null;
}

function onDrop(event: DragEvent, state: UploadState) {
  event.preventDefault();
  dropTarget.value = null;
  const upload = dragged.value;
  dragged.value = null;
  if (upload && transitionsFor(upload).includes(state)) {
    emit("move", upload, state);
  }
}
</script>
