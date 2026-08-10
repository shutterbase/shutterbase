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

      <!-- active run: one overall bar + one bar per parallel download -->
      <div v-if="run" class="mt-6 rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark" data-testid="download-run">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-primary-900 dark:text-white">{{ run.configName }} — {{ run.delta ? "delta" : "full" }} download</p>
            <p class="mt-0.5 text-xs tabular-nums text-primary-500 dark:text-primary-400">
              {{ run.progress.done }} / {{ run.progress.total }} images
              <span v-if="run.progress.bytesTotal"> · {{ formatBytes(overallBytes) }} / {{ formatBytes(run.progress.bytesTotal) }}</span>
              <span v-if="etaSeconds !== null" class="font-semibold text-primary-700 dark:text-primary-200"> · ~{{ formatDuration(etaSeconds) }} remaining</span>
              <span v-if="run.progress.skipped"> · {{ run.progress.skipped }} skipped</span>
              <span v-if="run.progress.failed.length" class="text-red-600 dark:text-red-400"> · {{ run.progress.failed.length }} failed</span>
            </p>
          </div>
          <button v-if="!run.finished" type="button" class="btn-secondary" @click="run.aborted = true">Cancel</button>
          <button v-else type="button" class="btn-secondary" @click="run = null">Dismiss</button>
        </div>
        <div class="mt-3 h-2 overflow-hidden rounded-full bg-primary-100 dark:bg-primary-800">
          <div class="h-full rounded-full bg-accent-500 transition-all" :style="{ width: `${overallPercent}%` }"></div>
        </div>
        <div v-if="!run.finished" class="mt-3 space-y-2">
          <div v-for="(worker, slot) in run.progress.workers" :key="slot" class="flex items-center gap-3">
            <span class="w-56 truncate font-mono text-[11px] text-primary-500 dark:text-primary-400">
              {{ worker ? worker.fileName : "—" }}
            </span>
            <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-primary-100 dark:bg-primary-800">
              <div
                v-if="worker"
                class="h-full rounded-full bg-accent-400 transition-all"
                :class="{ 'animate-pulse': !worker.total }"
                :style="{ width: worker.total ? `${Math.min(100, (worker.received / worker.total) * 100)}%` : '100%' }"
              ></div>
            </div>
            <span class="w-32 text-right font-mono text-[11px] tabular-nums text-primary-400">
              <template v-if="worker"
                >{{ formatBytes(worker.received) }}<template v-if="worker.total"> / {{ formatBytes(worker.total) }}</template></template
              >
            </span>
          </div>
        </div>
        <div v-if="run.finished && run.progress.failed.length" class="mt-3 text-xs text-red-600 dark:text-red-400">
          <p class="font-medium">Failed after {{ RETRY_COUNT }} retries (a delta run will pick them up again):</p>
          <p class="mt-1 max-h-24 overflow-y-auto font-mono">{{ run.progress.failed.join(", ") }}</p>
        </div>
      </div>

      <!-- preview: what a run would do, straight from the shared plan logic -->
      <div v-if="preview" class="mt-6 rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark" data-testid="download-preview">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-primary-900 dark:text-white">{{ preview.config.name }} — preview</p>
            <p class="mt-0.5 text-xs tabular-nums text-primary-500 dark:text-primary-400">
              {{ preview.images.length }} matching · {{ preview.plan.counts.present }} already in folder ·
              <span class="font-semibold text-accent-600 dark:text-accent-400">{{ preview.plan.counts.new + preview.plan.counts.changed }} to download</span>
              ({{ preview.plan.counts.new }} new, {{ preview.plan.counts.changed }} changed)
              <span v-if="preview.plan.counts.excluded"> · {{ preview.plan.counts.excluded }} excluded</span>
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="btn-primary"
              :disabled="!!activeRun || preview.plan.counts.new + preview.plan.counts.changed === 0"
              @click="startRun(preview.config, true)"
            >
              <ArrowPathIcon class="h-4 w-4" />
              Download new + changed ({{ preview.plan.counts.new + preview.plan.counts.changed }})
            </button>
            <button type="button" class="btn-secondary" :disabled="!!activeRun" @click="startRun(preview.config, false)">
              <ArrowDownTrayIcon class="h-4 w-4" />
              Full ({{ preview.images.length - preview.plan.counts.excluded }})
            </button>
            <button type="button" class="btn-secondary" @click="preview = null">Dismiss</button>
          </div>
        </div>
        <div class="mt-4 grid max-h-[28rem] grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] gap-1.5 overflow-y-auto">
          <div v-for="image in preview.images" :key="image.id" class="group relative aspect-[3/2] overflow-hidden rounded-md bg-primary-100 dark:bg-primary-800">
            <img
              :src="image.downloadUrls?.['256']"
              :alt="image.computedFileName"
              loading="lazy"
              class="h-full w-full object-cover"
              :class="{ 'opacity-30': preview.plan.statuses.get(image.id) === 'excluded' }"
            />
            <span class="absolute bottom-1 left-1 rounded px-1 py-px text-[10px] font-semibold" :class="statusBadgeClass(preview.plan.statuses.get(image.id))">
              {{ statusLabel(preview.plan.statuses.get(image.id)) }}
            </span>
          </div>
        </div>
      </div>
      <!-- sync progress -->
      <div v-if="syncProgress" class="mt-6 rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark" data-testid="sync-progress">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-primary-900 dark:text-white">Syncing — {{ syncProgress.configName }}</p>
            <p class="mt-0.5 truncate text-xs text-primary-500 dark:text-primary-400">
              <template v-if="syncProgress.phase === 'scanning'">Scanning local files…</template>
              <template v-else-if="syncProgress.fileName">{{ syncProgress.fileName }}</template>
              <template v-else>Processing files…</template>
            </p>
          </div>
          <span class="whitespace-nowrap text-xs tabular-nums text-primary-500 dark:text-primary-400">
            <template v-if="syncProgress.total">{{ syncProgress.current }} / {{ syncProgress.total }}</template>
          </span>
        </div>
        <div class="mt-3 h-2 overflow-hidden rounded-full bg-primary-100 dark:bg-primary-800">
          <div v-if="!syncProgress.total" class="h-full w-1/3 animate-pulse rounded-full bg-accent-500"></div>
          <div
            v-else
            class="h-full rounded-full bg-accent-500 transition-all duration-200"
            :style="{ width: `${Math.min(100, (syncProgress.current / syncProgress.total) * 100)}%` }"
          ></div>
        </div>
      </div>

      <!-- latest sync result -->
      <div v-if="syncResult" class="mt-6 rounded-lg border border-primary-200 bg-surface p-4 dark:border-primary-800 dark:bg-surface-dark" data-testid="sync-result">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-sm font-semibold text-primary-900 dark:text-white">Sync finished</p>
            <p class="mt-0.5 text-xs text-primary-500 dark:text-primary-400">{{ syncResult.configName }} · {{ formatDateTime(syncResult.syncedAt.toISOString()) }}</p>
          </div>
          <button type="button" class="btn-secondary" @click="syncResult = null">Dismiss</button>
        </div>
        <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-md border border-red-200 bg-red-500/10 px-3 py-2 dark:border-red-800">
            <p class="text-2xl font-semibold tabular-nums text-red-700 dark:text-red-300">{{ syncResult.result.deletedCount }}</p>
            <p class="text-xs text-red-700/80 dark:text-red-300/80">moved to <code>deleted/</code></p>
          </div>
          <div class="rounded-md border border-amber-200 bg-amber-500/10 px-3 py-2 dark:border-amber-800">
            <p class="text-2xl font-semibold tabular-nums text-amber-700 dark:text-amber-300">{{ syncResult.result.blacklistedCount }}</p>
            <p class="text-xs text-amber-700/80 dark:text-amber-300/80">moved to <code>blacklist/</code></p>
          </div>
          <div class="rounded-md border border-primary-200 bg-primary-50 px-3 py-2 dark:border-primary-700 dark:bg-primary-900/30">
            <p class="text-2xl font-semibold tabular-nums text-primary-700 dark:text-primary-200">{{ syncResult.result.movedFiles.length }}</p>
            <p class="text-xs text-primary-600 dark:text-primary-300">files moved in total</p>
          </div>
        </div>
        <div v-if="syncResult.result.movedFiles.length" class="mt-5">
          <p class="text-xs font-semibold uppercase tracking-wide text-primary-500 dark:text-primary-400">Moved files</p>
          <div class="mt-2 max-h-64 overflow-y-auto rounded-md border border-primary-200 dark:border-primary-700">
            <table class="min-w-full divide-y divide-primary-200 text-left text-xs dark:divide-primary-700">
              <thead class="sticky top-0 bg-primary-50 dark:bg-primary-900">
                <tr>
                  <th class="px-3 py-2 font-semibold text-primary-600 dark:text-primary-300">File</th>
                  <th class="px-3 py-2 font-semibold text-primary-600 dark:text-primary-300">Reason</th>
                  <th class="px-3 py-2 font-semibold text-primary-600 dark:text-primary-300">New location</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-primary-100 dark:divide-primary-800">
                <tr v-for="moved in syncResult.result.movedFiles" :key="moved.fromPath" class="text-primary-700 dark:text-primary-200">
                  <td class="max-w-[28rem] break-all px-3 py-2 font-mono">{{ moved.basename }}</td>
                  <td class="whitespace-nowrap px-3 py-2">{{ moved.reason }}</td>
                  <td class="max-w-[32rem] break-all px-3 py-2 font-mono">{{ moved.toPath }}</td>
                </tr>
              </tbody>
            </table>
          </div>
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
            <span class="font-semibold text-primary-700 dark:text-primary-200" data-testid="match-count">
              {{ matchCounts[config.id] !== undefined ? `${matchCounts[config.id]} matching images` : "counting…" }}
            </span>
            <span v-if="config.blockedImageIds.length"> · {{ config.blockedImageIds.length }} blocked</span>
            <span v-if="config.deltaSubfolder"> · delta subfolder</span>
            <span v-if="config.groupByDate"> · by date</span>
          </p>
          <p class="mt-1 flex items-center gap-1 text-xs text-primary-500 dark:text-primary-400">
            <FolderIcon class="h-3.5 w-3.5 flex-shrink-0" />
            <span class="truncate">{{ folderNames[config.id] ?? "no folder picked yet" }}</span>
            <button
              type="button"
              class="ml-auto flex-shrink-0 cursor-pointer rounded px-1.5 py-0.5 text-[11px] font-medium text-accent-600 transition-colors hover:bg-accent-500/10 dark:text-accent-400"
              @click="changeFolder(config)"
            >
              {{ folderNames[config.id] ? "change" : "pick" }}
            </button>
          </p>
          <p class="mt-1 text-xs text-primary-500 dark:text-primary-400">
            <span v-if="config.lastDownloadAt">last download {{ formatDateTime(config.lastDownloadAt) }}</span>
            <span v-else>never downloaded</span>
          </p>
          <div class="mt-3 flex gap-2 border-t border-primary-100 pt-3 dark:border-primary-800">
            <button type="button" class="btn-primary flex-1" :disabled="!supported || !!activeRun || previewLoading || !!syncProgress" @click="openPreview(config)">
              <EyeIcon class="h-4 w-4" />
              {{ previewLoading === config.id ? "Scanning…" : "Preview" }}
            </button>
            <button type="button" class="btn-secondary flex-1" :disabled="!supported || !!activeRun || !!syncProgress" @click="syncLocalFiles(config)">
              <ArrowPathIcon class="h-4 w-4" />
              Sync
            </button>
            <button type="button" class="btn-secondary flex-1" :disabled="!supported || !!activeRun || !!syncProgress" @click="startRun(config, true)">
              <ArrowPathIcon class="h-4 w-4" />
              Delta
            </button>
            <button type="button" class="btn-secondary flex-1" :disabled="!supported || !!activeRun || !!syncProgress" @click="startRun(config, false)">
              <ArrowDownTrayIcon class="h-4 w-4" />
              Full
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
import { ArrowDownTrayIcon, ArrowPathIcon, EyeIcon, FolderIcon, PencilIcon, PlusIcon } from "@heroicons/vue/24/outline";
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
import { tagLabel } from "src/util/tagOrder";
import { ensurePermission, getStoredDirHandle, storeDirHandle } from "src/util/dirHandleStore";
import {
  collectExistingFiles,
  DownloadPlan,
  estimateEtaSeconds,
  formatBytes,
  formatDuration,
  ImageStatus,
  isDirectoryPickerSupported,
  pickDirectory,
  planDownload,
  RETRY_COUNT,
  RunProgress,
  runDownload,
  reconcileLocalFiles,
  ReconcileProgress,
  ReconcileResult,
} from "src/util/downloadRunner";

