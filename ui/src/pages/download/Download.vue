<template>
  <div class="mx-auto w-full max-w-screen-2xl">
    <div class="px-4 pt-6 sm:px-6 lg:px-8">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="min-w-0">
          <h1 class="display text-3xl text-primary-900 dark:text-white">Download</h1>
          <p class="mt-2 text-sm text-primary-500 dark:text-primary-400">
            Bulk-download this project's photos straight into a local folder — filters, delta sync and blocklist live in your personal download configs.
          </p>
        </div>
        <button v-if="supported" type="button" class="btn-primary" data-testid="add-download-config" @click="openCreate">
          <PlusIcon class="h-4 w-4" />
          New config
        </button>
      </div>

      <!-- Chromium-only: showDirectoryPicker is unavailable elsewhere. -->
      <div v-if="!supported" class="mt-6 rounded-lg border border-amber-300 bg-amber-500/10 px-4 py-3 text-sm text-amber-800 dark:border-amber-700 dark:text-amber-200">
        Your browser cannot write into local folders. Bulk download needs the File System Access API — please use Chrome or Edge on desktop.
      </div>

      <!-- active run -->
      <div v-if="run" class="mt-6 rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark" data-testid="download-run">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-primary-900 dark:text-white">{{ run.configName }} — {{ run.delta ? "delta" : "full" }} download</p>
            <p class="mt-0.5 text-xs tabular-nums text-primary-500 dark:text-primary-400">
              {{ run.progress.done }} / {{ run.progress.total }} downloaded
              <span v-if="run.progress.skipped"> · {{ run.progress.skipped }} skipped</span>
              <span v-if="run.progress.failed.length" class="text-red-600 dark:text-red-400"> · {{ run.progress.failed.length }} failed</span>
            </p>
          </div>
          <button v-if="!run.finished" type="button" class="btn-secondary" @click="run.aborted = true">Cancel</button>
          <button v-else type="button" class="btn-secondary" @click="run = null">Dismiss</button>
        </div>
        <div class="mt-3 h-2 overflow-hidden rounded-full bg-primary-100 dark:bg-primary-800">
          <div
            class="h-full rounded-full bg-accent-500 transition-all"
            :style="{ width: run.progress.total ? `${(run.progress.done / run.progress.total) * 100}%` : '100%' }"
          ></div>
        </div>
        <div v-if="run.finished && run.progress.failed.length" class="mt-3 text-xs text-red-600 dark:text-red-400">
          <p class="font-medium">Failed after {{ RETRY_COUNT }} retries (a delta run will pick them up again):</p>
          <p class="mt-1 max-h-24 overflow-y-auto font-mono">{{ run.progress.failed.join(", ") }}</p>
        </div>
      </div>

      <!-- config cards -->
      <div v-if="configs.length" class="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="config in configs"
          :key="config.id"
          class="flex flex-col rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark"
          data-testid="download-config-card"
        >
          <div class="flex items-start justify-between gap-2">
            <p class="truncate text-sm font-semibold text-primary-900 dark:text-white">{{ config.name }}</p>
            <button
              type="button"
              class="flex h-7 w-7 flex-shrink-0 cursor-pointer items-center justify-center rounded-md text-primary-400 transition-colors hover:bg-primary-100 hover:text-primary-700 dark:hover:bg-primary-800 dark:hover:text-primary-200"
              aria-label="Edit config"
              @click="openEdit(config)"
            >
              <PencilIcon class="h-4 w-4" />
            </button>
          </div>
          <div class="mt-2 flex flex-wrap gap-1.5 text-xs">
            <span
              v-for="tagId in config.whitelistTagIds"
              :key="tagId"
              class="rounded-full border border-accent-500 bg-accent-500/15 px-2 py-0.5 text-accent-700 dark:text-accent-200"
            >
              {{ tagName(tagId) }}
            </span>
            <span
              v-for="tagId in config.blacklistTagIds"
              :key="tagId"
              class="rounded-full border border-red-400 bg-red-500/10 px-2 py-0.5 text-red-700 line-through dark:text-red-300"
            >
              {{ tagName(tagId) }}
            </span>
            <span v-if="!config.whitelistTagIds.length && !config.blacklistTagIds.length" class="text-primary-400">all photos</span>
          </div>
          <p class="mt-2 text-xs text-primary-500 dark:text-primary-400">
            <span v-if="config.blockedImageIds.length">{{ config.blockedImageIds.length }} blocked · </span>
            <span v-if="config.deltaSubfolder">delta subfolder · </span>
            <span v-if="config.groupByDate">by date · </span>
            <span v-if="config.lastDownloadAt">last download {{ formatDateTime(config.lastDownloadAt) }}</span>
            <span v-else>never downloaded</span>
          </p>
          <div class="mt-3 flex gap-2 border-t border-primary-100 pt-3 dark:border-primary-800">
            <button type="button" class="btn-primary flex-1" :disabled="!supported || !!activeRun" @click="startRun(config, false)">
              <ArrowDownTrayIcon class="h-4 w-4" />
              Full
            </button>
            <button type="button" class="btn-secondary flex-1" :disabled="!supported || !!activeRun" @click="startRun(config, true)">
              <ArrowPathIcon class="h-4 w-4" />
              Delta
            </button>
          </div>
        </div>
      </div>
      <p v-else-if="loaded" class="mt-10 text-center text-sm text-primary-400">No download configs yet — create one to get started.</p>
    </div>

    <DownloadConfigDialog
      :show="dialogOpen"
      :create="dialogCreate"
      :config="dialogConfig"
      :project-tags="projectTags"
      @closed="dialogOpen = false"
      @save="saveConfig"
      @deleted="deleteConfig"
    />
    <UnexpectedErrorMessage :show="showUnexpectedErrorMessage" :error="unexpectedError" @closed="showUnexpectedErrorMessage = false" />
  </div>
