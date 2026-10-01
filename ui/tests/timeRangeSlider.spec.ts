import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { createApp, h, nextTick, reactive, type App } from "vue";
import TimeRangeSlider from "src/components/image/TimeRangeSlider.vue";

// The Time popover's dual-thumb slider. The Playwright suite
// (tests/e2e/time-range.spec.ts) drives the real thing in a browser; these are
// the state-machine invariants, which jsdom can assert exactly and cheaply —
// and which a browser test cannot easily reach, because whether a response lands
// between two gestures is a matter of timing there.

const TRACK_LEFT = 0;
const TRACK_WIDTH = 1000;
const THUMB_RADIUS = 7;
const MINUTE = 60_000;

const iso = (ms: number) => new Date(ms).toISOString();
const stepOf = (ms: number) => Math.floor(ms / MINUTE);
/** The ISO a given instant's minute step stands for — what the component emits. */
const isoFor = (ms: number) => iso(Math.floor(ms / MINUTE) * MINUTE);

const DOMAIN_MIN = Date.parse("2026-08-11T10:00:00.000Z");
const DOMAIN_MAX = Date.parse("2026-08-11T12:00:00.000Z");
const MIN_STEP = stepOf(DOMAIN_MIN);
const MAX_STEP = stepOf(DOMAIN_MAX);

/** Client x that lands the pointer on a given minute step of the track. */
const xForStep = (step: number) => {
  const pct = (step - MIN_STEP) / (MAX_STEP - MIN_STEP);
  return TRACK_LEFT + THUMB_RADIUS + pct * (TRACK_WIDTH - 2 * THUMB_RADIUS);
};

beforeAll(() => {
  // jsdom has no layout and no ResizeObserver: the component measures the track
  // and only then renders the thumbs, the ticks and the selected segment.
  class StubResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  (globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = StubResizeObserver;
  // jsdom 24 ships no PointerEvent, and the whole drag is pointer-driven.
  class StubPointerEvent extends MouseEvent {
    readonly pointerId: number;
    constructor(type: string, init: PointerEventInit = {}) {
      super(type, init);
      this.pointerId = init.pointerId ?? 0;
    }
  }
  (globalThis as unknown as { PointerEvent: unknown }).PointerEvent = StubPointerEvent;
  Object.defineProperty(HTMLElement.prototype, "clientWidth", { configurable: true, value: TRACK_WIDTH });
  HTMLElement.prototype.getBoundingClientRect = function () {
    return {
      left: TRACK_LEFT,
      right: TRACK_LEFT + TRACK_WIDTH,
      top: 0,
      bottom: 24,
      width: TRACK_WIDTH,
      height: 24,
      x: TRACK_LEFT,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect;
  };
});

interface ChangeEvent {
  type: "change";
  from: string | null;
  to: string | null;
  opts?: { replace?: boolean };
}
interface PreviewEvent {
  type: "preview";
  from: string;
  to: string;
}
interface RestoreEvent {
  type: "restore";
  from: string;
  to: string;
}
type Emitted = ChangeEvent | PreviewEvent | RestoreEvent;

interface Harness {
  set: (patch: Record<string, unknown>) => Promise<void>;
  events: Emitted[];
  changes: () => ChangeEvent[];
  lastChange: () => ChangeEvent;
  restores: () => RestoreEvent[];
  track: () => HTMLElement;
  thumb: (which: "low" | "high") => HTMLElement;
  step: (which: "low" | "high") => number;
  tickCount: () => number;
  unmount: () => void;
}

let current: Harness | null = null;

async function mount(initial: Record<string, unknown> = {}): Promise<Harness> {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const state = reactive<Record<string, unknown>>({
    min: iso(DOMAIN_MIN),
    max: iso(DOMAIN_MAX),
    from: null,
    to: null,
    disabled: false,
    ticks: null,
    ...initial,
  });
  const events: Emitted[] = [];
  const app: App = createApp({
    setup() {
      return () =>
        h(TimeRangeSlider, {
          ...state,
          onChange: (from: string | null, to: string | null, opts?: { replace?: boolean }) => events.push({ type: "change", from, to, opts }),
          onPreview: (from: string, to: string) => events.push({ type: "preview", from, to }),
          onRestore: (from: string, to: string) => events.push({ type: "restore", from, to }),
        });
    },
  });
  app.mount(host);
  // the measure hook watches the track ref flush:"post", so the thumbs only
  // exist one tick after the mount
  await nextTick();

  const q = (sel: string) => host.querySelector(sel) as HTMLElement;
  const harness: Harness = {
    events,
    changes: () => events.filter((e): e is ChangeEvent => e.type === "change"),
    lastChange: () => {
      const found = events.filter((e): e is ChangeEvent => e.type === "change");
      if (!found.length) throw new Error("no change event was emitted");
      return found[found.length - 1];
    },
    restores: () => events.filter((e): e is RestoreEvent => e.type === "restore"),
    set: async (patch) => {
      Object.assign(state, patch);
      await nextTick();
    },
    track: () => q('[class*="touch-none"]'),
    thumb: (which) => q(`[data-testid="range-${which === "low" ? "start" : "end"}-thumb"]`),
    step: (which) => Number(harness.thumb(which).getAttribute("aria-valuenow")),
    tickCount: () => host.querySelectorAll('[class*="bg-primary-300"]').length,
    unmount: () => {
      app.unmount();
      host.remove();
    },
  };
  current = harness;
  return harness;
}

/**
 * Dispatch a pointer event. pointerdown goes on the track (that is what the
 * component listens on), the rest on the window (that is where the drag
 * listeners live), mirroring how a real drag is delivered.
 */
function pointer(type: "pointerdown" | "pointermove" | "pointerup" | "pointercancel", clientX: number, pointerId = 1) {
  const target = type === "pointerdown" ? current!.track() : window;
  target.dispatchEvent(new PointerEvent(type, { clientX, pointerId, bubbles: true, cancelable: true }));
}

function key(target: "low" | "high", k: string) {
  current!.thumb(target).dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true }));
}

