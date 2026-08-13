import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

vi.mock("canvas-confetti", () => ({ default: vi.fn() }));

import confetti from "canvas-confetti";
import { celebrate } from "src/util/confetti";

describe("celebrate", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("fires an immediate burst and staggered side volleys", () => {
    celebrate();
    expect(confetti).toHaveBeenCalledTimes(1);
    vi.runAllTimers();
    expect(confetti).toHaveBeenCalledTimes(7);
  });

  it("every burst respects reduced motion and flies above the modal", () => {
    celebrate();
    vi.runAllTimers();
    for (const [options] of vi.mocked(confetti).mock.calls) {
      expect(options).toMatchObject({ disableForReducedMotion: true, zIndex: 2000 });
    }
  });
});