</template>

<script setup lang="ts">
import { ArrowDownTrayIcon, ArrowPathIcon, PencilIcon, PlusIcon } from "@heroicons/vue/24/outline";
import { DateTime } from "luxon";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref, watch } from "vue";
import { api } from "src/api";
import { showNotificationToast } from "src/boot/mitt";
import DownloadConfigDialog from "src/components/download/DownloadConfigDialog.vue";
import UnexpectedErrorMessage from "src/components/UnexpectedErrorMessage.vue";
import { DownloadConfigCreate, DownloadConfigUpdate } from "src/api/downloadConfigs";
import { useUserStore } from "src/stores/user-store";
import { DownloadConfig, Image, ImageTag } from "src/types/api";
import { isDirectoryPickerSupported, pickDirectory, runDownload, RunProgress, RETRY_COUNT } from "src/util/downloadRunner";

const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const supported = isDirectoryPickerSupported();
const configs = ref<DownloadConfig[]>([]);
const projectTags = ref<ImageTag[]>([]);
const loaded = ref(false);

const unexpectedError = ref<any>(null);
const showUnexpectedErrorMessage = ref(false);
const fail = (error: any) => {
  unexpectedError.value = error;
  showUnexpectedErrorMessage.value = true;
};

async function loadData() {
  if (!activeProjectId.value) return;
  try {
    const [configList, tagList] = await Promise.all([
      api.downloadConfigs.list(activeProjectId.value),
      api.imageTags.list({ projectId: activeProjectId.value, limit: 500, sort: "name", order: "asc" }),
    ]);
    configs.value = configList.items;
    projectTags.value = tagList.items;
    loaded.value = true;
  } catch (error: any) {
    fail(error);
  }
}
onMounted(loadData);
watch(activeProjectId, loadData);

