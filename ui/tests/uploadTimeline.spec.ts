import { describe, expect, it } from "vitest";
import {
  EditorTrack,
  MIN_TRACK_MS,
  TimedImage,
  addScheduleTrack,
  addTagTrack,
  boundaryImages,
  clampWindow,
  expandTrack,
  hasScheduleOverlap,
  imagesInTrack,
  initialTracks,
  moveEdge,
  setEnabled,
  timelineWindow,
  toApiTracks,
} from "src/util/uploadTimeline";
import { ScheduleItem } from "src/types/api";

const H = 3_600_000;
const T0 = Date.parse("2026-08-11T08:00:00Z");

function track(key: string, start: number, end: number, opts: Partial<EditorTrack> = {}): EditorTrack {
  return { key, label: key, start, end, enabled: true, ...opts };
}
const sched = (key: string, start: number, end: number, opts: Partial<EditorTrack> = {}) => track(key, start, end, { scheduleItemId: key, ...opts });
const img = (id: string, time: number): TimedImage => ({ id, time });

function scheduleItem(id: string, start: number, end: number): ScheduleItem {
  return {
    id,
    title: id,
    description: "",
    start: new Date(start).toISOString(),
    end: new Date(end).toISOString(),
    cardinality: 1,
    assignees: [],
    tags: [],
    project: { id: "p" },
    createdAt: "",
    updatedAt: "",
  };
}

describe("timelineWindow", () => {
  it("spans images and tracks with padding", () => {
    const w = timelineWindow([img("a", T0), img("b", T0 + 2 * H)], [track("t", T0 - H, T0)]);
    expect(w.start).toBeLessThan(T0 - H);
    expect(w.end).toBeGreaterThan(T0 + 2 * H);
  });
  it("defaults to an hour around now when empty", () => {
    const w = timelineWindow([], [], T0);
    expect(w.end - w.start).toBe(H);
  });
});

describe("expand / clamp (transcript 22:11-22:31)", () => {
  const window = { start: T0, end: T0 + 10 * H };

  it("a lone schedule track expands to the full window", () => {
    const a = sched("a", T0 + 2 * H, T0 + 3 * H);
    const grown = expandTrack(a, [a], window);
    expect(grown.start).toBe(window.start);
    expect(grown.end).toBe(window.end);
  });

  it("expand stops at the neighboring enabled schedule track", () => {
    const a = sched("a", T0 + 1 * H, T0 + 2 * H);
    const b = sched("b", T0 + 4 * H, T0 + 5 * H);
    const grown = expandTrack(a, [a, b], window);
    expect(grown.start).toBe(window.start);
    expect(grown.end).toBe(b.start); // does not compromise b
  });

  it("a DISABLED schedule neighbor does not block expansion", () => {
    const a = sched("a", T0 + 1 * H, T0 + 2 * H);
    const b = sched("b", T0 + 4 * H, T0 + 5 * H, { enabled: false });
    expect(expandTrack(a, [a, b], window).end).toBe(window.end);
  });

  it("tag tracks ignore schedule neighbors entirely (stacking)", () => {
    const a = track("tag", T0 + 1 * H, T0 + 2 * H, { tagId: "x" });
    const b = sched("b", T0 + 4 * H, T0 + 5 * H);
    expect(expandTrack(a, [a, b], window).end).toBe(window.end);
    expect(clampWindow(a, [a, b], window)).toEqual(window);
  });
});

describe("moveEdge", () => {
  const window = { start: T0, end: T0 + 10 * H };

  it("clamps to the window and keeps the minimum length", () => {
    const a = sched("a", T0 + 2 * H, T0 + 3 * H);
    expect(moveEdge(a, "start", T0 - 5 * H, [a], window).start).toBe(window.start);
    expect(moveEdge(a, "start", T0 + 9 * H, [a], window).start).toBe(a.end - MIN_TRACK_MS);
    expect(moveEdge(a, "end", T0 + 20 * H, [a], window).end).toBe(window.end);
    expect(moveEdge(a, "end", T0, [a], window).end).toBe(a.start + MIN_TRACK_MS);
  });

  it("stops at an enabled schedule neighbor", () => {
    const a = sched("a", T0 + 1 * H, T0 + 2 * H);
    const b = sched("b", T0 + 4 * H, T0 + 5 * H);
    expect(moveEdge(a, "end", T0 + 6 * H, [a, b], window).end).toBe(b.start);
  });
});