interface SyncResultState {
  configName: string;
  result: ReconcileResult;
  syncedAt: Date;
}

const syncProgress = ref<({ configName: string } & ReconcileProgress) | null>(null);
const syncResult = ref<SyncResultState | null>(null);
const userStore = useUserStore();
const { activeProjectId } = storeToRefs(userStore);

const supported = isDirectoryPickerSupported();
const configs = ref<DownloadConfig[]>([]);
const projectTags = ref<ImageTag[]>([]);
const matchCounts = ref<Record<string, number>>({});
const folderNames = ref<Record<string, string>>({});
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
    void loadCardMeta(configList.items);
  } catch (error: any) {
    fail(error);
  }
}
onMounted(loadData);
watch(activeProjectId, loadData);

// Per-card metadata: how many images match the filter (limit=1, total only)
// and the name of the remembered target folder.
async function loadCardMeta(list: DownloadConfig[]) {
  await Promise.all(
    list.map(async (config) => {
      try {
        const page = await api.images.list({
          projectId: activeProjectId.value,
          tagId: config.whitelistTagIds.length ? config.whitelistTagIds : undefined,
          limit: 1,
        });
        matchCounts.value[config.id] = page.total;
      } catch {
        // count stays "counting…" — non-fatal
      }
      const handle = await getStoredDirHandle(config.id);
      if (handle) folderNames.value[config.id] = handle.name;
    }),
  );
}

