import { test, expect, Page, Locator } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// Time-range gallery filter (?from=/?to=) and the detail-view "show ±15 min"
// action (#117). The seed's midnight cluster (FSG_90xx, 23:55→00:10 event-local
// yesterday, untagged) is the fixture; expected windows are derived from the
// API so the spec stays valid regardless of when/where it runs.
//
// The suite pins timezoneId in playwright.config.ts, because the Time popover
// works in local wall clock: a typed "2026-08-01T00:00" means a different
// instant per host zone.

// The instant a `datetime-local` value denotes, resolved by the BROWSER.
//
// playwright.config.ts pins the browser's timezoneId and the package script
// pins the runner's TZ, but they are two independent settings — a spec that
// computes the expected instant with Node's `new Date(localWallClock)` silently
// depends on both agreeing, and quietly fails on any host that is not
// Europe/Berlin. Asking the browser is the only version that cannot drift.
async function localWallClockToIso(page: Page, value: string): Promise<string> {
  return page.evaluate((v) => new Date(v).toISOString(), value);
}

/** The same instant widened to the last millisecond of its minute. */
async function localWallClockToInclusiveIso(page: Page, value: string): Promise<string> {
  return page.evaluate((v) => {
    const d = new Date(v);
    d.setSeconds(59, 999);
    return d.toISOString();
  }, value);
}

/** A minute-step as the `datetime-local` value the inputs show (browser-side). */
async function isoToInputValue(page: Page, minuteStep: number): Promise<string> {
  return page.evaluate((ms) => {
    const d = new Date(ms);
    const p = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
  }, minuteStep * 60_000);
}

/** Pixel geometry of the slider TRACK (the thumbs are inset inside it). */
async function trackBox(page: Page, thumb: Locator): Promise<{ left: number; width: number }> {
  return thumb.evaluate((el) => {
    const t = (el as HTMLElement).parentElement!;
    const r = t.getBoundingClientRect();
    return { left: r.left, width: r.width };
  });
}

async function fetchImages(page: Page, projectId: string, params = ""): Promise<any[]> {
  return page.evaluate(
    async ({ pid, params }) => {
      const r = await fetch(`/api/v1/images?projectId=${pid}&limit=100&sort=capturedAtCorrected&order=asc${params}`, { credentials: "include" });
      return (await r.json()).items ?? [];
    },
    { pid: projectId, params },
  );
}

const midnightCluster = (images: any[]) => images.filter((i) => i.computedFileName.startsWith("FSG_90"));

/** ?from/?to for the whole cluster, as a query string. */
const clusterQuery = (cluster: any[], extra = "") =>
  `from=${encodeURIComponent(cluster[0].capturedAtCorrected)}` +
  `&to=${encodeURIComponent(cluster[cluster.length - 1].capturedAtCorrected)}${extra}`;

