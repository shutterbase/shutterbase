import { describe, it, expect } from "vitest";
import {
  classifyImage,
  planDownload,
  isExcluded,
  targetSegments,
  downloadFileName,
  estimateEtaSeconds,
  formatBytes,
  formatDuration,
  DELTA_SAFETY_MARGIN_MS,
  extractDateFromTag,
  weekdaySegment,
  eventDayDate,
  reconcileLocalFiles,
} from "src/util/downloadRunner";
import { DownloadConfig, Image } from "src/types/api";

const config = {
  blacklistTagIds: ["internal"],
  blockedImageIds: ["blockedimg00001"],
  lastDownloadAt: "2026-07-28T12:00:00Z",
  deltaSubfolder: false,
  groupByDate: false,
} as unknown as DownloadConfig;

const image = {
  id: "img000000000001",
  computedFileName: "FSG26_001_max",
  imageTags: ["podium"],
  updatedAt: "2026-07-28T09:00:00Z",
} as unknown as Image;

describe("downloadFileName", () => {
  it("appends .jpg to the computed name", () => {
    expect(downloadFileName(image)).toBe("FSG26_001_max.jpg");
  });
});

describe("isExcluded", () => {
  it("passes an unfiltered image", () => {
    expect(isExcluded(image, config)).toBe(false);
  });
  it("excludes blocked image ids", () => {
    expect(isExcluded({ ...image, id: "blockedimg00001" }, config)).toBe(true);
  });
  it("excludes blocked file names", () => {
    expect(isExcluded(image, { ...config, blockedImageIds: ["FSG26_001_max"] })).toBe(true);
  });
  it("OR-excludes blacklisted tags", () => {
    expect(isExcluded({ ...image, imageTags: ["podium", "internal"] }, config)).toBe(true);
  });
  it("tolerates images without denormalized tags", () => {
    expect(isExcluded({ ...image, imageTags: undefined as unknown as string[] }, config)).toBe(false);
  });
  it("keeps unreviewed images unless reviewedOnly is set", () => {
    expect(isExcluded({ ...image, upload: { id: "up1", name: "u", state: "ready" } }, config)).toBe(false);
  });
  it("excludes everything but reviewed uploads when reviewedOnly is set", () => {
    const reviewedOnly = { ...config, reviewedOnly: true };
    expect(isExcluded({ ...image, upload: { id: "up1", name: "u", state: "reviewed" } }, reviewedOnly)).toBe(false);
    expect(isExcluded({ ...image, upload: { id: "up1", name: "u", state: "ready" } }, reviewedOnly)).toBe(true);
    expect(isExcluded(image, reviewedOnly)).toBe(true); // no upload in the payload -> fails closed
  });
});

describe("classifyImage", () => {
  const existing = new Set(["FSG26_001_max.jpg"]);

  it("marks missing files as new", () => {
    expect(classifyImage(image, config, new Set())).toBe("new");
  });
  it("marks existing untouched files as present", () => {
    expect(classifyImage(image, config, existing)).toBe("present");
  });
  it("marks files updated after lastDownloadAt minus the safety margin as changed", () => {
    expect(classifyImage({ ...image, updatedAt: "2026-07-28T12:30:00Z" }, config, existing)).toBe("changed");
    const margin = new Date(new Date(config.lastDownloadAt!).getTime() - DELTA_SAFETY_MARGIN_MS / 2).toISOString();
    expect(classifyImage({ ...image, updatedAt: margin }, config, existing)).toBe("changed");
  });
  it("marks blacklisted images as excluded before anything else", () => {
    expect(classifyImage({ ...image, imageTags: ["internal"] }, config, new Set())).toBe("excluded");
  });
  it("treats existing files as present when the config never completed a run", () => {
    expect(classifyImage(image, { ...config, lastDownloadAt: null }, existing)).toBe("present");
  });
});

