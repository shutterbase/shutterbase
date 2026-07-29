import { describe, it, expect } from "vitest";
import { isExcluded, needsDownload, targetSegments, downloadFileName, DELTA_SAFETY_MARGIN_MS } from "src/util/downloadRunner";

const config = {
  blacklistTagIds: ["internal"],
  blockedImageIds: ["blockedimg00001"],
  lastDownloadAt: "2026-07-28T12:00:00Z",
  deltaSubfolder: false,
  groupByDate: false,
};

const image = { id: "img000000000001", computedFileName: "FSG26_001_max", imageTags: ["podium"], updatedAt: "2026-07-28T09:00:00Z" };

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

describe("needsDownload", () => {
  const existing = new Set(["FSG26_001_max.jpg"]);

  it("downloads everything on a full run", () => {
    expect(needsDownload(image, config, existing, { delta: false })).toBe(true);
  });
  it("downloads missing files on a delta run", () => {
    expect(needsDownload(image, config, new Set(), { delta: true })).toBe(true);
  });
  it("skips existing files older than the delta window", () => {
    expect(needsDownload(image, config, existing, { delta: true })).toBe(false);
  });
  it("re-downloads files updated after lastDownloadAt minus the safety margin", () => {
    const updated = { ...image, updatedAt: "2026-07-28T12:30:00Z" };
    expect(needsDownload(updated, config, existing, { delta: true })).toBe(true);
    const margin = new Date(new Date(config.lastDownloadAt).getTime() - DELTA_SAFETY_MARGIN_MS / 2).toISOString();
    expect(needsDownload({ ...image, updatedAt: margin }, config, existing, { delta: true })).toBe(true);
  });
  it("skips existing files when the config never completed a run", () => {
    expect(needsDownload(image, { lastDownloadAt: null }, existing, { delta: true })).toBe(false);
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
