import { describe, it, expect } from "vitest";
import { kenBurnsVariant, nextSlideIndex, previousSlideIndex, preloadIndices, shouldFetchMore } from "src/util/slideshow";

describe("preloadIndices (decode-ahead window)", () => {
  it("returns the next headroom indices", () => {
    expect(preloadIndices(0, 20, 5)).toEqual([1, 2, 3, 4, 5]);
    expect(preloadIndices(10, 20, 3)).toEqual([11, 12, 13]);
  });

  it("clamps at the end of the loaded list", () => {
    expect(preloadIndices(18, 20, 5)).toEqual([19]);
    expect(preloadIndices(19, 20, 5)).toEqual([]);
  });

  it("is empty for an empty list", () => {
    expect(preloadIndices(0, 0)).toEqual([]);
  });
});

describe("shouldFetchMore (double headroom before the loaded end)", () => {
  it("requests the next page well before display catches up", () => {
    expect(shouldFetchMore(10, 20, 100, 5)).toBe(true);
    expect(shouldFetchMore(9, 20, 100, 5)).toBe(false);
  });

  it("never fetches once everything is loaded", () => {
    expect(shouldFetchMore(19, 20, 20, 5)).toBe(false);
  });
});

describe("nextSlideIndex / previousSlideIndex", () => {
  it("advances and wraps when looping", () => {
    expect(nextSlideIndex(0, 3, true)).toBe(1);
    expect(nextSlideIndex(2, 3, true)).toBe(0);
    expect(previousSlideIndex(0, 3, true)).toBe(2);
    expect(previousSlideIndex(1, 3, true)).toBe(0);
  });

  it("ends the show at the last slide when not looping", () => {
    expect(nextSlideIndex(2, 3, false)).toBeNull();
    expect(previousSlideIndex(0, 3, false)).toBe(0);
  });

  it("ends immediately on an empty list", () => {
    expect(nextSlideIndex(0, 0, true)).toBeNull();
  });
});

describe("kenBurnsVariant", () => {
  it("cycles deterministically through 4 variants", () => {
    expect([0, 1, 2, 3, 4, 5].map(kenBurnsVariant)).toEqual([0, 1, 2, 3, 0, 1]);
  });
});
