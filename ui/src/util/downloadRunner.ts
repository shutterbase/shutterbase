// Bulk download runner for the Download page — the in-browser successor of
// cmd/downloader. Pure planning logic (filtering, delta decisions, target
// paths, retry policy) is separated from the File System Access API glue so
// vitest covers it without a browser directory handle.
//
// Delta semantics match the CLI: an image is downloaded when its file is
// missing locally OR it was updated after (lastDownloadAt - 5min). The
// timestamp now lives on the server-side download config instead of a local
// .timestamp file. Failed fetches retry with a wait (coco's PR #38/#40); a
// failed stream is aborted, so no corrupt partial files ever land in the
// folder.
import { API_BASE } from "src/boot/axios";
import { Image, DownloadConfig } from "src/types/api";

export const DELTA_SAFETY_MARGIN_MS = 5 * 60 * 1000;
export const RETRY_COUNT = 3;
export const RETRY_WAIT_MS = 5000;
export const PARALLELISM = 4;

export interface RunOptions {
  delta: boolean;
  runDate: Date; // start of this run — delta folder name + next lastDownloadAt
}

export interface ReconcileProgress {
  current: number;
  total: number;
  fileName?: string;
  phase: "scanning" | "moving" | "finished";
}

export function downloadFileName(image: Pick<Image, "computedFileName">): string {
  return `${image.computedFileName}.jpg`;
}

// isExcluded: blacklist tags OR-exclude; blocked entries match the image id
// or its computed file name (users paste file names, like the CLI blocklist).
export function isExcluded(image: Pick<Image, "id" | "computedFileName" | "imageTags">, config: Pick<DownloadConfig, "blacklistTagIds" | "blockedImageIds">): boolean {
  if (config.blockedImageIds.includes(image.id) || config.blockedImageIds.includes(image.computedFileName)) return true;
  return config.blacklistTagIds.some((tagId) => (image.imageTags ?? []).includes(tagId));
}

// ImageStatus drives both the preview grid and the run plan:
//   excluded — blacklist tag or blocklist entry
//   new      — no local file yet, will be downloaded
//   changed  — local file exists but the image was updated after the delta
//              window start, will be re-downloaded
//   present  — local file exists and is up to date
export type ImageStatus = "excluded" | "new" | "changed" | "present";

export function classifyImage(
  image: Pick<Image, "id" | "computedFileName" | "imageTags" | "updatedAt">,
  config: Pick<DownloadConfig, "blacklistTagIds" | "blockedImageIds" | "lastDownloadAt">,
  existingFiles: Set<string>,
): ImageStatus {
  if (isExcluded(image, config)) return "excluded";
  if (!existingFiles.has(downloadFileName(image))) return "new";
  if (config.lastDownloadAt) {
    const windowStart = new Date(config.lastDownloadAt).getTime() - DELTA_SAFETY_MARGIN_MS;
    if (new Date(image.updatedAt).getTime() > windowStart) return "changed";
  }
  return "present";
}

// Tag date extraction: finds YYYYMMDD inside tag text (adjust if tags differ)
const TAG_DATE_REGEX = /(\d{4})(\d{2})(\d{2})/;

export function extractDateFromTag(tags: string[] | undefined): Date | null {
  if (!tags) return null;
  for (const t of tags) {
    const match = t.match(TAG_DATE_REGEX);
    if (match) {
      const [, y, m, d] = match;
      const date = new Date(Number(y), Number(m) - 1, Number(d));
      if (!isNaN(date.getTime())) return date;
    }
  }
  return null;
}

export function weekdaySegment(date: Date, locale: string = "en-US"): string {
  const yyyy = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, "0");
  const dd = String(date.getDate()).padStart(2, "0");
  const weekday = date.toLocaleDateString(locale, { weekday: "long" });
  return `${yyyy}${mm}${dd} ${weekday}`;
}

// New richer file-collection helpers (do not remove the existing collectExistingFiles)
export interface ExistingFileEntry {
  basename: string;
  parentDirHandle: FileSystemDirectoryHandle;
  relativeSegments: string[]; // path from root excluding filename
}

