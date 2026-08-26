import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// Time-range gallery filter (?from=/?to=) and the detail-view "show ±15 min"
// action (#117). The seed's midnight cluster (FSG_90xx, 23:55→00:10 event-local
// yesterday, untagged) is the fixture; expected windows are derived from the
// API so the spec stays valid regardless of when/where it runs.

const listOf = (b: any): any[] => b?.items || [];

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

    const from = encodeURIComponent(cluster[0].capturedAtCorrected);
    const to = encodeURIComponent(cluster[cluster.length - 1].capturedAtCorrected);
    await page.goto(`/images?from=${from}&to=${to}`);

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

    // detail closed, window set around the photo, chronological reading order
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
    await expect(tiles).toHaveCount(expected, "window contains exactly the in-bounds photos");
  });

  test("the popover writes the query and clears both sides", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByTestId("time-range-button").click();
    await expect(page.getByTestId("time-from-input")).toBeVisible();
    await page.getByTestId("time-from-input").fill("2026-01-01T00:00");
    await expect.poll(() => new URL(page.url()).searchParams.get("from")).toBeTruthy();
    await page.getByTestId("time-to-input").fill("2026-01-02T23:59");
    await expect.poll(() => new URL(page.url()).searchParams.get("to")).toBeTruthy();

    // panel is still open — clear inside it
    await page.getByRole("button", { name: "Clear time range" }).click();
    await expect.poll(() => new URL(page.url()).searchParams.get("from")).toBeNull();
    await expect.poll(() => new URL(page.url()).searchParams.get("to")).toBeNull();
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
    const from = encodeURIComponent(cluster[0].capturedAtCorrected);
    const to = encodeURIComponent(cluster[cluster.length - 1].capturedAtCorrected);
    await page.goto(`/images?from=${from}&to=${to}`);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);

    // suspend: everything visible, bounds still in the URL
    await page.getByTestId("time-range-toggle").click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(11);
    expect(new URL(page.url()).searchParams.get("from")).toBeTruthy();
    expect(new URL(page.url()).searchParams.get("to")).toBeTruthy();

    // resume: back to the cluster
    await page.getByTestId("time-range-toggle").click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);

    // clearing the range resets to active — a new window must filter again
    await page.getByTestId("time-range-chip").getByRole("button").last().click();
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(11);
    await page.goto(`/images?from=${from}&to=${to}`);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(8);
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
    await expect(page.getByTestId("filters-pill")).toBeVisible();

    // the pill re-applies the paused narrowing filters (search still set);
    // assert the pill's own state flip first — tile counts lag behind on the wire
    const pill = page.getByTestId("filters-pill");
    const pausedClass = /border-primary-300/;
    await pill.click();
    await expect(pill).not.toHaveClass(pausedClass);
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(1);

    await pill.click();
    await expect(pill).toHaveClass(pausedClass);
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