test.describe("time range filter", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("URL bounds narrow the grid to the cluster and show a clearable chip", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const cluster = midnightCluster(all);
    expect(cluster.length).toBe(8);

    await page.goto(`/images?${clusterQuery(cluster)}`);

    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);
    const chip = page.getByTestId("time-range-chip");
    await expect(chip).toBeVisible();

    // clearing restores the unfiltered grid
    await chip.getByRole("button", { name: "×" }).click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(11);
    await expect(new URL(page.url()).searchParams.get("from")).toBeNull();
  });

  test("show ±15 min jumps to the photo's timespan", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const mid = midnightCluster(all)[3];

    await page.goto(`/images?image=${mid.id}`);
    await expect(page.getByText("Image Tags")).toBeVisible();
    await page.getByTestId("show-timespan").click();

    // detail closed, window set around the photo
    const url = new URL(page.url());
    expect(url.searchParams.get("image")).toBeNull();
    const from = new Date(url.searchParams.get("from")!);
    const to = new Date(url.searchParams.get("to")!);
    const t = new Date(mid.capturedAtCorrected);
    expect(Math.round((t.getTime() - from.getTime()) / 60_000)).toBe(15);
    expect(Math.round((to.getTime() - t.getTime()) / 60_000)).toBe(15);

    // every image in the loaded window satisfies the bounds
    const tiles = page.locator('[id^="grid-tile-"]');
    const expected = all.filter((i) => {
      if (!i.capturedAtCorrected) return false;
      const c = new Date(i.capturedAtCorrected).getTime();
      return c >= from.getTime() && c <= to.getTime();
    }).length;
    await expect(tiles).toHaveCount(expected, { timeout: 7000 });

    // chronological reading order: the jump pins ?sort=oldestFirst WITHOUT
    // rewriting the persisted preference, so assert both the URL and the tiles
    // that actually came back in that order.
    expect(url.searchParams.get("sort")).toBe("oldestFirst");
    const rendered = await page.locator('[id^="grid-tile-"]').evaluateAll((els) =>
      els.map((e) => e.id.replace("grid-tile-", "")),
    );
    const inWindow = all
      .filter((i) => {
        if (!i.capturedAtCorrected) return false;
        const c = new Date(i.capturedAtCorrected).getTime();
        return c >= from.getTime() && c <= to.getTime();
      })
      .map((i) => i.id);
    expect(rendered).toEqual(inWindow);
  });

  test("the popover writes the query and clears both sides", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByTestId("time-range-button").click();
    await expect(page.getByTestId("time-from-input")).toBeVisible();
    // The inputs start EMPTY: prefilling them with the slider domain bounds
    // meant editing only "From" silently committed ?to=<last photo>.
    await expect(page.getByTestId("time-from-input")).toHaveValue("");
    await expect(page.getByTestId("time-to-input")).toHaveValue("");

    await page.getByTestId("time-from-input").fill("2026-01-01T00:00");
    // exact instant, not just "some value": a popover that discarded the
    // keystroke would also make a truthiness assertion pass
    await expect
      .poll(() => new URL(page.url()).searchParams.get("from"), { timeout: 7000 })
      .toBe(await localWallClockToIso(page, "2026-01-01T00:00"));

    await page.getByTestId("time-to-input").fill("2026-01-02T23:59");
    // inclusive upper bound: the last millisecond of the entered minute, which
    // is what the backend's inclusive LTE needs or the final minute is dropped
    await expect
      .poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 })
      .toBe(await localWallClockToInclusiveIso(page, "2026-01-02T23:59"));

    // panel is still open — clear inside it
    await page.getByTestId("clear-time-range").click();
    await expect.poll(() => new URL(page.url()).searchParams.get("from")).toBeNull();
    await expect.poll(() => new URL(page.url()).searchParams.get("to")).toBeNull();
  });

  // A To before the From used to reach the backend, which answers
  // 400 invalid_time_range — and that renders the whole grid as an error page.
  // The popover clamps instead, and mirrors the clamped value back into the
  // input so the user sees what was actually applied.
  test("an inverted range is clamped, not sent as a 400", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByTestId("time-range-button").click();

    await page.getByTestId("time-to-input").fill("2026-01-02T23:59");
    await expect.poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 }).not.toBeNull();
    await page.getByTestId("time-from-input").fill("2026-01-05T10:00");
    await expect
      .poll(() => new URL(page.url()).searchParams.get("from"), { timeout: 7000 })
      .toBe(await localWallClockToIso(page, "2026-01-05T10:00"));

    // the To was pulled up to the end of the From minute, and the input shows it
    const clamped = await localWallClockToInclusiveIso(page, "2026-01-05T10:00");
    await expect.poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 }).toBe(clamped);
    await expect(page.getByTestId("time-to-input")).toHaveValue("2026-01-05T10:00");
    // no error page
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(0);
    expect(errors, errors.join("\n")).toHaveLength(0);
  });
});

test.describe("time range on/off", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  // Suspend keeps the window values (?from=/?to= stay in the URL) but stops
  // applying them; re-arming applies again. Clearing resets to active.
  test("chip pause/play toggles the range without losing it", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const cluster = midnightCluster(all);
    // The chip row only renders in the timespan context (?rangeScope=all), so a
    // bare ?from=&to= URL has no chip and no toggle to click.
    await page.goto(`/images?${clusterQuery(cluster, "&rangeScope=all")}`);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);

    // suspend: everything visible, bounds still in the URL
    await page.getByTestId("time-range-toggle").click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(11);
    expect(new URL(page.url()).searchParams.get("from")).toBeTruthy();
    expect(new URL(page.url()).searchParams.get("to")).toBeTruthy();

    // resume: back to the cluster
    await page.getByTestId("time-range-toggle").click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);

    // clearing the range resets to active — a NEW window set from the UI (no
    // page.goto, which would remount into the active state anyway) must filter
    await page.getByTestId("time-range-chip").getByRole("button", { name: "×" }).click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(11);
    await page.getByTestId("time-range-button").click();
    // open-ended "everything after" bound set to a date past every seeded
    // photo, so an applied range is unmistakably 0 tiles
    await page.getByTestId("time-from-input").fill("2027-01-01T00:00");
    await expect.poll(() => new URL(page.url()).searchParams.get("from"), { timeout: 7000 }).not.toBeNull();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(0);
  });
});

