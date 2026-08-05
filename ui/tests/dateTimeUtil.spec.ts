import { describe, it, expect } from "vitest";
import { DateTime } from "luxon";
import { appliedTimeOffset, timeOffsetUpToDate, toWasmTimeOffsets, backendTimeToUnixSeconds } from "src/util/dateTimeUtil";
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

// Regression: the seed writes offsets from time.Now(), Postgres keeps microseconds,
// so backend timestamps essentially always carry a fraction. BigInt() throws a
// RangeError on fractional input, which killed the whole browser upload pipeline
// (the throw surfaced as "resizing" never completing).
describe("toWasmTimeOffsets", () => {
  const fractional = offsetWithServerTime("2026-07-25T15:08:17.382Z");

  it("truncates sub-second precision instead of throwing", () => {
    expect(() => toWasmTimeOffsets([fractional])).not.toThrow();
    expect(toWasmTimeOffsets([fractional])[0].server_time).toBe(1784992097n);
  });

  it("keeps serverTime and cameraTime as whole-second bigints", () => {
    const [mapped] = toWasmTimeOffsets([fractional]);
    expect(typeof mapped.server_time).toBe("bigint");
    expect(typeof mapped.camera_time).toBe("bigint");
    expect(typeof mapped.time_offset).toBe("bigint");
  });

  it("survives a fractional timeOffset from the API", () => {
    const drifting = { ...offsetWithServerTime("2026-07-25T15:08:17.382Z"), timeOffset: 10.4 };
    expect(toWasmTimeOffsets([drifting])[0].time_offset).toBe(10n);
  });

  it("maps every offset it is given", () => {
    expect(toWasmTimeOffsets([fractional, fractional])).toHaveLength(2);
    expect(toWasmTimeOffsets([])).toEqual([]);
  });
});

describe("appliedTimeOffset (corrected minus original)", () => {
  const base = "2026-07-25T15:08:17Z";
  it("formats seconds, minutes and hours with a sign", () => {
    expect(appliedTimeOffset(base, "2026-07-25T15:08:22Z")).toBe("+5s");
    expect(appliedTimeOffset(base, "2026-07-25T15:07:32Z")).toBe("-45s");
    expect(appliedTimeOffset(base, "2026-07-25T15:10:20Z")).toBe("+2m 03s");
    expect(appliedTimeOffset(base, "2026-07-25T16:10:20Z")).toBe("+1h 02m 03s");
    expect(appliedTimeOffset(base, "2026-07-24T15:08:16Z")).toBe("-24h 00m 01s");
  });
  it("reports a zero offset neutrally and rounds sub-second drift", () => {
    expect(appliedTimeOffset(base, base)).toBe("±0s");
    expect(appliedTimeOffset(base, "2026-07-25T15:08:17.400Z")).toBe("±0s");
    expect(appliedTimeOffset(base, "2026-07-25T15:08:17.600Z")).toBe("+1s");
  });
});

describe("backendTimeToUnixSeconds", () => {
  it("floors to whole seconds", () => {
    expect(backendTimeToUnixSeconds("2026-07-25T15:08:17.382Z")).toBe(1784992097);
    expect(Number.isInteger(backendTimeToUnixSeconds("2026-07-25T15:08:17.999Z"))).toBe(true);
  });
});