export async function collectExistingFileEntries(
  root: FileSystemDirectoryHandle,
  ignoreDirNames: string[] = [
    "deleted",
    "blacklist",
    "_deleted",
    "_blacklist",
  ],
): Promise<ExistingFileEntry[]> {
  const results: ExistingFileEntry[] = [];
  async function walk(dir: FileSystemDirectoryHandle, segments: string[]) {
    for await (const [name, handle] of (dir as any).entries()) {
      if (handle.kind === "directory") {
        if (ignoreDirNames.includes(name)) continue;
        await walk(handle, [...segments, name]);
      } else if (handle.kind === "file") {
        results.push({ basename: name, parentDirHandle: dir, relativeSegments: segments });
      }
    }
  }
  await walk(root, []);
  return results;
}

export async function moveFileToFolder(
  root: FileSystemDirectoryHandle,
  entry: ExistingFileEntry,
  targetTopLevelSegment: "deleted" | "blacklist",
): Promise<string> {
  const targetSegments = [
    targetTopLevelSegment,
    ...entry.relativeSegments,
  ];

  const targetDir = await ensureDirectory(root, targetSegments);

  const fileHandle = await entry.parentDirHandle.getFileHandle(
    entry.basename,
  );

  const file = await fileHandle.getFile();

  const destHandle = await targetDir.getFileHandle(entry.basename, {
    create: true,
  });

  const writable = await destHandle.createWritable();

  try {
    await writable.write(await file.arrayBuffer());
    await writable.close();
  } catch (error) {
    await writable.abort().catch(() => undefined);
    throw error;
  }

  await entry.parentDirHandle.removeEntry(entry.basename);

  return [
    ...targetSegments,
    entry.basename,
  ].join("/");
}

export interface ReconcileResult {
  deletedCount: number;
  blacklistedCount: number;
  movedFiles: Array<{ basename: string; reason: "deleted" | "blacklisted"; fromPath: string; toPath: string }>;
}

export async function reconcileLocalFiles(
  directory: FileSystemDirectoryHandle,
  config: DownloadConfig,
  images: Image[],
  onProgress?: (progress: ReconcileProgress) => void,
): Promise<ReconcileResult> {
  onProgress?.({
    current: 0,
    total: 0,
    phase: "scanning",
  });

  const entries = await collectExistingFileEntries(directory);

  const currentByBasename = new Map(
    images.map((image) => [
      downloadFileName(image),
      image,
    ]),
  );

  const result: ReconcileResult = {
    deletedCount: 0,
    blacklistedCount: 0,
    movedFiles: [],
  };

  const total = entries.length;
  let current = 0;

  onProgress?.({
    current: 0,
    total,
    phase: "moving",
  });

  for (const entry of entries) {
    const fromPath = [
      ...entry.relativeSegments,
      entry.basename,
    ].join("/");

    onProgress?.({
      current,
      total,
      fileName: entry.basename,
      phase: "moving",
    });

    const image = currentByBasename.get(entry.basename);

if (!image) {
  const toPath = await moveFileToFolder(
    directory,
    entry,
    "deleted",
  );

  result.deletedCount++;

  result.movedFiles.push({
    basename: entry.basename,
    reason: "deleted",
    fromPath,
    toPath,
  });
} else if (isExcluded(image, config)) {
  const toPath = await moveFileToFolder(
    directory,
    entry,
    "blacklist",
  );

  result.blacklistedCount++;

  result.movedFiles.push({
    basename: entry.basename,
    reason: "blacklisted",
    fromPath,
    toPath,
  });
}

    current++;

    onProgress?.({
      current,
      total,
      fileName: entry.basename,
      phase: "moving",
    });
  }

  onProgress?.({
    current: total,
    total,
    phase: "finished",
  });

  return result;
}


export interface DownloadPlan {
  statuses: Map<string, ImageStatus>; // by image id
  counts: Record<ImageStatus, number>;
  // download order for the given mode: full = everything not excluded,
  // delta = new + changed only
  wanted: Image[];
}

// planDownload is the single source of truth shared by the preview panel and
// the actual run — what the preview shows is exactly what a run would do.
export function planDownload(images: Image[], config: DownloadConfig, existingFiles: Set<string>, options: Pick<RunOptions, "delta">): DownloadPlan {
  const statuses = new Map<string, ImageStatus>();
  const counts: Record<ImageStatus, number> = { excluded: 0, new: 0, changed: 0, present: 0 };
  const wanted: Image[] = [];
  for (const image of images) {
    const status = classifyImage(image, config, existingFiles);
    statuses.set(image.id, status);
    counts[status]++;
    if (status === "new" || status === "changed" || (!options.delta && status === "present")) {
      wanted.push(image);
    }
  }
  return { statuses, counts, wanted };
}