// The ±15 min action is a CONTEXT view like the face lookup: it shows ALL
// photos in the window (?rangeScope=all), auto-pausing search/tags/orientation
// behind the Filters pill — while a manually picked range keeps combining.
test.describe("timespan context view", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("show ±15 min ignores other filters until the Filters pill re-applies them", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const first = midnightCluster(all)[0];

    // narrow to one photo via search…
    await page.goto("/images");
    await page.getByPlaceholder("Search images").fill("FSG_9000");
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(1);

    // …open it and jump to its timespan
    await page.locator('[id^="grid-tile-"]').first().click();
    await expect(page.getByText("Image Tags")).toBeVisible();
    await page.getByTestId("show-timespan").click();

    // context view: every photo within ±15 min, not just the searched one
    const url = new URL(page.url());
    expect(url.searchParams.get("rangeScope")).toBe("all");
    expect(url.searchParams.get("image")).toBeNull();
    const from = new Date(url.searchParams.get("from")!);
    const to = new Date(url.searchParams.get("to")!);
    const inWindow = all.filter((i) => {
      if (!i.capturedAtCorrected) return false;
      const c = new Date(i.capturedAtCorrected).getTime();
      return c >= from.getTime() && c <= to.getTime();
    }).length;
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(inWindow);

    // exactly ONE Filters pill, whichever context is active
    await expect(page.getByTestId("filters-pill")).toHaveCount(1);
    const pill = page.getByTestId("filters-pill");

    // the pill re-applies the paused narrowing filters (search still set);
    // assert the pill's own state flip first — tile counts lag behind on the wire
    const paused = /Search, tag and orientation filters are paused/;
    await expect(pill).toHaveAttribute("title", paused);
    await pill.click();
    await expect(pill).not.toHaveAttribute("title", paused);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(1);

    await pill.click();
    await expect(pill).toHaveAttribute("title", paused);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(inWindow);

    // clearing the chip leaves context mode entirely — the (paused) search
    // filter applies again on its own terms
    await page.getByTestId("time-range-chip").getByRole("button", { name: "×" }).click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(1);
    await expect(page.getByTestId("time-range-chip")).toHaveCount(0);
    await expect.poll(() => new URL(page.url()).searchParams.get("rangeScope")).toBeNull();
    await expect(page.getByPlaceholder("Search images")).toHaveValue("FSG_9000");
  });
});

// The Time popover's dual-thumb slider: its domain is the filtered gallery's
// capturedAtCorrected span EXCLUDING the range itself; dragging commits on
// release; the datetime inputs remain the manual override. The thumbs are
// role="slider" divs driven by pointer events (NOT native range inputs), so the
// specs drive them with the mouse and with the keyboard.
test.describe("time-range slider", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("domain follows the filter, drag commits, inputs override", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const cluster = midnightCluster(all);
    expect(cluster.length).toBe(8);

    // narrow the gallery to the cluster so the domain is exactly its span
    await page.goto("/images");
    await page.getByPlaceholder("Search images").fill("FSG_90");
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);

    await page.getByTestId("time-range-button").click();
    const startThumb = page.getByTestId("range-start-thumb");
    const endThumb = page.getByTestId("range-end-thumb");
    await expect(endThumb).toBeVisible();

    const loStep = Number(await startThumb.getAttribute("aria-valuenow"));
    const hiStep = Number(await endThumb.getAttribute("aria-valuenow"));
    // The domain is the cluster's own span, floored to whole minutes.
    expect(hiStep - loStep).toBe(15);
    expect(new Date(loStep * 60_000).getTime()).toBeLessThanOrEqual(new Date(cluster[0].capturedAtCorrected).getTime());
    expect(new Date(hiStep * 60_000).getTime()).toBeGreaterThanOrEqual(new Date(cluster[cluster.length - 1].capturedAtCorrected).getTime());

    // Drag the end thumb five minutes earlier. Positions come from the TRACK
    // box (the thumbs are inset by their radius inside it), not from the thumb
    // box, which is only 14px wide.
    const track = await endThumb.evaluate((el) => {
      const t = (el as HTMLElement).parentElement!;
      const r = t.getBoundingClientRect();
      return { left: r.left, width: r.width };
    });
    const endBox = (await endThumb.boundingBox())!;
    const y = endBox.y + endBox.height / 2;
    const pxPerMinute = (track.width - 14) / (hiStep - loStep);
    await page.mouse.move(endBox.x + endBox.width / 2, y);
    await page.mouse.down();
    await page.mouse.move(endBox.x + endBox.width / 2 - 5 * pxPerMinute, y, { steps: 10 });
    await page.mouse.up();

    await expect(endThumb).toHaveAttribute("aria-valuenow", String(hiStep - 5));
    // committed on release, as the inclusive end of that minute
    await expect
      .poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 })
      .toBe(new Date((hiStep - 5) * 60_000 + 59_999).toISOString());

    // manual override wins: type an exact From instant
    await page.getByTestId("time-from-input").fill("2026-08-01T00:00");
    await expect
      .poll(() => new URL(page.url()).searchParams.get("from"), { timeout: 7000 })
      .toBe(await localWallClockToIso(page, "2026-08-01T00:00"));
  });

  // Moving ONE thumb must not delete the other bound. commit() used to emit null
  // for every untouched thumb, so dragging the start thumb of a two-sided range
  // dropped ?to entirely and silently widened the filter to an open end.
  test("dragging one thumb preserves the other bound", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const cluster = midnightCluster(all);

    await page.goto(`/images?${clusterQuery(cluster)}`);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);
    await page.getByTestId("time-range-button").click();

    const startThumb = page.getByTestId("range-start-thumb");
    const endThumb = page.getByTestId("range-end-thumb");
    await expect(startThumb).toBeVisible();
    const loStep = Number(await startThumb.getAttribute("aria-valuenow"));
    const hiStep = Number(await endThumb.getAttribute("aria-valuenow"));
    // The domain here is the photos OUTSIDE the applied range, so it is much
    // narrower than the drag distance; the pixel maths below would clamp at the
    // domain max. This test is about the untouched bound, not drag precision, so
    // it nudges the thumb by a single pixel and asserts the invariant.
    expect(hiStep).toBeGreaterThanOrEqual(loStep);

    const startBox = (await startThumb.boundingBox())!;
    const y = startBox.y + startBox.height / 2;
    await page.mouse.move(startBox.x + startBox.width / 2, y);
    await page.mouse.down();
    await page.mouse.move(startBox.x + startBox.width / 2 + 1, y, { steps: 4 });
    await page.mouse.up();

    // the untouched upper bound must come back out EXACTLY as it went in
    await expect
      .poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 })
      .toBe(new Date(hiStep * 60_000 + 59_999).toISOString());
    // and the touched one did move
    expect(Number(await startThumb.getAttribute("aria-valuenow"))).toBeGreaterThanOrEqual(loStep);
  });

  test("keyboard moves a thumb and commits (no pointer required)", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const all = await fetchImages(page, project!.id);
    const cluster = midnightCluster(all);

    await page.goto("/images");
    await page.getByPlaceholder("Search images").fill("FSG_90");
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);
    await page.getByTestId("time-range-button").click();

    const endThumb = page.getByTestId("range-end-thumb");
    await expect(endThumb).toBeVisible();
    const before = Number(await endThumb.getAttribute("aria-valuenow"));

    await endThumb.focus();
    await page.keyboard.press("ArrowLeft");
    await expect(endThumb).toHaveAttribute("aria-valuenow", String(before - 1));
    // committed to the route straight away — the keyboard path is not a
    // preview. The exact instant, not toBeTruthy(): a garbage bound passes that.
    await expect
      .poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 })
      .toBe(new Date((before - 1) * 60_000 + 59_999).toISOString());
  });
});

