import { describe, expect, it } from "vitest";
import { aiBadgeLabel, aiBadgeTitle, aiUploadSummary, faceBoxStyle, faceCropStyle } from "src/util/aiDetection";

describe("aiBadgeLabel", () => {
  it("shows the queue position only while pending", () => {
    expect(aiBadgeLabel("pending", 12)).toBe("#12");
    expect(aiBadgeLabel("pending", 0)).toBe("");
    expect(aiBadgeLabel("processing", 12)).toBe("");
    expect(aiBadgeLabel("done")).toBe("");
    expect(aiBadgeLabel(null)).toBe("");
  });
});

describe("aiBadgeTitle", () => {
  it("describes each state", () => {
    expect(aiBadgeTitle("pending", 3)).toContain("position 3");
    expect(aiBadgeTitle("processing")).toContain("running");
    expect(aiBadgeTitle("done")).toContain("done");
    expect(aiBadgeTitle("error", undefined, "boom")).toContain("boom");
    expect(aiBadgeTitle(undefined)).toBe("");
  });
});

describe("faceBoxStyle", () => {
  it("maps relative coords to percentages", () => {
    expect(faceBoxStyle({ x: 0.1, y: 0.25, w: 0.5, h: 0.2 })).toEqual({
      left: "10.00%",
      top: "25.00%",
      width: "50.00%",
      height: "20.00%",
    });
  });
  it("clamps out-of-range values", () => {
    const s = faceBoxStyle({ x: -0.5, y: 1.5, w: 2, h: 0.5 });
    expect(s.left).toBe("0.00%");
    expect(s.top).toBe("100.00%");
    expect(s.width).toBe("100.00%");
  });
});

describe("faceCropStyle", () => {
  it("centers the face inside a margin-scaled square crop", () => {
    // 1000×500 image, 100×100px face centered at (500, 250), margin 2 → 200px crop at (400, 150)
    const s = faceCropStyle({ x: 0.45, y: 0.4, w: 0.1, h: 0.2 }, 1000, 500, 2)!;
    expect(s.img).toEqual({ width: "500.00%", left: "-200.00%", top: "-75.00%" });
    expect(s.box).toEqual({ left: "25.00%", top: "25.00%", width: "50.00%", height: "50.00%" });
  });
  it("clamps the crop into the image near edges", () => {
    // face at the top-left corner: crop cannot go negative
    const s = faceCropStyle({ x: 0, y: 0, w: 0.1, h: 0.2 }, 1000, 500, 2)!;
    expect(s.img.left).toBe("-0.00%");
    expect(s.img.top).toBe("-0.00%");
    expect(s.box.left).toBe("0.00%");
  });
  it("caps the crop at the image's shorter edge", () => {
    // huge face: margin-scaled square would exceed the image height
    const s = faceCropStyle({ x: 0.25, y: 0.1, w: 0.5, h: 0.8 }, 1000, 500, 2.75)!;
    expect(s.img.width).toBe("200.00%"); // crop side = 500 = image height
  });
  it("returns null without image dimensions", () => {
    expect(faceCropStyle({ x: 0.1, y: 0.1, w: 0.2, h: 0.2 }, 0, 0)).toBeNull();
  });
});

describe("aiUploadSummary", () => {
  it("summarizes the rollup", () => {
    expect(aiUploadSummary({ pending: 20, processing: 3, done: 12, error: 5, ahead: 213 })).toBe("12/40 done · 5 failed · 213 ahead");
    expect(aiUploadSummary({ pending: 0, processing: 0, done: 4, error: 0, ahead: 0 })).toBe("4/4 done");
    expect(aiUploadSummary({ pending: 0, processing: 0, done: 0, error: 0, ahead: 0 })).toBe("");
  });
});