describe("setEnabled", () => {
  it("refuses enabling into a schedule collision", () => {
    const a = sched("a", T0, T0 + 2 * H);
    const b = sched("b", T0 + H, T0 + 3 * H, { enabled: false });
    expect(setEnabled(b, true, [a, b])).toBeNull();
    expect(setEnabled(a, false, [a, b])).toEqual({ ...a, enabled: false });
  });
});

describe("images in track + boundaries (transcript 23:06)", () => {
  const images = [img("a", T0), img("b", T0 + H), img("c", T0 + 2 * H), img("d", T0 + 3 * H)];
  const t = track("t", T0 + H, T0 + 3 * H, { tagId: "x" });

  it("covers [start, end) like the server", () => {
    expect(imagesInTrack(images, t).map((i) => i.id)).toEqual(["b", "c"]);
  });

  it("previews before/first/last/after", () => {
    const b = boundaryImages(images, t);
    expect(b.before?.id).toBe("a");
    expect(b.first?.id).toBe("b");
    expect(b.last?.id).toBe("c");
    expect(b.after?.id).toBe("d");
  });
});

describe("initialTracks (transcript 21:37)", () => {
  const labels = { scheduleItem: (id: string) => `item-${id}`, tag: (id: string) => `tag-${id}` };
  const images = [img("a", T0 + H), img("b", T0 + 2 * H)];

  it("pre-populates my items intersecting the image span", () => {
    const mine = [scheduleItem("hit", T0, T0 + 90 * 60_000), scheduleItem("miss", T0 + 10 * H, T0 + 11 * H)];
    const tracks = initialTracks(undefined, mine, images, labels);
    expect(tracks.map((t) => t.scheduleItemId)).toEqual(["hit"]);
    expect(tracks[0].enabled).toBe(true);
  });

  it("persisted state wins and restores disabled lanes verbatim", () => {
    const persisted = [{ tagId: "x", start: new Date(T0).toISOString(), end: new Date(T0 + H).toISOString(), enabled: false }];
    const tracks = initialTracks(persisted, [scheduleItem("hit", T0, T0 + H)], images, labels);
    expect(tracks).toHaveLength(1);
    expect(tracks[0].tagId).toBe("x");
    expect(tracks[0].label).toBe("tag-x");
    expect(tracks[0].enabled).toBe(false);
  });
});

describe("adding tracks", () => {
  const window = { start: T0, end: T0 + 10 * H };
  const images = [img("a", T0 + H), img("b", T0 + 2 * H)];

  it("new tag lanes span the image range", () => {
    const [t] = addTagTrack([], "x", "Pits", images, window);
    expect(t.start).toBe(T0 + H);
    expect(t.end).toBe(T0 + 2 * H + MIN_TRACK_MS);
    expect(t.tagId).toBe("x");
  });

  it("a colliding schedule item arrives disabled; duplicates are ignored", () => {
    const existing = sched("a", T0, T0 + 3 * H);
    const added = addScheduleTrack([existing], scheduleItem("b", T0 + H, T0 + 2 * H));
    expect(added).toHaveLength(2);
    expect(added[1].enabled).toBe(false);
    expect(addScheduleTrack(added, scheduleItem("b", T0, T0 + H))).toHaveLength(2);
  });
});

describe("serialization + validation mirror", () => {
  it("round-trips to API tracks", () => {
    const [api] = toApiTracks([sched("a", T0, T0 + H, { enabled: false })]);
    expect(api).toEqual({ scheduleItemId: "a", tagId: undefined, start: new Date(T0).toISOString(), end: new Date(T0 + H).toISOString(), enabled: false });
  });

  it("detects enabled schedule overlaps like the server", () => {
    expect(hasScheduleOverlap([sched("a", T0, T0 + 2 * H), sched("b", T0 + H, T0 + 3 * H)])).toBe(true);
    expect(hasScheduleOverlap([sched("a", T0, T0 + H), sched("b", T0 + H, T0 + 2 * H)])).toBe(false); // touching is fine
    expect(hasScheduleOverlap([sched("a", T0, T0 + 2 * H), sched("b", T0 + H, T0 + 3 * H, { enabled: false })])).toBe(false);
  });
});