beforeEach(() => {
  vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
  current?.unmount();
  current = null;
});

/** Every emitted pair must be ascending — anything else is a 400 from the API. */
function expectOrdered(events: { from: string | null; to: string | null }[]) {
  for (const e of events) {
    expect(new Date(e.from!).getTime()).toBeLessThanOrEqual(new Date(e.to!).getTime());
  }
}

describe("TimeRangeSlider commit", () => {
  // THE bug this file exists for: the cross-clamp moved the other thumb without
  // marking it touched, so a commit took that side from the PARENT instead of
  // from the thumb — emitting from=14:00, to=12:00. images_controller.go answers
  // 400 invalid_time_range and loadImages turns the whole grid into an error page.
  it("never emits from after to, however the thumbs cross (pointer)", async () => {
    // the range sits inside the domain, so the low thumb has room to be pushed
    // past the high one (the domain clamps at its own edges)
    const h2 = await mount({ from: isoFor((MIN_STEP + 10) * MINUTE), to: isoFor((MIN_STEP + 20) * MINUTE) });
    pointer("pointerdown", xForStep(MIN_STEP + 10));
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 35));
    await nextTick();
    pointer("pointerup", xForStep(MIN_STEP + 35));
    await nextTick();

    const last = h2.lastChange();
    expectOrdered([last]);
    // the cross-clamp carried the high thumb, so both sides are committed
    expect(last.from).toBe(isoFor((MIN_STEP + 35) * MINUTE));
    expect(last.to).toBe(isoFor((MIN_STEP + 35) * MINUTE));
    // and the track agrees with what was committed, rather than showing the
    // high thumb stranded where the route used to say
    expect(h2.step("low")).toBe(MIN_STEP + 35);
    expect(h2.step("high")).toBe(MIN_STEP + 35);
  });

  it("never emits from after to, however the thumbs cross (keyboard)", async () => {
    // The keyboard moves and commits in the SAME handler, so a cross-clamp
    // deferred to the next tick would not run in time.
    const h2 = await mount({ from: isoFor((MIN_STEP + 10) * MINUTE), to: isoFor((MIN_STEP + 20) * MINUTE) });
    key("low", "End");
    await nextTick();

    expectOrdered(h2.changes());
    const last = h2.lastChange();
    expect(last.from).toBe(isoFor(DOMAIN_MAX));
    expect(last.to).toBe(isoFor(DOMAIN_MAX));
    // the track followed: the high thumb was dragged to the low thumb's position
    // in the SAME handler, before the commit went out
    expect(h2.step("low")).toBe(MAX_STEP);
    expect(h2.step("high")).toBe(MAX_STEP);
  });

  it("stays ordered across a sweep that crosses the thumbs twice", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP));
    await nextTick();
    for (const step of [MAX_STEP - 10, MIN_STEP + 5, MAX_STEP - 1]) {
      pointer("pointermove", xForStep(Math.min(Math.max(step, MIN_STEP), MAX_STEP)));
      await nextTick();
    }
    pointer("pointerup", xForStep(MAX_STEP - 1));
    await nextTick();
    expect(h2.changes().length).toBeGreaterThan(0);
    expectOrdered(h2.changes());
  });

  // The behaviour the cross-clamp must NOT break: an untouched side goes back
  // out exactly as it came in, so a one-thumb gesture never deletes the other
  // bound and widens the filter to an open end.
  it("keeps the untouched bound when only one thumb moves", async () => {
    const to = isoFor(DOMAIN_MAX);
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to });
    pointer("pointerdown", xForStep(MIN_STEP));
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 5));
    await nextTick();
    pointer("pointerup", xForStep(MIN_STEP + 5));
    await nextTick();

    const last = h2.lastChange();
    expect(last.from).toBe(isoFor((MIN_STEP + 5) * MINUTE));
    expect(last.to).toBe(to);
  });

  it("keeps the untouched bound on the keyboard path too", async () => {
    const from = isoFor(DOMAIN_MIN);
    const h2 = await mount({ from, to: isoFor(DOMAIN_MAX) });
    key("high", "ArrowLeft");
    await nextTick();
    const last = h2.lastChange();
    expect(last.from).toBe(from);
    expect(last.to).toBe(isoFor(DOMAIN_MAX - MINUTE));
    expect(last.opts).toEqual({ replace: true });
  });

  it("leaves an open-ended side open", async () => {
    const to = isoFor(DOMAIN_MAX);
    const h2 = await mount({ from: null, to });
    pointer("pointerdown", xForStep(MIN_STEP + 10));
    await nextTick();
    pointer("pointerup", xForStep(MIN_STEP + 10));
    await nextTick();
    // the pointerdown alone picks the low thumb and commits it
    expect(h2.lastChange().from).toBe(isoFor((MIN_STEP + 10) * MINUTE));
    expect(h2.lastChange().to).toBe(to);
  });
});

