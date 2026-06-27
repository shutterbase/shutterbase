import { describe, it, expect } from "vitest";
import { DateTime } from "luxon";
import { timeOffsetUpToDate } from "src/util/dateTimeUtil";
import { TimeOffset } from "src/types/api";

function offsetWithServerTime(serverTime: string): TimeOffset {
  return {
    id: "x",
    serverTime,
    cameraTime: serverTime,
    timeOffset: 0,
    camera: { id: "c", name: "cam" },
    upToDate: false,
    createdAt: serverTime,
    updatedAt: serverTime,
  };
}

describe("timeOffsetUpToDate (24h window)", () => {
  it("is up to date when serverTime is within the last 24h", () => {
    const within = DateTime.now().minus({ hours: 1 }).toISO()!;
    expect(timeOffsetUpToDate(offsetWithServerTime(within))).toBe(true);
  });

  it("is up to date for a serverTime 23h ago", () => {
    const within = DateTime.now().minus({ hours: 23 }).toISO()!;
    expect(timeOffsetUpToDate(offsetWithServerTime(within))).toBe(true);
  });

  it("is NOT up to date when serverTime is older than 24h", () => {
    const stale = DateTime.now().minus({ hours: 25 }).toISO()!;
    expect(timeOffsetUpToDate(offsetWithServerTime(stale))).toBe(false);
  });
});
