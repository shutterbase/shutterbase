import { describe, it, expect } from "vitest";
import { canEditTag, canAddImages, allowedTransitions, formatDuration, formatTaggingRate, isReviewErrorTag, isReviewRejectedTag, isReviewerOnlyTag, reviewVerdicts } from "src/util/uploadReview";

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

  it("reserves the review tags for the reviewer in every state", () => {
    for (const tagName of ["error", "rejected"]) {
      for (const uploadState of ["open", "ready", "reviewed"] as const) {
        expect(canEditTag({ ...base, uploadState, tagType: "custom", tagName })).toBe(false);
        expect(canEditTag({ ...base, uploadState, tagType: "custom", tagName, isReviewer: true })).toBe(true);
      }
    }
    expect(canEditTag({ ...base, uploadState: "open", tagType: "custom", tagName: "Rejected" })).toBe(false);
    expect(isReviewErrorTag("ERROR")).toBe(true);
    expect(isReviewRejectedTag("Rejected")).toBe(true);
    expect(isReviewerOnlyTag("rejects")).toBe(false);
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

describe("reviewVerdicts", () => {
  const tags = (...names: string[]) => names.map((name) => ({ tag: { name } }));

  it("flags the reserved verdict tags, case-insensitively", () => {
    expect(reviewVerdicts({ reviewEnabled: true, tags: tags("Race", "Rejected") })).toEqual({ rejected: true, error: false });
    expect(reviewVerdicts({ reviewEnabled: true, tags: tags("error") })).toEqual({ rejected: false, error: true });
    expect(reviewVerdicts({ reviewEnabled: true, tags: tags("error", "rejected") })).toEqual({ rejected: true, error: true });
  });

  it("stays silent without the review flow, or without tags", () => {
    expect(reviewVerdicts({ reviewEnabled: false, tags: tags("error", "rejected") })).toEqual({ rejected: false, error: false });
    expect(reviewVerdicts({ reviewEnabled: true })).toEqual({ rejected: false, error: false });
    expect(reviewVerdicts({ reviewEnabled: true, tags: [{ tag: null }] })).toEqual({ rejected: false, error: false });
  });
});