describe("TimeRangeSlider props re-sync", () => {
  // The swap this used to do hid the inversion in the DISPLAY: a route carrying
  // from=14:00&to=12:00 rendered as a plausible 12:00-14:00 slider while the
  // filter stayed inverted and the request 400'd.
  it("shows an inverted incoming pair as given, never swapped", async () => {
    // ordered first, then the route is replaced by an inverted pair: the re-sync
    // is where a swap would happen
    const h2 = await mount({ from: isoFor((MIN_STEP + 30) * MINUTE), to: isoFor((MIN_STEP + 90) * MINUTE) });
    await h2.set({ from: isoFor((MIN_STEP + 90) * MINUTE), to: isoFor((MIN_STEP + 30) * MINUTE) });
    expect(h2.step("low")).toBe(MIN_STEP + 90);
    expect(h2.step("high")).toBe(MIN_STEP + 30);
  });

  it("still refuses to emit the inverted pair it was handed", async () => {
    const h2 = await mount({ from: isoFor((MIN_STEP + 30) * MINUTE), to: isoFor((MIN_STEP + 90) * MINUTE) });
    await h2.set({ from: isoFor((MIN_STEP + 90) * MINUTE), to: isoFor((MIN_STEP + 30) * MINUTE) });
    pointer("pointerdown", xForStep(MIN_STEP + 90));
    await nextTick();
    pointer("pointerup", xForStep(MIN_STEP + 90));
    await nextTick();
    expectOrdered(h2.changes());
  });

  // A bounds/tick response can land mid-drag: the popover-open fetch is not
  // awaited before the pointer goes down. Re-syncing snapped the thumb out from
  // under the pointer, and the release then committed the parent's values.
  it("does not re-sync the thumbs while a drag is in flight", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP));
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 5));
    await nextTick();

    // the tick/bounds response lands here
    await h2.set({ ticks: [iso(DOMAIN_MIN + 30_000), iso(DOMAIN_MAX)] });
    expect(h2.step("low")).toBe(MIN_STEP + 5);

    pointer("pointerup", xForStep(MIN_STEP + 5));
    await nextTick();
    // the gesture survived: the dragged value is what was committed
    expect(h2.lastChange().from).toBe(isoFor((MIN_STEP + 5) * MINUTE));
  });

  it("orders the pair when the route changes under an in-flight drag", async () => {
    // The untouched side comes straight back out of props, and props are live.
    // A To that lands before the dragged thumb mid-gesture would otherwise go
    // out straddled.
    const h2 = await mount({ from: isoFor((MIN_STEP + 10) * MINUTE), to: isoFor((MIN_STEP + 80) * MINUTE) });
    pointer("pointerdown", xForStep(MIN_STEP + 10));
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 60));
    await nextTick();
    await h2.set({ to: isoFor((MIN_STEP + 5) * MINUTE) });
    pointer("pointerup", xForStep(MIN_STEP + 60));
    await nextTick();

    expectOrdered(h2.changes());
    // the TOUCHED side is the user's decision and wins; the other is pulled onto it
    expect(h2.lastChange().from).toBe(isoFor((MIN_STEP + 60) * MINUTE));
    expect(h2.lastChange().to).toBe(isoFor((MIN_STEP + 60) * MINUTE));
  });

  it("keeps the gesture across a pure domain change", async () => {    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP));
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 5));
    await nextTick();
    // a bounds response that widens the domain under the pointer
    await h2.set({ min: iso(DOMAIN_MIN - 30 * MINUTE), max: iso(DOMAIN_MAX + 30 * MINUTE) });

    pointer("pointerup", xForStep(MIN_STEP + 5));
    await nextTick();
    expect(h2.lastChange().from).toBe(isoFor((MIN_STEP + 5) * MINUTE));
  });

  it("follows the route once the gesture is over", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    await h2.set({ from: isoFor(DOMAIN_MIN + 20 * MINUTE) });
    expect(h2.step("low")).toBe(MIN_STEP + 20);
  });

  // toStep(null, minStep) put the DOMAIN edge in the parent's input, so
  // cancelling a drag on an open-ended range showed a bound the route does not
  // carry and revealed "Clear time range" for a range that was never applied.
  it("restores an open-ended route as empty, not as the domain edge", async () => {
    const h2 = await mount({ from: null, to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP + 10));
    await nextTick();
    pointer("pointercancel", xForStep(MIN_STEP + 10));
    await nextTick();

    expect(h2.restores()).toHaveLength(1);
    expect(h2.restores()[0].from).toBe("");
    expect(h2.restores()[0].to).toBe(isoFor(DOMAIN_MAX));
    expect(h2.changes()).toHaveLength(0);
  });

  it("restores a closed range as the route's own bounds", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN + 10 * MINUTE), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP + 20 * MINUTE));
    await nextTick();
    pointer("pointercancel", xForStep(MIN_STEP + 20 * MINUTE));
    await nextTick();
    expect(h2.restores()[0].from).toBe(isoFor(DOMAIN_MIN + 10 * MINUTE));
    expect(h2.restores()[0].to).toBe(isoFor(DOMAIN_MAX));
  });
});

