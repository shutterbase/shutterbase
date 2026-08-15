import { describe, it, expect } from "vitest";
import { StatDay, StatPhotographer } from "src/api/statistics";
import { buildAssignmentDaySeries, buildImageDaySeries, fillDayRange, foldPhotographers, peakDay, photographerLabel } from "src/util/statisticsView";

function photographer(id: string, imageCount: number): StatPhotographer {
  return { id, firstName: `F${id}`, lastName: `L${id}`, copyrightTag: id.toUpperCase(), imageCount };
}

function day(date: string, byUser: Record<string, number>): StatDay {
  const byHour = new Array(24).fill(0);
  const total = Object.values(byUser).reduce((a, b) => a + b, 0);
  byHour[12] = total;
  return { date, total, byUser, byHour };
}

describe("fillDayRange", () => {
  it("fills gaps between first and last day", () => {
    expect(fillDayRange(["2026-08-10", "2026-08-13"])).toEqual(["2026-08-10", "2026-08-11", "2026-08-12", "2026-08-13"]);
  });

  it("passes single days and empty input through", () => {
    expect(fillDayRange(["2026-08-10"])).toEqual(["2026-08-10"]);
    expect(fillDayRange([])).toEqual([]);
  });

  it("keeps ranges beyond 120 days sparse", () => {
    const filled = fillDayRange(["2026-01-01", "2026-08-01"]);
    expect(filled).toEqual(["2026-01-01", "2026-08-01"]);
  });
});

describe("foldPhotographers", () => {
  it("gives each photographer a fixed slot when they fit", () => {
    const series = foldPhotographers([photographer("a", 5), photographer("b", 3)]);
    expect(series).toHaveLength(2);
    expect(series[0].key).toBe("a");
    expect(series[0].color).not.toBe(series[1].color);
    expect(series.find((s) => s.key === "other")).toBeUndefined();
  });

  it("folds the tail into Other past the slot count", () => {
    const many = Array.from({ length: 8 }, (_, i) => photographer(`p${i}`, 10 - i));
    const series = foldPhotographers(many);
    expect(series).toHaveLength(7); // 6 slots + Other
    expect(series[6].key).toBe("other");
  });
});

describe("buildImageDaySeries", () => {
  const days = [day("2026-08-10", { a: 2, b: 1 }), day("2026-08-12", { a: 1 })];
  const series = foldPhotographers([photographer("a", 3), photographer("b", 1)]);

  it("unsplit: one segment per day, gap days empty", () => {
    const chart = buildImageDaySeries(days, series, false);
    expect(chart.map((d) => d.date)).toEqual(["2026-08-10", "2026-08-11", "2026-08-12"]);
    expect(chart[0].segments).toHaveLength(1);
    expect(chart[0].segments[0].value).toBe(3);
    expect(chart[1].segments).toHaveLength(0);
    expect(chart[1].total).toBe(0);
  });

  it("split: one segment per photographer in slot order", () => {
    const chart = buildImageDaySeries(days, series, true);
    expect(chart[0].segments.map((s) => [s.key, s.value])).toEqual([
      ["a", 2],
      ["b", 1],
    ]);
    expect(chart[2].segments.map((s) => s.key)).toEqual(["a"]);
  });

  it("split: users without a slot fold into Other", () => {
    const folded = foldPhotographers(Array.from({ length: 7 }, (_, i) => photographer(`p${i}`, 10 - i)));
    const chart = buildImageDaySeries([day("2026-08-10", { p0: 1, p6: 2 })], folded, true);
    const other = chart[0].segments.find((s) => s.key === "other");
    expect(other?.value).toBe(2);
  });
});

describe("buildAssignmentDaySeries", () => {
  it("splits manual and AI per day and fills gaps", () => {
    const chart = buildAssignmentDaySeries([
      { date: "2026-08-10", manual: 4, ai: 2 },
      { date: "2026-08-12", manual: 0, ai: 3 },
    ]);
    expect(chart).toHaveLength(3);
    expect(chart[0].segments.map((s) => [s.key, s.value])).toEqual([
      ["manual", 4],
      ["ai", 2],
    ]);
    expect(chart[1].segments).toHaveLength(0);
    expect(chart[2].segments.map((s) => s.key)).toEqual(["ai"]);
    expect(chart[2].total).toBe(3);
  });
});

describe("peakDay", () => {
  it("returns the busiest day and null on empty input", () => {
    expect(peakDay([day("2026-08-10", { a: 1 }), day("2026-08-11", { a: 5 })])?.date).toBe("2026-08-11");
    expect(peakDay([])).toBeNull();
  });
});

describe("photographerLabel", () => {
  it("prefers name, falls back to copyright tag", () => {
    expect(photographerLabel(photographer("a", 1))).toBe("Fa La");
    expect(photographerLabel({ id: "x", firstName: "", lastName: "", copyrightTag: "MAPA", imageCount: 0 })).toBe("MAPA");
  });
});
