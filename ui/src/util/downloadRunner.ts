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

export function downloadFileName(image: Pick<Image, "computedFileName">): string {
  return `${image.computedFileName}.jpg`;
}

// isExcluded: blacklist tags OR-exclude; blocked entries match the image id
// or its computed file name (users paste file names, like the CLI blocklist).
export function isExcluded(image: Pick<Image, "id" | "computedFileName" | "imageTags">, config: Pick<DownloadConfig, "blacklistTagIds" | "blockedImageIds">): boolean {
  if (config.blockedImageIds.includes(image.id) || config.blockedImageIds.includes(image.computedFileName)) return true;
  return config.blacklistTagIds.some((tagId) => (image.imageTags ?? []).includes(tagId));
}

// needsDownload: CLI delta semantics. existingFiles holds basenames found
// anywhere under the target directory (recursive), so files from earlier
// full runs, date folders or delta subfolders all count as present.
export function needsDownload(
  image: Pick<Image, "computedFileName" | "updatedAt">,
  config: Pick<DownloadConfig, "lastDownloadAt">,
  existingFiles: Set<string>,
  options: Pick<RunOptions, "delta">,
): boolean {
  if (!options.delta) return true;
  if (!existingFiles.has(downloadFileName(image))) return true;
  if (!config.lastDownloadAt) return false;
  const windowStart = new Date(config.lastDownloadAt).getTime() - DELTA_SAFETY_MARGIN_MS;
  return new Date(image.updatedAt).getTime() > windowStart;
}

// targetSegments: folder path (without the file name) inside the picked
// directory. Delta runs may write into a per-run delta_<date> subfolder
// (issue #26); groupByDate sorts into capture-date folders (PR #40).
export function targetSegments(
  image: Pick<Image, "capturedAtCorrected" | "capturedAt">,
  config: Pick<DownloadConfig, "deltaSubfolder" | "groupByDate">,
  options: RunOptions,
): string[] {
  const segments: string[] = [];
  if (options.delta && config.deltaSubfolder) {
    segments.push(`delta_${isoDate(options.runDate)}`);
  }
  if (config.groupByDate) {
    const captured = image.capturedAtCorrected || image.capturedAt;
    segments.push(captured ? isoDate(new Date(captured)) : "unknown-date");
  }
  return segments;
}

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export interface RunProgress {
  total: number;
  done: number;
  failed: string[]; // computedFileNames that exhausted retries
  skipped: number; // images filtered out (delta up-to-date, blacklist, blocked)
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
// retrying on failure. The writable is aborted on error, so a broken stream
// never leaves a partial file behind.
export async function downloadImageToDirectory(image: Image, root: FileSystemDirectoryHandle, segments: string[]): Promise<void> {
  let lastError: unknown = null;
  for (let attempt = 0; attempt <= RETRY_COUNT; attempt++) {
    if (attempt > 0) await sleep(RETRY_WAIT_MS);
    try {
      const response = await fetch(`${API_BASE}/download/${image.id}/original`, { credentials: "include" });
      if (!response.ok || !response.body) {
        throw new Error(`download of ${image.computedFileName} failed: status ${response.status}`);
      }
      const dir = await ensureDirectory(root, segments);
      const fileHandle = await dir.getFileHandle(downloadFileName(image), { create: true });
      const writable = await fileHandle.createWritable();
      try {
        await response.body.pipeTo(writable);
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

// runDownload executes the plan with a small worker pool. Returns the final
// progress; the caller persists lastDownloadAt only when nothing failed.
export async function runDownload(
  images: Image[],
  config: DownloadConfig,
  root: FileSystemDirectoryHandle,
  options: RunOptions,
  onProgress: (p: RunProgress) => void,
  shouldAbort: () => boolean,
): Promise<RunProgress> {
  const existing = await collectExistingFiles(root);
  const wanted = images.filter((img) => !isExcluded(img, config) && needsDownload(img, config, existing, options));
  const progress: RunProgress = { total: wanted.length, done: 0, failed: [], skipped: images.length - wanted.length };
  onProgress({ ...progress });

  const queue = [...wanted];
  const worker = async (): Promise<void> => {
    for (;;) {
      const image = queue.shift();
      if (!image || shouldAbort()) return;
      try {
        await downloadImageToDirectory(image, root, targetSegments(image, config, options));
      } catch {
        progress.failed.push(image.computedFileName);
      }
      progress.done++;
      onProgress({ ...progress });
    }
  };
  await Promise.all(Array.from({ length: PARALLELISM }, () => worker()));
  return progress;
}