describe("TimeRangeSlider pointer handling", () => {
  // A second finger on a phone is its own pointer: unfiltered, its move dragged
  // the range around and its release committed it.
  it("ignores moves and releases from another pointer", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP + 5), 1);
    await nextTick();
    const parked = h2.step("low");

    pointer("pointermove", xForStep(MAX_STEP), 2);
    await nextTick();
    expect(h2.step("low")).toBe(parked);
    // the second finger lifting must not commit either
    pointer("pointerup", xForStep(MAX_STEP), 2);
    await nextTick();
    expect(h2.changes()).toHaveLength(0);

    // the drag is still alive and the first pointer still drives it
    pointer("pointermove", xForStep(MIN_STEP + 8), 1);
    await nextTick();
    pointer("pointerup", xForStep(MIN_STEP + 8), 1);
    await nextTick();
    expect(h2.lastChange().from).toBe(isoFor((MIN_STEP + 8) * MINUTE));
  });

  it("ignores a cancel from another pointer", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP + 5), 1);
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 9), 1);
    await nextTick();
    pointer("pointercancel", xForStep(MIN_STEP + 9), 2);
    await nextTick();
    expect(h2.step("low")).toBe(MIN_STEP + 9);
    expect(h2.restores()).toHaveLength(0);
  });

  it("tears the drag down on window blur and restores", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    pointer("pointerdown", xForStep(MIN_STEP + 5), 1);
    await nextTick();
    pointer("pointermove", xForStep(MIN_STEP + 9), 1);
    await nextTick();
    window.dispatchEvent(new Event("blur"));
    await nextTick();
    expect(h2.step("low")).toBe(MIN_STEP);
    expect(h2.changes()).toHaveLength(0);
  });
});

