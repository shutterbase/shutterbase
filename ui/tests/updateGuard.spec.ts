import { describe, it, expect } from "vitest";
import { isVersionChange, shouldReloadOnChunkError, RELOAD_COOLDOWN_MS } from "src/util/updateGuard";

describe("isVersionChange", () => {
  it("detects a real version flip", () => {
    expect(isVersionChange("v1.12.0", "v1.12.1")).toBe(true);
  });
  it("is false for the same version", () => {
    expect(isVersionChange("v1.12.0", "v1.12.0")).toBe(false);
  });
  it("never fires on unknown versions (health unreachable)", () => {
    expect(isVersionChange(null, "v1.12.1")).toBe(false);
    expect(isVersionChange("v1.12.0", null)).toBe(false);
    expect(isVersionChange("v1.12.0", undefined)).toBe(false);
    expect(isVersionChange("", "v1.12.1")).toBe(false);
  });
});

describe("shouldReloadOnChunkError", () => {
  const now = 1_000_000_000;
  it("allows the first reload", () => {
    expect(shouldReloadOnChunkError(null, now)).toBe(true);
  });
  it("suppresses a second reload within the cooldown", () => {
    expect(shouldReloadOnChunkError(now - RELOAD_COOLDOWN_MS / 2, now)).toBe(false);
  });
  it("allows a reload after the cooldown", () => {
    expect(shouldReloadOnChunkError(now - RELOAD_COOLDOWN_MS - 1, now)).toBe(true);
  });
});