describe("planDownload", () => {
  const images = [
    image, // present
    { ...image, id: "img2", computedFileName: "FSG26_002_max" }, // new
    { ...image, id: "img3", updatedAt: "2026-07-28T13:00:00Z" }, // changed (same file name as img1 — use distinct name)
    { ...image, id: "img4", computedFileName: "FSG26_004_max", imageTags: ["internal"] }, // excluded
  ] as Image[];
  // distinct file for the changed case
  images[2] = { ...images[2], computedFileName: "FSG26_003_max" } as Image;
  const existing = new Set(["FSG26_001_max.jpg", "FSG26_003_max.jpg"]);

  it("counts all four statuses", () => {
    const plan = planDownload(images, config, existing, { delta: true });
    expect(plan.counts).toEqual({ present: 1, new: 1, changed: 1, excluded: 1 });
  });
  it("delta wants only new + changed", () => {
    const plan = planDownload(images, config, existing, { delta: true });
    expect(plan.wanted.map((i) => i.id)).toEqual(["img2", "img3"]);
  });
  it("full wants everything except excluded", () => {
    const plan = planDownload(images, config, existing, { delta: false });
    expect(plan.wanted.map((i) => i.id)).toEqual([image.id, "img2", "img3"]);
  });
  it("statuses map is keyed by image id", () => {
    const plan = planDownload(images, config, existing, { delta: true });
    expect(plan.statuses.get("img4")).toBe("excluded");
    expect(plan.statuses.get("img2")).toBe("new");
  });
});

describe("estimateEtaSeconds", () => {
  it("computes remaining seconds from the average rate", () => {
    // 100 MB done in 10s -> 10 MB/s; 300 MB remaining -> 30s
    expect(estimateEtaSeconds(100e6, 400e6, 10_000)).toBe(30);
  });
  it("returns null until there is signal", () => {
    expect(estimateEtaSeconds(0, 400e6, 10_000)).toBeNull(); // nothing moved
    expect(estimateEtaSeconds(1e6, 400e6, 1000)).toBeNull(); // too early
    expect(estimateEtaSeconds(400e6, 400e6, 10_000)).toBeNull(); // finished
    expect(estimateEtaSeconds(1e6, 0, 10_000)).toBeNull(); // unknown total
  });
});

describe("formatters", () => {
  it("formatBytes picks a sensible unit", () => {
    expect(formatBytes(512_000)).toBe("500 kB");
    expect(formatBytes(8_400_000)).toBe("8.0 MB");
    expect(formatBytes(2_500_000_000)).toBe("2.33 GB");
  });
  it("formatDuration scales from seconds to hours", () => {
    expect(formatDuration(42)).toBe("42s");
    expect(formatDuration(130)).toBe("2m 10s");
    expect(formatDuration(3790)).toBe("1h 3m");
  });
});

describe("targetSegments", () => {
  const runDate = new Date("2026-07-29T08:00:00Z");
  const captured = { capturedAtCorrected: "2026-07-27T14:30:00Z", capturedAt: "2026-07-27T14:00:00Z" };

  it("is flat by default", () => {
    expect(targetSegments(captured, config, { delta: false, runDate })).toEqual([]);
  });
  it("prefixes the capture-date folder with delta_/new_ only on delta runs", () => {
    const cfg = { ...config, deltaSubfolder: true };
    expect(targetSegments(captured, cfg, { delta: true, runDate }, "changed")).toEqual(["delta_2026-07-27"]);
    expect(targetSegments(captured, cfg, { delta: true, runDate }, "new")).toEqual(["new_2026-07-27"]);
    expect(targetSegments(captured, cfg, { delta: false, runDate }, "new")).toEqual([]);
  });
  it("groups by corrected capture date, falling back to capturedAt, then the run date", () => {
    const cfg = { ...config, groupByDate: true };
    expect(targetSegments(captured, cfg, { delta: false, runDate })).toEqual(["2026-07-27"]);
    expect(targetSegments({ capturedAtCorrected: "", capturedAt: "2026-07-26T10:00:00Z" }, cfg, { delta: false, runDate })).toEqual(["2026-07-26"]);
    expect(targetSegments({ capturedAtCorrected: "", capturedAt: "" }, cfg, { delta: false, runDate })).toEqual(["2026-07-29"]);
  });
});

