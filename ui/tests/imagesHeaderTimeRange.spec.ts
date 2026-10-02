import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createApp, h, nextTick, reactive, ref, type App } from "vue";
import { createPinia, setActivePinia, type Pinia } from "pinia";
import ImagesHeader from "src/components/image/ImagesHeader.vue";

// The Time popover panel of the images toolbar: the two datetime-local inputs,
// the slider wired to them, and the debounced route write. The e2e suite
// (tests/e2e/time-range.spec.ts) covers the wired-up browser behaviour; these
// pin the decisions only the component makes.

vi.mock("src/api", () => ({
  api: {
    uploads: { list: vi.fn().mockResolvedValue({ items: [] }) },
    imageTags: { list: vi.fn().mockResolvedValue({ items: [] }) },
  },
}));

// VueUse 10's useDebounceFn: the panel is opened and closed by a headlessui
// Popover, and the real timers are what make the "armed after unmount" bug
// observable, so the debounce is driven by the clock rather than faked away.
const type = (selector: string, value: string) => {
  const el = document.querySelector(selector) as HTMLInputElement;
  el.value = value;
  el.dispatchEvent(new Event("input", { bubbles: true }));
};

interface Harness {
  app: App;
  emits: { timeRange: [string | null, string | null, unknown?][] };
  set: (patch: Record<string, unknown>) => Promise<void>;
  input: (which: "from" | "to") => HTMLInputElement;
  unmount: () => void;
}

let current: Harness | null = null;

async function mount(props: Record<string, unknown> = {}): Promise<Harness> {
  const pinia: Pinia = createPinia();
  setActivePinia(pinia);
  const host = document.createElement("div");
  document.body.appendChild(host);
  const state = reactive<Record<string, unknown>>({
    totalImageCount: 11,
    showFilter: true,
    timeFrom: null,
    timeTo: null,
    timeBounds: null,
    timeTicks: null,
    timeRangeSuspended: false,
    ...props,
  });
  const emits: Harness["emits"] = { timeRange: [] };
  let unmounted = false;
  const app: App = createApp({
    setup() {
      return () =>
        h(ImagesHeader, {
          ...state,
          onTimeRange: (f: string | null, t: string | null, o?: unknown) => emits.timeRange.push([f, t, o]),
        });
    },
  });
  app.use(pinia);
  app.mount(host);
  await nextTick();
  // the panel only exists while the popover is open
  document.querySelector<HTMLElement>('[data-testid="time-range-button"]')!.click();
  await nextTick();
  await nextTick();

  const harness: Harness = {
    app,
    emits,
    set: async (patch) => {
      Object.assign(state, patch);
      await nextTick();
    },
    input: (which) => document.querySelector(`[data-testid="time-${which}-input"]`) as HTMLInputElement,
    unmount: () => {
      if (unmounted) return;
      unmounted = true;
      app.unmount();
      host.remove();
      const panel = document.querySelector('[data-testid="time-range-panel"]');
      panel?.closest("div[class*='absolute']")?.parentElement?.remove();
    },
  };
  current = harness;
  return harness;
}

async function settle(ms = 500) {
  await new Promise((r) => setTimeout(r, ms));
  await nextTick();
}

afterEach(() => {
  current?.unmount();
  current = null;
});