// targetSegments: folder path (without the file name) inside the picked
// directory. Delta runs may write into a per-run delta_<date> subfolder
// (issue #26); groupByDate sorts into capture-date folders (PR #40).
export function targetSegments(
  image: Pick<Image, "capturedAtCorrected" | "capturedAt">,
  config: Pick<DownloadConfig, "deltaSubfolder" | "groupByDate" | "folderStructure">,
  options: RunOptions,
  status?: ImageStatus,
): string[] {
  const segments: string[] = [];
  const captured = image.capturedAtCorrected || image.capturedAt;
  const capturedDate = captured ? isoDate(new Date(captured)) : isoDate(options.runDate);

  // Determine prefix based on status
  let prefix = "";
  if (options.delta && config.deltaSubfolder) {
    prefix = status === "new" ? "new_" : "delta_";  // ← New vs Changed
  }

  // Delta/Changed + weekday combined
  if (prefix && config.folderStructure === "weekday") {
    const tagDate = extractDateFromTag((image as any).tags?.map(t => typeof t === "string" ? t : t.tag?.name).filter(Boolean) as string[] | undefined)
      ?? new Date(image.capturedAtCorrected || image.capturedAt);
    segments.push(`${prefix}${weekdaySegment(tagDate, "en-US")}`);
  }
  // Delta/Changed only (no weekday)
  else if (prefix) {
    segments.push(`${prefix}${capturedDate}`);
  }
  // Weekday only (no delta/changed)
  else if (config.folderStructure === "weekday") {
    const tagDate = extractDateFromTag((image as any).tags?.map(t => typeof t === "string" ? t : t.tag?.name).filter(Boolean) as string[] | undefined)
      ?? new Date(image.capturedAtCorrected || image.capturedAt);
    segments.push(weekdaySegment(tagDate, "en-US"));
  }
  // Group by capture date only
  else if (config.groupByDate) {
    segments.push(capturedDate);
  }

  return segments;
}

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export interface FileProgress {
  fileName: string;
  received: number;
  total: number; // content-length, falling back to image.size; 0 = unknown
}

export interface RunProgress {
  total: number;
  done: number;
  failed: string[]; // computedFileNames that exhausted retries
  skipped: number; // images not part of this run (excluded / already present)
  bytesTotal: number; // sum of known image sizes in the plan (0 = unknown)
  bytesDone: number; // bytes of completed files
  startedAt: number; // epoch ms — ETA baseline
  workers: (FileProgress | null)[]; // per-parallel-slot file progress
}

// estimateEtaSeconds: simple average-rate ETA over the whole run. Returns null
// until there is enough signal (3s elapsed, some bytes moved) — a flickering
// wild guess is worse than none.
export function estimateEtaSeconds(bytesDone: number, bytesTotal: number, elapsedMs: number): number | null {
  if (bytesDone <= 0 || bytesTotal <= bytesDone || elapsedMs < 3000) return null;
  const bytesPerSecond = bytesDone / (elapsedMs / 1000);
  return Math.round((bytesTotal - bytesDone) / bytesPerSecond);
}

export function formatBytes(n: number): string {
  if (n >= 1073741824) return `${(n / 1073741824).toFixed(2)} GB`;
  if (n >= 1048576) return `${(n / 1048576).toFixed(1)} MB`;
  return `${Math.round(n / 1024)} kB`;
}

export function formatDuration(seconds: number): string {
  if (seconds >= 3600) return `${Math.floor(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
  if (seconds >= 60) return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`;
  return `${Math.max(1, Math.round(seconds))}s`;
}

// ---- File System Access API glue (Chromium desktop only) ----

export function isDirectoryPickerSupported(): boolean {
  return typeof (window as any).showDirectoryPicker === "function";
}

export async function pickDirectory(): Promise<FileSystemDirectoryHandle> {
  return (window as any).showDirectoryPicker({ mode: "readwrite" });
}

// collectExistingFiles walks the directory recursively and returns all file
// basenames. ponytail: basename collision across folders counts as "exists" —
// fine, computedFileName is unique per project.
export async function collectExistingFiles(dir: FileSystemDirectoryHandle): Promise<Set<string>> {
  const names = new Set<string>();
  const walk = async (handle: FileSystemDirectoryHandle): Promise<void> => {
    for await (const entry of (handle as any).values()) {
      if (entry.kind === "file") names.add(entry.name);
      else if (entry.kind === "directory") await walk(entry);
    }
  };
  await walk(dir);
  return names;
}

