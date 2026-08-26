import { describe, it, expect } from "vitest";
import { isoToLocalInput, localInputToIso } from "src/util/dateTimeUtil";

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
});