function tagName(id: string): string {
  const tag = projectTags.value.find((t) => t.id === id);
  return tag ? tagLabel(tag) : id;
}
const formatDateTime = (iso: string) => DateTime.fromISO(iso).toLocaleString(DateTime.DATETIME_SHORT);
// Overall data progress = completed files + everything the workers have
// streamed so far, so the bar moves smoothly instead of jumping per file.
const overallBytes = computed(() => {
  const p = run.value?.progress;
  if (!p) return 0;
  return p.bytesDone + p.workers.reduce((sum, w) => sum + (w?.received ?? 0), 0);
});
const overallPercent = computed(() => {
  const p = run.value?.progress;
  if (!p) return 0;
  if (p.bytesTotal) return Math.min(100, (overallBytes.value / p.bytesTotal) * 100);
  return p.total ? (p.done / p.total) * 100 : 0;
});
const etaSeconds = computed(() => {
  const p = run.value?.progress;
  if (!p || run.value?.finished) return null;
  return estimateEtaSeconds(overallBytes.value, p.bytesTotal, Date.now() - p.startedAt);
});

const statusLabel = (s?: ImageStatus) => ({ new: "new", changed: "changed", present: "present", excluded: "excluded" })[s ?? "present"];
function statusBadgeClass(s?: ImageStatus): string {
  switch (s) {
    case "new":
      return "bg-accent-600 text-white";
    case "changed":
      return "bg-amber-500 text-white";
    case "excluded":
      return "bg-red-600/90 text-white";
    default:
      return "bg-primary-900/70 text-white";
  }
}

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
    preview.value = null; // filters may have changed — a stale preview lies
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

