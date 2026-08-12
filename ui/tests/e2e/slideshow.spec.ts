import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

// Slideshow over the current grid view: setup dialog -> playing overlay with
// controls; images auto-advance on the configured show time.
test.describe("slideshow", () => {
  test("configures, plays, pauses and exits", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await expect(page.getByTestId("start-slideshow")).toBeVisible();
    await page.getByTestId("start-slideshow").click();

    // setup phase with config; use a short show time so auto-advance is testable
    await expect(page.getByTestId("slideshow-setup")).toBeVisible();
    await page.getByTestId("slideshow-show-seconds").fill("1");
    await page.getByTestId("slideshow-start").click();

    const position = page.getByTestId("slideshow-position");
    await expect(position).toContainText("1 /");
    // auto-advances past the first slide (1s show + transition)
    await expect(position).not.toContainText("1 /", { timeout: 10_000 });

    // pause freezes the position
    await page.getByTestId("slideshow-overlay").hover();
    await page.getByTestId("slideshow-play-pause").click();
    const frozen = await position.textContent();
    await page.waitForTimeout(2500);
    expect(await position.textContent()).toBe(frozen);

    // exit returns to the grid — wake the auto-faded controls first. The DEV
    // quick-actions bubble (z-9999, dev builds only) covers the button's hit
    // point, so a positional click lands on the bubble; dispatch straight to
    // the button instead — prod has no bubble.
    await page.mouse.move(300, 300);
    await expect(page.getByTestId("slideshow-controls")).toBeVisible();
    await page.getByTestId("slideshow-exit").dispatchEvent("click");
    await expect(page.getByTestId("slideshow-overlay")).toHaveCount(0);
  });

  // The seed has 3 images, the last one tagged "internal" — the grid shows all
  // three, the slideshow only ever counts and plays two.
  test("skips internal images", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await expect(page.locator('[id^="grid-tile-"]')).toHaveCount(3);
    await page.getByTestId("start-slideshow").click();
    await expect(page.getByTestId("slideshow-setup")).toContainText("2 images in the current view");
    await page.getByTestId("slideshow-start").click();
    await expect(page.getByTestId("slideshow-position")).toContainText("1 / 2");
  });

  test("never puts a scrollbar on the page", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByTestId("start-slideshow").click();
    // strong pan & zoom scales the slide past the viewport — the classic way to
    // grow a scrollbar
    await page.getByRole("button", { name: "Strong" }).click();
    await page.getByTestId("slideshow-start").click();
    await expect(page.getByTestId("slideshow-position")).toBeVisible();

    const viewport = await page.evaluate(() => ({
      verticalScrollbar: window.innerWidth - document.documentElement.clientWidth,
      horizontalScrollbar: window.innerHeight - document.documentElement.clientHeight,
      documentOverflow: getComputedStyle(document.body).overflow,
    }));
    expect(viewport).toEqual({ verticalScrollbar: 0, horizontalScrollbar: 0, documentOverflow: "hidden" });
  });

  test("escape closes the slideshow", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.getByTestId("start-slideshow").click();
    await expect(page.getByTestId("slideshow-setup")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByTestId("slideshow-overlay")).toHaveCount(0);
  });
});