function tagName(id: string): string {
  return projectTags.value.find((t) => t.id === id)?.name ?? id;
}
const formatDateTime = (iso: string) => DateTime.fromISO(iso).toLocaleString(DateTime.DATETIME_SHORT);

// ---- config CRUD ----
const dialogOpen = ref(false);
const dialogCreate = ref(true);
const dialogConfig = ref<DownloadConfig | null>(null);

function openCreate() {
  dialogCreate.value = true;
  dialogConfig.value = null;
  dialogOpen.value = true;
}
function openEdit(config: DownloadConfig) {
  dialogCreate.value = false;
  dialogConfig.value = config;
  dialogOpen.value = true;
}

async function saveConfig(payload: DownloadConfigCreate | DownloadConfigUpdate) {
  try {
    if (dialogCreate.value) {
      await api.downloadConfigs.create({ ...(payload as DownloadConfigCreate), projectId: activeProjectId.value });
      showNotificationToast({ headline: "Download config added", type: "success" });
    } else if (dialogConfig.value) {
      await api.downloadConfigs.update(dialogConfig.value.id, payload);
      showNotificationToast({ headline: "Download config saved", type: "success" });
    }
    dialogOpen.value = false;
    await loadData();
  } catch (error: any) {
    fail(error);
  }
}

async function deleteConfig() {
  if (!dialogConfig.value) return;
  try {
    await api.downloadConfigs.remove(dialogConfig.value.id);
    showNotificationToast({ headline: "Download config deleted", type: "success" });
    dialogOpen.value = false;
    await loadData();
  } catch (error: any) {
    fail(error);
  }
}

// ---- download run ----
interface RunState {
  configName: string;
  delta: boolean;
  progress: RunProgress;
  finished: boolean;
  aborted: boolean;
}
const run = ref<RunState | null>(null);
const activeRun = computed(() => run.value && !run.value.finished);

// fetchAllImages pages through /images with the config's AND whitelist —
// the same server-side filter the CLI used.
async function fetchAllImages(config: DownloadConfig): Promise<Image[]> {
  const pageSize = 500;
  const images: Image[] = [];
  for (let offset = 0; ; offset += pageSize) {
    const page = await api.images.list({
      projectId: activeProjectId.value,
      tagId: config.whitelistTagIds.length ? config.whitelistTagIds : undefined,
      limit: pageSize,
      offset,
      sort: "capturedAtCorrected",
      order: "asc",
    });
    images.push(...page.items);
    if (images.length >= page.total || page.items.length === 0) return images;
  }
}

async function startRun(config: DownloadConfig, delta: boolean) {
  let directory: FileSystemDirectoryHandle;
  try {
    directory = await pickDirectory();
  } catch {
    return; // user dismissed the picker
  }
  const runStart = new Date();
  const state: RunState = {
    configName: config.name,
    delta,
    progress: { total: 0, done: 0, failed: [], skipped: 0 },
    finished: false,
    aborted: false,
  };
  run.value = state;
  try {
    const images = await fetchAllImages(config);
    const result = await runDownload(
      images,
      config,
      directory,
      { delta, runDate: runStart },
      (p) => (state.progress = p),
      () => state.aborted,
    );
    state.finished = true;
    if (state.aborted) {
      showNotificationToast({ headline: "Download cancelled", type: "info" });
      return;
    }
    // Persist the delta anchor only for completed runs — an aborted run must
    // not shift the window past images it never fetched.
    await api.downloadConfigs.update(config.id, { lastDownloadAt: runStart.toISOString() });
    await loadData();
    showNotificationToast({
      headline: result.failed.length ? `Download finished — ${result.failed.length} images failed` : `Downloaded ${result.done - result.failed.length} images`,
      type: result.failed.length ? "error" : "success",
    });
  } catch (error: any) {
    state.finished = true;
    fail(error);
  }
}
</script>

<style scoped>
.btn-primary {
  @apply inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-offset-primary-950;
}
.btn-secondary {
  @apply inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white;
}
</style>
