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
  it("adds the per-run delta subfolder only on delta runs", () => {
    const cfg = { ...config, deltaSubfolder: true };
    expect(targetSegments(captured, cfg, { delta: true, runDate })).toEqual(["delta_2026-07-29"]);
    expect(targetSegments(captured, cfg, { delta: false, runDate })).toEqual([]);
  });
  it("groups by corrected capture date, falling back to capturedAt", () => {
    const cfg = { ...config, groupByDate: true };
    expect(targetSegments(captured, cfg, { delta: false, runDate })).toEqual(["2026-07-27"]);
    expect(targetSegments({ capturedAtCorrected: "", capturedAt: "2026-07-26T10:00:00Z" }, cfg, { delta: false, runDate })).toEqual(["2026-07-26"]);
    expect(targetSegments({ capturedAtCorrected: "", capturedAt: "" }, cfg, { delta: false, runDate })).toEqual(["unknown-date"]);
  });
  it("stacks delta subfolder before the date folder", () => {
    const cfg = { ...config, deltaSubfolder: true, groupByDate: true };
    expect(targetSegments(captured, cfg, { delta: true, runDate })).toEqual(["delta_2026-07-29", "2026-07-27"]);
  });
});