// ---- directory persistence ----
// The picked folder is remembered per config (IndexedDB) — runs reuse it
// silently; "change" on the card re-picks.
async function getDirectory(config: DownloadConfig, forcePick = false): Promise<FileSystemDirectoryHandle | null> {
  if (!forcePick) {
    const stored = await getStoredDirHandle(config.id);
    if (stored && (await ensurePermission(stored))) return stored;
  }
  try {
    const handle = await pickDirectory();
    await storeDirHandle(config.id, handle);
    folderNames.value[config.id] = handle.name;
    return handle;
  } catch {
    return null; // user dismissed the picker
  }
}

async function changeFolder(config: DownloadConfig) {
  await getDirectory(config, true);
}

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

// ---- preview ----
interface PreviewState {
  config: DownloadConfig;
  images: Image[];
  plan: DownloadPlan; // delta-mode plan: new/changed/present/excluded counts
  directory: FileSystemDirectoryHandle;
}
const preview = ref<PreviewState | null>(null);
const previewLoading = ref<string | null>(null);

async function openPreview(config: DownloadConfig) {
  previewLoading.value = config.id;
  try {
    const directory = await getDirectory(config);
    if (!directory) return;
    const [images, existing] = await Promise.all([fetchAllImages(config), collectExistingFiles(directory)]);
    preview.value = { config, images, plan: planDownload(images, config, existing, { delta: true }), directory };
  } catch (error: any) {
    fail(error);
  } finally {
    previewLoading.value = null;
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

async function startRun(config: DownloadConfig, delta: boolean) {
  const directory = preview.value?.config.id === config.id ? preview.value.directory : await getDirectory(config);
  if (!directory) return;
  const runStart = new Date();
  run.value = {
    configName: config.name,
    delta,
    progress: { total: 0, done: 0, failed: [], skipped: 0, bytesTotal: 0, bytesDone: 0, startedAt: Date.now(), workers: [] },
    finished: false,
    aborted: false,
  };
  // Work through the ref's reactive proxy — mutating the raw object the ref
  // was created from bypasses Vue's tracking and freezes the panel at 0/0
  // (the original bug this rewrite fixes).
  const state = run.value;
  preview.value = null; // the run invalidates the preview's counts
  try {
    // Re-plan at run time — the preview may be minutes old.
    const [images, existing] = await Promise.all([fetchAllImages(config), collectExistingFiles(directory)]);
    const plan = planDownload(images, config, existing, { delta });
    const result = await runDownload(
      plan,
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

// ---- sync local files ----
// Reconcile moves local files that left the catalog (or got blacklisted)
// aside — never deletes. See reconcileLocalFiles for the rules.
async function syncLocalFiles(config: DownloadConfig) {
  const directory = await getDirectory(config);
  if (!directory) return;
  syncResult.value = null;
  syncProgress.value = { configName: config.name, current: 0, total: 0, phase: "scanning" };
  try {
    const images = await fetchAllImages(config);
    const result = await reconcileLocalFiles(directory, config, images, (progress) => {
      syncProgress.value = { configName: config.name, ...progress };
    });
    syncResult.value = { configName: config.name, result, syncedAt: new Date() };
    const moved = result.deletedCount || result.blacklistedCount;
    showNotificationToast({
      headline: moved ? `Synced: ${result.deletedCount} deleted, ${result.blacklistedCount} blacklisted` : "No changes to sync",
      type: moved ? "success" : "info",
    });
  } catch (error: any) {
    fail(error);
  } finally {
    syncProgress.value = null;
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