describe("event-day derivation", () => {
  const runDate = new Date("2026-07-29T08:00:00Z");
  // local-time constructors keep these assertions timezone-independent
  const localIso = (y: number, m: number, d: number, h: number) => new Date(y, m - 1, d, h, 30).toISOString();

  it("extractDateFromTag finds YYYYMMDD inside tag text", () => {
    const date = extractDateFromTag(["foo", "20260804", "bar"]);
    expect(date && [date.getFullYear(), date.getMonth() + 1, date.getDate()]).toEqual([2026, 8, 4]);
  });
  it("extractDateFromTag returns null when no tag carries a date", () => {
    expect(extractDateFromTag(["a", "b", "c"])).toBeNull();
    expect(extractDateFromTag(undefined)).toBeNull();
  });
  it("weekdaySegment renders the English weekday", () => {
    expect(weekdaySegment(new Date(2026, 7, 4, 12))).toBe("20260804 Tuesday");
  });
  it("the date tag is authoritative over the capture time", () => {
    const image = { capturedAtCorrected: localIso(2026, 7, 29, 12), capturedAt: "", tags: [{ tag: { name: "20260727" } }] };
    expect(weekdaySegment(eventDayDate(image as any, runDate))).toBe("20260727 Monday");
  });
  it("captures before 04:00 belong to the previous event day", () => {
    const night = { capturedAtCorrected: localIso(2026, 7, 28, 2), capturedAt: "", tags: [] };
    expect(weekdaySegment(eventDayDate(night as any, runDate))).toBe("20260727 Monday");
    const morning = { capturedAtCorrected: localIso(2026, 7, 28, 4), capturedAt: "", tags: [] };
    expect(weekdaySegment(eventDayDate(morning as any, runDate))).toBe("20260728 Tuesday");
  });
  it("falls back to the run date without any capture data", () => {
    const image = { capturedAtCorrected: "", capturedAt: "", tags: [] };
    expect(eventDayDate(image as any, runDate)).toBe(runDate);
  });
});

describe("targetSegments with folderStructure = weekday", () => {
  const runDate = new Date("2026-07-29T08:00:00Z");
  const cfg = { ...config, folderStructure: "weekday" } as unknown as DownloadConfig;
  const captured = { capturedAtCorrected: "2026-07-27T14:30:00Z", capturedAt: "2026-07-27T14:00:00Z", tags: [{ tag: { name: "20260727" } }] };

  it("uses the tag-derived weekday folder", () => {
    expect(targetSegments(captured as any, cfg, { delta: false, runDate })).toEqual(["20260727 Monday"]);
  });
  it("prefixes the weekday folder with new_/delta_ on delta runs", () => {
    const deltaCfg = { ...cfg, deltaSubfolder: true } as unknown as DownloadConfig;
    expect(targetSegments(captured as any, deltaCfg, { delta: true, runDate }, "new")).toEqual(["new_20260727 Monday"]);
    expect(targetSegments(captured as any, deltaCfg, { delta: true, runDate }, "changed")).toEqual(["delta_20260727 Monday"]);
  });
});

// Minimal in-memory FileSystemDirectoryHandle fake — just enough surface for
// collectExistingFileEntries / moveFileToFolder / reconcileLocalFiles.
class FakeFile {
  kind = "file" as const;
  content = "x";
  constructor(public name: string) {}
  async getFile() {
    const content = this.content;
    return {
      stream: () =>
        new ReadableStream({
          start(controller) {
            controller.enqueue(new TextEncoder().encode(content));
            controller.close();
          },
        }),
    } as unknown as File;
  }
  async createWritable() {
    const chunks: Uint8Array[] = [];
    return new WritableStream<Uint8Array>({
      write: (chunk) => void chunks.push(chunk),
      close: () => void (this.content = chunks.map((chunk) => new TextDecoder().decode(chunk)).join("")),
    });
  }
}

