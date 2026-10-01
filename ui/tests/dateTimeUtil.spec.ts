import { describe, it, expect } from "vitest";
import { DateTime } from "luxon";
import { appliedTimeOffset, timeOffsetUpToDate, toWasmTimeOffsets, backendTimeToUnixSeconds } from "src/util/dateTimeUtil";
import { isoToLocalInput, localInputToIso, localInputToIsoInclusive, isoToEndOfMinute } from "src/util/dateTimeUtil";
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

  // Pins > vs >= at the boundary. 23h59m is inside; the comparison is strictly
  // greater-than, so a few seconds either side of 24h decides it.
  it("treats the 24h mark itself as stale", () => {
    const just = DateTime.now().minus({ hours: 24 }).toISO()!;
    expect(timeOffsetUpToDate(offsetWithServerTime(just))).toBe(false);
    const justInside = DateTime.now().minus({ hours: 24 }).plus({ seconds: 30 }).toISO()!;
    expect(timeOffsetUpToDate(offsetWithServerTime(justInside))).toBe(true);
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
    // The exact value, not Number.isInteger: Math.round(0.999) is an integer
    // too, so an isInteger assertion passes for a rounding implementation.
    expect(backendTimeToUnixSeconds("2026-07-25T15:08:17.999Z")).toBe(1784992097);
    expect(backendTimeToUnixSeconds("2026-07-25T15:08:18.001Z")).toBe(1784992098);
  });
});

describe("datetime-local conversions", () => {
  it("round-trips ISO -> input value -> ISO at minute precision", () => {
    const iso = "2026-08-25T20:55:30.123Z";
    const input = isoToLocalInput(iso);
    expect(input).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    const back = new Date(localInputToIso(input)!);
    // seconds truncated, wall-clock minutes preserved
    expect(back.getTime()).toBe(new Date(iso).getTime() - 30_123);
  });

  it("maps null/empty to empty string and back to null", () => {
    expect(isoToLocalInput(null)).toBe("");
    expect(isoToLocalInput("")).toBe("");
    expect(isoToLocalInput("bogus")).toBe("");
    expect(localInputToIso(null)).toBeNull();
    expect(localInputToIso("")).toBeNull();
    expect(localInputToIso("bogus")).toBeNull();
  });

  // The backend applies the `to` bound as an inclusive LTE and both the input
  // and the slider carry minute precision — so an unwidened bound silently
  // dropped every photo captured inside the final minute of the range.
  it("widens the upper bound to the last millisecond of the minute", () => {
    const input = "2026-08-25T23:59";
    expect(localInputToIso(input)).toBe(new Date(input).toISOString());
    const inclusive = new Date(localInputToIsoInclusive(input)!);
    expect(inclusive.getSeconds()).toBe(59);
    expect(inclusive.getMilliseconds()).toBe(999);
    // and strictly later than the un-widened value it replaces
    expect(inclusive.getTime()).toBeGreaterThan(new Date(localInputToIso(input)!).getTime());
  });

  // The suite pins TZ=Europe/Berlin (package.json test script), so the
  // spring-forward gap is a fact about the runner, not a conditional. Without
  // the pin this case degraded to a tautology on a UTC runner and the
  // non-existent-local-time path was never exercised in CI at all.
  it("rejects a local time that does not exist (DST spring-forward gap)", () => {
    expect(Intl.DateTimeFormat().resolvedOptions().timeZone).toBe("Europe/Berlin");
    // 2026-03-29 02:00→03:00 is the spring-forward gap in Europe/Berlin
    expect(isoToLocalInput(new Date("2026-03-29T02:30").toISOString())).not.toBe("2026-03-29T02:30");
    expect(localInputToIso("2026-03-29T02:30")).toBeNull();
    expect(localInputToIsoInclusive("2026-03-29T02:30")).toBeNull();
    // an hour later, on the same day, is a perfectly ordinary local time
    expect(localInputToIso("2026-03-29T03:30")).not.toBeNull();
  });

  // The round-trip guard exists for the gap case ONLY. An exact comparison also
  // rejected these, and a null there deletes the bound — the range silently
  // opens to every photo, which is far worse than accepting a seconds-carrying
  // or date-only value.
  it("accepts values that are not byte-identical to isoToLocalInput", () => {
    const withSeconds = localInputToIso("2026-08-11T10:00:30");
    expect(withSeconds).not.toBeNull();
    expect(new Date(withSeconds!).getSeconds()).toBe(30);

    // a date-only value: `new Date` reads it as UTC midnight, which is a
    // different instant from local midnight — and still a usable bound
    const dateOnly = localInputToIso("2026-08-11");
    expect(dateOnly).not.toBeNull();

    // and the canonical minute-precision value still round-trips
    expect(localInputToIso(isoToLocalInput("2026-08-11T10:00:00.000Z")!)).toBe("2026-08-11T10:00:00.000Z");
  });

  it("pads the year to 4 digits so pre-1000 dates stay valid input values", () => {
    expect(isoToLocalInput("0999-01-01T00:00:00.000Z")).toMatch(/^\d{4}-01-01T/);
    expect(isoToLocalInput("0999-01-01T00:00:00.000Z").startsWith("0999-")).toBe(true);
  });
});

