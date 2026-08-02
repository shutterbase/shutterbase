import { describe, it, expect } from "vitest";
import { pageWindow } from "src/util/pagination";

describe("pageWindow (pager buttons with ellipsis gaps)", () => {
  it("lists every page while they fit", () => {
    expect(pageWindow(1, 1)).toEqual([1]);
    expect(pageWindow(3, 7)).toEqual([1, 2, 3, 4, 5, 6, 7]);
  });

  it("collapses the tail when current is near the start", () => {
    expect(pageWindow(1, 10)).toEqual([1, 2, "…", 10]);
    expect(pageWindow(3, 10)).toEqual([1, 2, 3, 4, "…", 10]);
  });

  it("collapses the head when current is near the end", () => {
    expect(pageWindow(10, 10)).toEqual([1, "…", 9, 10]);
    expect(pageWindow(8, 10)).toEqual([1, "…", 7, 8, 9, 10]);
  });

  it("collapses both sides around a middle page", () => {
    expect(pageWindow(5, 10)).toEqual([1, "…", 4, 5, 6, "…", 10]);
  });

  it("renders no buttons for zero pages", () => {
    expect(pageWindow(1, 0)).toEqual([]);
  });
});
