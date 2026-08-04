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
  stepEdgeByImages,
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
  it("is EXACTLY the first-to-last picture span — tracks never widen the axis", () => {
    const w = timelineWindow([img("a", T0), img("b", T0 + 2 * H)], [track("t", T0 - 5 * H, T0 + 9 * H)]);
    expect(w.start).toBe(T0);
    expect(w.end).toBe(T0 + 2 * H + MIN_TRACK_MS); // [start, end): the last image stays coverable
  });
  it("falls back to the track union without timed images", () => {
    const w = timelineWindow([], [track("t", T0, T0 + H), track("u", T0 + 2 * H, T0 + 3 * H)]);
    expect(w).toEqual({ start: T0, end: T0 + 3 * H });
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

describe("stepEdgeByImages — hotkeys step pictures, not minutes", () => {
  // four pictures an hour apart; the axis ends MIN_TRACK_MS past the last one
  const images = [img("a", T0), img("b", T0 + H), img("c", T0 + 2 * H), img("d", T0 + 3 * H)];
  const window = timelineWindow(images, []);
  const t = track("t", T0 + H, T0 + 3 * H, { tagId: "x" }); // covers b, c

  const step = (edge: "start" | "end", delta: number) => stepEdgeByImages(images, t, edge, delta, window);
  const covered = (start: number, end: number) => imagesInTrack(images, { ...t, start, end }).map((i) => i.id);

  it("the out-point takes in / drops exactly one picture per step", () => {
    expect(covered(t.start, step("end", 1))).toEqual(["b", "c", "d"]);
    expect(covered(t.start, step("end", -1))).toEqual(["b"]);
  });

  it("the in-point takes in / drops exactly one picture per step", () => {
    expect(covered(step("start", -1), t.end)).toEqual(["a", "b", "c"]);
    expect(covered(step("start", 1), t.end)).toEqual(["c"]);
  });

  it("stepping past the last picture parks at the axis end, keeping it covered", () => {
    expect(step("end", 10)).toBe(window.end);
    expect(covered(t.start, step("end", 10))).toEqual(["b", "c", "d"]);
  });

  it("clamps at the first picture and no-ops without images", () => {
    expect(step("start", -10)).toBe(T0);
    expect(stepEdgeByImages([], t, "end", 1, window)).toBe(t.end);
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

  it("restores persisted lanes verbatim, incl. disabled ones", () => {
    const persisted = [{ tagId: "x", start: new Date(T0).toISOString(), end: new Date(T0 + H).toISOString(), enabled: false }];
    const tracks = initialTracks(persisted, [], images, labels);
    expect(tracks).toHaveLength(1);
    expect(tracks[0].tagId).toBe("x");
    expect(tracks[0].label).toBe("tag-x");
    expect(tracks[0].enabled).toBe(false);
  });

  it("merges items assigned AFTER the timeline was applied, keeping edited lanes untouched", () => {
    // "old" was already applied and its out-point dragged back; "new" was
    // assigned in the schedule afterwards and must show up on its own.
    const persisted = [{ scheduleItemId: "old", start: new Date(T0).toISOString(), end: new Date(T0 + H).toISOString(), enabled: true }];
    const mine = [scheduleItem("old", T0, T0 + 3 * H), scheduleItem("new", T0 + 90 * 60_000, T0 + 2 * H)];
    const tracks = initialTracks(persisted, mine, images, labels);
    expect(tracks.map((t) => t.scheduleItemId)).toEqual(["old", "new"]);
    expect(tracks[0].end).toBe(T0 + H); // the dragged out-point survives
    expect(tracks[1].enabled).toBe(true);
  });

  it("a merged item overlapping an enabled lane arrives disabled", () => {
    const persisted = [{ scheduleItemId: "old", start: new Date(T0).toISOString(), end: new Date(T0 + 3 * H).toISOString(), enabled: true }];
    const tracks = initialTracks(persisted, [scheduleItem("new", T0 + H, T0 + 2 * H)], images, labels);
    expect(tracks[1].scheduleItemId).toBe("new");
    expect(tracks[1].enabled).toBe(false);
  });

  it("keeps the persisted timeline while no image carries a time yet", () => {
    const persisted = [{ tagId: "x", start: new Date(T0).toISOString(), end: new Date(T0 + H).toISOString(), enabled: true }];
    expect(initialTracks(persisted, [scheduleItem("hit", T0, T0 + H)], [], labels)).toHaveLength(1);
  });
});

describe("adding tracks", () => {
  const window = { start: T0, end: T0 + 10 * H };

  it("new tag lanes span the full axis", () => {
    const [t] = addTagTrack([], "x", "Pits", window);
    expect(t.start).toBe(window.start);
    expect(t.end).toBe(window.end);
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