describe("isoToEndOfMinute (instant -> inclusive minute end)", () => {
  it("widens an ISO instant to the last millisecond of its minute", () => {
    const iso = "2026-08-25T20:55:00.000Z";
    expect(isoToEndOfMinute(iso)).toBe("2026-08-25T20:55:59.999Z");
    // an instant already carrying seconds keeps them, like the input path does
    expect(isoToEndOfMinute("2026-08-25T20:55:30.123Z")).toBe("2026-08-25T20:55:59.999Z");
    // and never moves backwards
    expect(new Date(isoToEndOfMinute(iso)!).getTime()).toBeGreaterThanOrEqual(new Date(iso).getTime());
  });

  it("maps null/empty and unparseable values to null", () => {
    expect(isoToEndOfMinute(null)).toBeNull();
    expect(isoToEndOfMinute("")).toBeNull();
    expect(isoToEndOfMinute("bogus")).toBeNull();
  });

  // One rule, two producers: an ISO instant and the datetime-local value naming
  // the same minute must widen identically, or the slider and the popover disagree
  // about the last included instant.
  it("agrees with localInputToIsoInclusive for the same minute", () => {
    const local = isoToLocalInput("2026-08-11T10:00:00.000Z")!;
    expect(isoToEndOfMinute("2026-08-11T10:00:00.000Z")).toBe(localInputToIsoInclusive(local));
  });

  // The suite pins TZ=Europe/Berlin (package.json test script). On 2026-10-25
  // 03:00→02:00 (01:00 UTC) the local clock repeats 02:00–03:00, so 00:30Z and
  // 01:30Z are BOTH local 02:30 under different UTC offsets. The old
  // setSeconds(59, 999) re-resolved that wall clock to its earlier occurrence,
  // rewinding the bound by an hour: 01:30Z came back as 00:30:59.999Z, dropping
  // the last 30 real minutes of the `to` filter.
  it("widens correctly across the DST repeated hour (fall-back)", () => {
    expect(Intl.DateTimeFormat().resolvedOptions().timeZone).toBe("Europe/Berlin");
    // 00:30Z is local 02:30 CEST (first pass), 01:30Z is local 02:30 CET (second)
    expect(isoToLocalInput("2026-10-25T00:30:00.000Z")).toBe("2026-10-25T02:30");
    expect(isoToLocalInput("2026-10-25T01:30:00.000Z")).toBe("2026-10-25T02:30");
    // both widen to the end of THEIR own minute — no rewinding by the DST offset
    expect(isoToEndOfMinute("2026-10-25T00:30:00.000Z")).toBe("2026-10-25T00:30:59.999Z");
    expect(isoToEndOfMinute("2026-10-25T01:30:00.000Z")).toBe("2026-10-25T01:30:59.999Z");
    // and the ordinary minutes either side of the transition are untouched
    expect(isoToEndOfMinute("2026-10-24T23:30:00.000Z")).toBe("2026-10-24T23:30:59.999Z");
    expect(isoToEndOfMinute("2026-10-25T02:30:00.000Z")).toBe("2026-10-25T02:30:59.999Z");
  });
});
