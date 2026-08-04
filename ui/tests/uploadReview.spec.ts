import { describe, it, expect } from "vitest";
import { canEditTag, canAddImages, allowedTransitions, formatDuration, formatTaggingRate, isReviewErrorTag } from "src/util/uploadReview";

// These rules must stay in lockstep with api/internal/authorization — the server
// is authoritative, so a divergence here means the UI offers 403s.
const base = { reviewEnabled: true, tagType: "manual", tagName: "Race", isReviewer: false, isEditor: true };

describe("canEditTag", () => {
  it("leaves tagging untouched while the review flow is off", () => {
    expect(canEditTag({ ...base, reviewEnabled: false, uploadState: "ready" })).toBe(true);
  });

  it("denies a projectViewer regardless of state", () => {
    expect(canEditTag({ ...base, isEditor: false, uploadState: "open" })).toBe(false);
  });

  it("freezes official tags once the upload is submitted, but not custom ones", () => {
    expect(canEditTag({ ...base, uploadState: "open" })).toBe(true);
    expect(canEditTag({ ...base, uploadState: "ready" })).toBe(false);
    expect(canEditTag({ ...base, uploadState: "reviewed" })).toBe(false);
    expect(canEditTag({ ...base, uploadState: "ready", tagType: "custom", tagName: "todo" })).toBe(true);
  });

  it("reserves the error tag for the reviewer in every state", () => {
    expect(canEditTag({ ...base, uploadState: "open", tagType: "custom", tagName: "error" })).toBe(false);
    expect(canEditTag({ ...base, uploadState: "open", tagType: "custom", tagName: "Error", isReviewer: true })).toBe(true);
    expect(isReviewErrorTag("ERROR")).toBe(true);
  });

  it("never freezes the reviewer", () => {
    expect(canEditTag({ ...base, uploadState: "reviewed", isReviewer: true })).toBe(true);
  });
});

describe("allowedTransitions", () => {
  it("lets the photographer submit and nothing else", () => {
    expect(allowedTransitions("open", { isReviewer: false, isOwner: true })).toEqual(["ready"]);
    expect(allowedTransitions("ready", { isReviewer: false, isOwner: true })).toEqual([]);
    expect(allowedTransitions("open", { isReviewer: false, isOwner: false })).toEqual([]);
  });

  it("lets the reviewer move to any other state", () => {
    expect(allowedTransitions("ready", { isReviewer: true, isOwner: false })).toEqual(["open", "reviewed"]);
    expect(allowedTransitions("reviewed", { isReviewer: true, isOwner: false })).toEqual(["open", "ready"]);
  });
});

describe("canAddImages", () => {
  it("closes a submitted upload to the photographer only", () => {
    expect(canAddImages({ reviewEnabled: true, uploadState: "open", isReviewer: false })).toBe(true);
    expect(canAddImages({ reviewEnabled: true, uploadState: "ready", isReviewer: false })).toBe(false);
    expect(canAddImages({ reviewEnabled: true, uploadState: "reviewed", isReviewer: false })).toBe(false);
    expect(canAddImages({ reviewEnabled: true, uploadState: "ready", isReviewer: true })).toBe(true);
    expect(canAddImages({ reviewEnabled: false, uploadState: "reviewed", isReviewer: false })).toBe(true);
  });
});

describe("metric formatting", () => {
  it("formats durations compactly", () => {
    expect(formatDuration(0)).toBe("0s");
    expect(formatDuration(45)).toBe("45s");
    expect(formatDuration(150)).toBe("2m 30s");
    expect(formatDuration(4320)).toBe("1h 12m");
  });

  it("pairs images per second with the readable per-minute rate", () => {
    expect(formatTaggingRate(0)).toBe("–");
    expect(formatTaggingRate(0.0833)).toBe("0.083 img/s (5.0/min)");
  });
});