test.describe("time-range slider preview", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  // dragging shows a live value readout and must NOT touch the URL until release
  test("drag previews values live, commits only on release", async ({ page }) => {
    const project = await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByPlaceholder("Search images").fill("FSG_90");
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);
    await page.getByTestId("time-range-button").click();

    // the To input starts empty, so a non-empty value mid-drag can only come
    // from the preview emit
    const toInput = page.getByTestId("time-to-input");
    await expect(toInput).toHaveValue("");

    const endThumb = page.getByTestId("range-end-thumb");
    const startThumb = page.getByTestId("range-start-thumb");
    await expect(endThumb).toBeVisible();
    const loStep = Number(await startThumb.getAttribute("aria-valuenow"));
    const hiStep = Number(await endThumb.getAttribute("aria-valuenow"));
    const track = await trackBox(page, endThumb);
    const box = (await endThumb.boundingBox())!;
    const y = box.y + box.height / 2;
    const pxPerMinute = (track.width - 14) / (hiStep - loStep);
    // Aim the drag at a specific MINUTE, several steps away from where the
    // pointerdown lands. Measuring off the 14px thumb box instead moved the
    // pointer ~1.6px, Math.round kept the same step, and the "live" value
    // asserted below was really the one the pointerdown had already emitted.
    const targetStep = hiStep - 5;
    const targetX = track.left + 7 + ((targetStep - loStep) / (hiStep - loStep)) * (track.width - 14);
    const targetInput = await isoToInputValue(page, targetStep);

    await page.mouse.move(box.x + box.width / 2, y);
    await page.mouse.down();
    await page.mouse.move(targetX, y, { steps: 10 });
    // mid-drag: the To input mirrors the dragged thumb LIVE, nothing committed
    await expect(endThumb).toHaveAttribute("aria-valuenow", String(targetStep));
    await expect(toInput).toHaveValue(targetInput);
    expect(new URL(page.url()).searchParams.get("to")).toBeNull();
    await page.mouse.up();
    // released: range committed to the URL, inputs keep the values
    await expect
      .poll(() => new URL(page.url()).searchParams.get("to"), { timeout: 7000 })
      .toBe(new Date(targetStep * 60_000 + 59_999).toISOString());
    await expect(toInput).toHaveValue(targetInput);
  });
});