class FakeDir {
  kind = "directory" as const;
  children = new Map<string, FakeDir | FakeFile>();
  constructor(public name: string) {}
  async *entries() {
    yield* this.children.entries();
  }
  async getFileHandle(name: string, opts?: { create?: boolean }) {
    let child = this.children.get(name);
    if (!child && opts?.create) {
      child = new FakeFile(name);
      this.children.set(name, child);
    }
    if (!child || child.kind !== "file") throw new Error(`no file ${name}`);
    return child;
  }
  async getDirectoryHandle(name: string, opts?: { create?: boolean }) {
    let child = this.children.get(name);
    if (!child && opts?.create) {
      child = new FakeDir(name);
      this.children.set(name, child);
    }
    if (!child || child.kind !== "directory") throw new Error(`no directory ${name}`);
    return child;
  }
  async removeEntry(name: string) {
    this.children.delete(name);
  }
  addFile(name: string): FakeFile {
    const file = new FakeFile(name);
    this.children.set(name, file);
    return file;
  }
  addDir(name: string): FakeDir {
    const dir = new FakeDir(name);
    this.children.set(name, dir);
    return dir;
  }
}

describe("reconcileLocalFiles", () => {
  const catalog = [
    { ...image, id: "img1", computedFileName: "FSG26_001_max" },
    { ...image, id: "img2", computedFileName: "FSG26_002_max", imageTags: ["internal"] }, // blacklisted
  ] as Image[];

  function makeRoot(): FakeDir {
    const root = new FakeDir("root");
    root.addFile("FSG26_001_max.jpg"); // in catalog — stays
    root.addFile("FSG26_gone_max.jpg"); // left the catalog — moved to deleted/
    root.addDir("20260804 Tuesday").addFile("FSG26_002_max.jpg"); // blacklisted — moved to blacklist/, subpath kept
    root.addDir("deleted").addFile("previously-swept.jpg"); // target dirs are never scanned
    return root;
  }

  it("moves catalog-less files to deleted/ and blacklisted files to blacklist/, keeping subpaths", async () => {
    const root = makeRoot();
    const events: string[] = [];
    const result = await reconcileLocalFiles(root as unknown as FileSystemDirectoryHandle, config, catalog, (p) => events.push(p.phase));

    expect(result.deletedCount).toBe(1);
    expect(result.blacklistedCount).toBe(1);
    expect(result.movedFiles).toEqual([
      { basename: "FSG26_gone_max.jpg", reason: "deleted", fromPath: "FSG26_gone_max.jpg", toPath: "deleted/FSG26_gone_max.jpg" },
      { basename: "FSG26_002_max.jpg", reason: "blacklisted", fromPath: "20260804 Tuesday/FSG26_002_max.jpg", toPath: "blacklist/20260804 Tuesday/FSG26_002_max.jpg" },
    ]);
    expect(events[0]).toBe("scanning");
    expect(events[events.length - 1]).toBe("finished");

    // moved, not deleted — originals gone, copies present with content intact
    expect(root.children.has("FSG26_gone_max.jpg")).toBe(false);
    const deletedDir = root.children.get("deleted") as FakeDir;
    expect((deletedDir.children.get("FSG26_gone_max.jpg") as FakeFile).content).toBe("x");
    expect(deletedDir.children.has("previously-swept.jpg")).toBe(true);
    const blacklistDay = (root.children.get("blacklist") as FakeDir).children.get("20260804 Tuesday") as FakeDir;
    expect(blacklistDay.children.has("FSG26_002_max.jpg")).toBe(true);
    expect((root.children.get("20260804 Tuesday") as FakeDir).children.size).toBe(0);
    expect(root.children.has("FSG26_001_max.jpg")).toBe(true);
  });

  it("is idempotent — a second run moves nothing", async () => {
    const root = makeRoot();
    await reconcileLocalFiles(root as unknown as FileSystemDirectoryHandle, config, catalog);
    const second = await reconcileLocalFiles(root as unknown as FileSystemDirectoryHandle, config, catalog);
    expect(second.movedFiles).toEqual([]);
  });
});