async function ensureDirectory(root: FileSystemDirectoryHandle, segments: string[]): Promise<FileSystemDirectoryHandle> {
  let dir = root;
  for (const segment of segments) {
    dir = await dir.getDirectoryHandle(segment, { create: true });
  }
  return dir;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

// downloadImageToDirectory streams one original into the target folder,
// retrying on failure and reporting byte progress. The writable is aborted on
// error, so a broken stream never leaves a partial file behind.
export async function downloadImageToDirectory(image: Image, root: FileSystemDirectoryHandle, segments: string[], onProgress?: (p: FileProgress) => void): Promise<void> {
  let lastError: unknown = null;
  for (let attempt = 0; attempt <= RETRY_COUNT; attempt++) {
    if (attempt > 0) await sleep(RETRY_WAIT_MS);
    try {
      const response = await fetch(`${API_BASE}/download/${image.id}/original`, { credentials: "include" });
      if (!response.ok || !response.body) {
        throw new Error(`download of ${image.computedFileName} failed: status ${response.status}`);
      }
      // content-length can be absent (chunked EXIF-injected stream) — fall
      // back to the catalog size so the file bar still has a denominator.
      const total = Number(response.headers.get("content-length") ?? 0) || image.size || 0;
      const dir = await ensureDirectory(root, segments);
      const fileHandle = await dir.getFileHandle(downloadFileName(image), { create: true });
      const writable = await fileHandle.createWritable();
      const reader = response.body.getReader();
      let received = 0;
      let lastEmitted = 0;
      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          await writable.write(value);
          received += value.byteLength;
          // throttle UI updates to every 256 kB — per-chunk emits at 4
          // parallel streams would re-render the panel hundreds of times/s
          if (received - lastEmitted >= 262144) {
            lastEmitted = received;
            onProgress?.({ fileName: image.computedFileName, received, total });
          }
        }
        await writable.close();
        onProgress?.({ fileName: image.computedFileName, received, total });
      } catch (streamError) {
        await writable.abort().catch(() => undefined);
        throw streamError;
      }
      return;
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError;
}

// runDownload executes a precomputed plan with a small worker pool. Returns
// the final progress; the caller persists lastDownloadAt only when the run
// completed (not aborted).
export async function runDownload(
  plan: DownloadPlan,
  config: DownloadConfig,
  root: FileSystemDirectoryHandle,
  options: RunOptions,
  onProgress: (p: RunProgress) => void,
  shouldAbort: () => boolean,
): Promise<RunProgress> {
  const progress: RunProgress = {
    total: plan.wanted.length,
    done: 0,
    failed: [],
    skipped: plan.statuses.size - plan.wanted.length,
    bytesTotal: plan.wanted.reduce((sum, image) => sum + (image.size || 0), 0),
    bytesDone: 0,
    startedAt: Date.now(),
    workers: Array.from({ length: PARALLELISM }, () => null),
  };
  onProgress({ ...progress, workers: [...progress.workers] });
  const emit = () => onProgress({ ...progress, workers: [...progress.workers] });

  const queue = [...plan.wanted];
  const worker = async (slot: number): Promise<void> => {
    for (;;) {
      const image = queue.shift();
      if (!image || shouldAbort()) {
        progress.workers[slot] = null;
        emit();
        return;
      }
      progress.workers[slot] = { fileName: image.computedFileName, received: 0, total: image.size || 0 };
      emit();
      let fileBytes = 0;
      try {
        // Status aus dem Plan holen
        const status = plan.statuses.get(image.id)!;  // ← HIER
        await downloadImageToDirectory(image, root, targetSegments(image, config, options, status), (p) => {  // ← status übergeben
          fileBytes = p.received;
          progress.workers[slot] = p;
          emit();
        });
        progress.bytesDone += fileBytes || image.size || 0;
      } catch {
        progress.failed.push(image.computedFileName);
      }
      progress.done++;
      progress.workers[slot] = null;
      emit();
    }
  };
  await Promise.all(Array.from({ length: PARALLELISM }, (_, slot) => worker(slot)));
  return progress;
}