describe("TimeRangeSlider accessibility", () => {
  it("takes both thumbs out of the tab order when disabled", async () => {
    const h2 = await mount({ disabled: true, from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    expect(h2.thumb("low").getAttribute("tabindex")).toBe("-1");
    expect(h2.thumb("high").getAttribute("tabindex")).toBe("-1");
    expect(h2.thumb("low").getAttribute("aria-disabled")).toBe("true");
  });

  it("keeps them focusable when enabled", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    expect(h2.thumb("low").getAttribute("tabindex")).toBe("0");
    expect(h2.thumb("low").getAttribute("aria-disabled")).toBeNull();
  });

  it("bounds each thumb by the other thumb, not by the whole domain", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN + 30 * MINUTE), to: isoFor(DOMAIN_MAX - 30 * MINUTE) });
    const lo = h2.thumb("low");
    const hi = h2.thumb("high");
    // the low thumb cannot be pushed past the high one
    expect(lo.getAttribute("aria-valuemax")).toBe(hi.getAttribute("aria-valuenow"));
    expect(hi.getAttribute("aria-valuemin")).toBe(lo.getAttribute("aria-valuenow"));
  });

  it("announces a human time, not the raw epoch minutes", async () => {
    const h2 = await mount({ from: isoFor(DOMAIN_MIN), to: isoFor(DOMAIN_MAX) });
    const text = h2.thumb("low").getAttribute("aria-valuetext") ?? "";
    expect(text).toMatch(/2026/);
    expect(text).not.toMatch(/^29\d{6}$/);
  });
});

describe("TimeRangeSlider ticks", () => {
  // Two photos can share a capturedAtCorrected; a duplicate v-for key mis-patches
  // the strip on the next update (a node is reused for the wrong sample) and
  // logs a Vue warning.
  it("renders one tick per sample even when timestamps repeat", async () => {
    const a = iso(DOMAIN_MIN + MINUTE);
    const b = iso(DOMAIN_MIN + 90 * MINUTE);
    const h2 = await mount({ ticks: [a, a, b] });
    expect(h2.tickCount()).toBe(3);

    await h2.set({ ticks: [b, b, a] });
    expect(h2.tickCount()).toBe(3);
    const warnings = vi.mocked(console.warn).mock.calls.map((c) => String(c[0]));
    expect(warnings.filter((w) => /Duplicate keys/.test(w))).toHaveLength(0);
  });

  it("re-positions every tick when the samples change", async () => {
    const a = iso(DOMAIN_MIN + MINUTE);
    const b = iso(DOMAIN_MIN + 90 * MINUTE);
    const h2 = await mount({ ticks: [a, b] });
    const offsets = () =>
      Array.from(document.querySelectorAll('[class*="bg-primary-300"]')).map((n) => (n as HTMLElement).style.left);
    const afterA = offsets();
    await h2.set({ ticks: [b, b] });
    // both samples are at the same instant now, so both sit at the same offset
    expect(offsets()).toEqual([afterA[1], afterA[1]]);
  });
});