describe("ImagesHeader time inputs", () => {
  // localInputToIso returns null both for an EMPTY field (which legitimately
  // means "no bound") and for a wall clock it refuses — a DST spring-forward
  // gap, which really does not happen. Collapsing the two deleted that side of
  // the route: the filter silently widened while the input still showed the
  // refused text.
  it("leaves the route alone when a typed time does not exist", async () => {
    const h2 = await mount();
    // Europe/Berlin springs forward on 2026-03-29: 02:00 jumps to 03:00
    type('[data-testid="time-from-input"]', "2026-03-29T02:30");
    type('[data-testid="time-to-input"]', "2026-03-29T05:00");
    await settle();

    expect(h2.emits.timeRange).toHaveLength(0);
    expect(h2.input("from").getAttribute("aria-invalid")).toBe("true");
    expect(h2.input("to").getAttribute("aria-invalid")).toBe("false");
    expect(document.querySelector('[data-testid="time-range-invalid"]')).not.toBeNull();
    // the refused text is still on screen, now marked rather than silently dropped
    expect(h2.input("from").value).toBe("2026-03-29T02:30");
  });

  it("still commits the valid side's companion and clears the error on the next valid keystroke", async () => {
    const h2 = await mount();
    type('[data-testid="time-from-input"]', "2026-03-29T02:30");
    await settle();
    expect(h2.emits.timeRange).toHaveLength(0);

    type('[data-testid="time-from-input"]', "2026-03-29T04:30");
    await settle();
    expect(h2.input("from").getAttribute("aria-invalid")).toBe("false");
    expect(document.querySelector('[data-testid="time-range-invalid"]')).toBeNull();
    expect(h2.emits.timeRange).toHaveLength(1);
    // midnight UTC in late March is 02:00 CEST; assert the instant, not a
    // hand-computed offset
    expect(h2.emits.timeRange[0][0]).toBe(new Date("2026-03-29T04:30").toISOString());
  });

  it("treats an emptied field as no bound, not as an error", async () => {
    const h2 = await mount();
    type('[data-testid="time-from-input"]', "2026-01-01T00:00");
    type('[data-testid="time-to-input"]', "2026-01-02T23:59");
    await settle();
    expect(h2.emits.timeRange).toHaveLength(1);

    type('[data-testid="time-to-input"]', "");
    await settle();
    expect(h2.emits.timeRange).toHaveLength(2);
    expect(h2.emits.timeRange[1][0]).toBe(new Date("2026-01-01T00:00").toISOString());
    expect(h2.emits.timeRange[1][1]).toBeNull();
    expect(document.querySelector('[data-testid="time-range-invalid"]')).toBeNull();
  });

  // vueuse 10's useDebounceFn exposes no cancel and registers no scope dispose,
  // so an armed 400ms write fired after unmount and put ?from/?to onto whatever
  // route the user had navigated to in the meantime.
  it("does not write the route after the panel is gone", async () => {
    const h2 = await mount();
    type('[data-testid="time-from-input"]', "2026-01-01T00:00");
    // the panel goes away inside the debounce window
    h2.app.unmount();
    await settle();
    expect(h2.emits.timeRange).toHaveLength(0);
  });
});

describe("ImagesHeader trigger", () => {
  // The active-filter badge was a literal "·" with no accessible text, so a
  // screen reader announced the button as "Time ·".
  it("names the active time range in the trigger", async () => {
    await mount({ timeFrom: "2026-01-01T00:00:00.000Z" });
    const button = document.querySelector('[data-testid="time-range-button"]') as HTMLElement;
    expect(button.textContent).toContain("Time range active");
    // the dot is decoration, not the label
    const badge = button.querySelector(".sr-only") as HTMLElement;
    expect(badge).not.toBeNull();
    expect(badge.textContent).toBe("Time range active");
  });

  it("shows no badge when no range is applied", async () => {
    await mount();
    const button = document.querySelector('[data-testid="time-range-button"]') as HTMLElement;
    expect(button.textContent).not.toContain("Time range active");
  });
});

describe("ImagesHeader suspension", () => {  // Suspension disables the slider, but the inputs stayed live and Images.vue
  // only re-arms on an EMPTIED range — so typing while suspended wrote the
  // route, moved the chip, and did not filter.
  it("disables the inputs together with the slider", async () => {
    const h2 = await mount({ timeRangeSuspended: true });
    expect(h2.input("from").disabled).toBe(true);
    expect(h2.input("to").disabled).toBe(true);
  });

  it("re-enables them when the range applies again", async () => {
    const h2 = await mount({ timeRangeSuspended: true });
    await h2.set({ timeRangeSuspended: false });
    expect(h2.input("from").disabled).toBe(false);
    expect(h2.input("to").disabled).toBe(false);
  });
});
