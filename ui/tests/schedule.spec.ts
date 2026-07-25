import { describe, expect, it } from "vitest";
import { assignLanes, calendarDays, dayPosition, isAssigned, itemsOnDay, occupancyStatus, startOfDay, ScheduleItemLike } from "src/util/schedule";

function item(id: string, start: string, end: string, cardinality = 1, assignees: string[] = []): ScheduleItemLike {
  return { id, start, end, cardinality, assignees: assignees.map((a) => ({ id: a })) };
}

describe("occupancyStatus", () => {
  it("maps counts to the four states", () => {
    expect(occupancyStatus(0, 3)).toBe("empty");
    expect(occupancyStatus(1, 3)).toBe("partial");
    expect(occupancyStatus(3, 3)).toBe("full");
    expect(occupancyStatus(4, 3)).toBe("over");
  });
  it("zero assignees is empty even for cardinality 0 leftovers", () => {
    expect(occupancyStatus(0, 1)).toBe("empty");
  });
});

describe("calendarDays", () => {
  const items = [item("a", "2026-08-11T09:00:00Z", "2026-08-11T11:00:00Z"), item("b", "2026-08-13T09:00:00Z", "2026-08-13T11:00:00Z")];

  it("uses the project period when set", () => {
    const days = calendarDays({ startAt: "2026-08-10T00:00:00Z", endAt: "2026-08-14T23:00:00Z" }, []);
    expect(days.length).toBeGreaterThanOrEqual(5); // TZ-dependent ±1 column
    expect(days[0].getTime()).toBe(startOfDay(new Date("2026-08-10T00:00:00Z")).getTime());
  });

  it("falls back to the item span without a period", () => {
    const days = calendarDays({}, items);
    expect(days[0].getTime()).toBe(startOfDay(new Date(items[0].start)).getTime());
    expect(days.length).toBe(3);
  });

  it("falls back to the current week (Mon-Sun) without items", () => {
    const days = calendarDays({}, [], new Date("2026-07-25T10:00:00")); // a Saturday
    expect(days.length).toBe(7);
    expect(days[0].getDay()).toBe(1); // Monday
    expect(days[6].getDay()).toBe(0); // Sunday
  });

  it("caps a runaway period at 31 columns", () => {
    const days = calendarDays({ startAt: "2026-01-01T00:00:00Z", endAt: "2027-01-01T00:00:00Z" }, []);
    expect(days.length).toBe(31);
  });

  it("ignores an inverted period and falls back", () => {
    const days = calendarDays({ startAt: "2026-08-14T00:00:00Z", endAt: "2026-08-10T00:00:00Z" }, items);
    expect(days.length).toBe(3);
  });
});

describe("itemsOnDay / dayPosition", () => {
  it("selects only intersecting items", () => {
    const a = item("a", "2026-08-11T09:00:00", "2026-08-11T11:00:00");
    const b = item("b", "2026-08-12T09:00:00", "2026-08-12T11:00:00");
    const day = startOfDay(new Date("2026-08-11T00:00:00"));
    expect(itemsOnDay([a, b], day).map((i) => i.id)).toEqual(["a"]);
  });

  it("positions an item on the fixed 24h axis", () => {
    const a = item("a", "2026-08-11T10:00:00", "2026-08-11T12:00:00");
    const day = startOfDay(new Date("2026-08-11T00:00:00"));
    const pos = dayPosition(a, day);
    expect(pos.topPct).toBeCloseTo((10 / 24) * 100, 5);
    expect(pos.heightPct).toBeCloseTo((2 / 24) * 100, 5);
  });

  it("clamps a multi-day item to the column", () => {
    const a = item("a", "2026-08-10T22:00:00", "2026-08-12T02:00:00");
    const day = startOfDay(new Date("2026-08-11T00:00:00"));
    const pos = dayPosition(a, day);
    expect(pos.topPct).toBe(0);
    expect(pos.heightPct).toBe(100);
  });
});

describe("assignLanes", () => {
  it("splits overlapping items into side-by-side lanes", () => {
    const a = item("a", "2026-08-11T09:00:00", "2026-08-11T12:00:00");
    const b = item("b", "2026-08-11T10:00:00", "2026-08-11T11:00:00");
    const c = item("c", "2026-08-11T13:00:00", "2026-08-11T14:00:00");
    const lanes = assignLanes([a, b, c]);
    expect(lanes.get("a")).toEqual({ lane: 0, lanes: 2 });
    expect(lanes.get("b")).toEqual({ lane: 1, lanes: 2 });
    expect(lanes.get("c")).toEqual({ lane: 0, lanes: 1 }); // separate cluster: full width
  });

  it("reuses a freed lane (boundary touch does not overlap)", () => {
    const a = item("a", "2026-08-11T09:00:00", "2026-08-11T10:00:00");
    const b = item("b", "2026-08-11T09:30:00", "2026-08-11T11:00:00");
    const c = item("c", "2026-08-11T10:00:00", "2026-08-11T10:30:00");
    const lanes = assignLanes([a, b, c]);
    expect(lanes.get("c")?.lane).toBe(0); // a's lane is free again
    expect(lanes.get("b")?.lane).toBe(1);
  });
});

describe("isAssigned", () => {
  const a = item("a", "2026-08-11T09:00:00", "2026-08-11T10:00:00", 2, ["u1"]);
  it("matches assignee ids", () => {
    expect(isAssigned(a, "u1")).toBe(true);
    expect(isAssigned(a, "u2")).toBe(false);
    expect(isAssigned(a, undefined)).toBe(false);
  });
});
