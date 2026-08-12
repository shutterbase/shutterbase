import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// Keyboard tagging flow (double-enter), zen mode (z) and detail-view zoom.
// Exercised as admin with the seed project active.

const imageParam = (page: Page) => new URL(page.url()).searchParams.get("image");
// badge spans only — the (v-show hidden) tagging dialog lists tag names in <p>s
const visibleBadge = (page: Page, name: string) => page.locator(`span:text-is("${name}"):visible`);

test.describe("detail view keyboard flow", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
    await loginAs(page, "admin");
    await page.goto("/images");
    await page.locator('[id^="grid-tile-"]').first().click();
    await expect(page.getByText("Image Tags")).toBeVisible();
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("double-enter advances without applying the top recent tag", async ({ page }) => {
    const uniq = `e2e-zen-${Date.now()}`;
    const first = imageParam(page);

    // apply a fresh custom tag so the recent-tags list is non-empty
    await page.keyboard.press("t");
    const search = page.getByPlaceholder("Search tag...");
    await expect(search).toBeVisible();
    await search.fill(uniq);
    await page.getByRole("button", { name: /Create custom tag/ }).click();
    await expect(visibleBadge(page, uniq)).toHaveCount(1); // sidebar badge of image 1

    // Enter on the empty search box → next image, dialog closed
    await page.keyboard.press("Enter");
    await expect.poll(() => imageParam(page)).not.toBe(first);
    const second = imageParam(page);
    await expect(search).toBeHidden();

    // regression: reopening the dialog preselects nothing, so Enter advances
    // instead of applying the recent tag
    await page.keyboard.press("t");
    await expect(page.getByText("Recent tags")).toBeVisible();
    await page.keyboard.press("Enter");
    await expect.poll(() => imageParam(page)).not.toBe(second);
    await expect(visibleBadge(page, uniq)).toHaveCount(0); // image 2 stayed untagged

    // arrows opt into the recent list: ArrowDown + Enter applies the tag
    await page.keyboard.press("t");
    await expect(page.getByText("Recent tags")).toBeVisible();
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Enter");
    await expect(visibleBadge(page, uniq)).toHaveCount(1);
    await page.keyboard.press("Escape");
  });

  test("z toggles zen over either view, g stays grid/detail only", async ({ page }) => {
    const zen = page.getByTestId("zen-overlay");
    const first = imageParam(page);

    // apply a custom tag first — zen must list it too (all tags, not just the EXIF subset)
    const uniq = `e2e-zenall-${Date.now()}`;
    await page.keyboard.press("t");
    const search = page.getByPlaceholder("Search tag...");
    await expect(search).toBeVisible();
    await search.fill(uniq);
    await page.getByRole("button", { name: /Create custom tag/ }).click();
    await expect(visibleBadge(page, uniq)).toHaveCount(1);
    await page.keyboard.press("Escape");

    await page.keyboard.press("z"); // detail → zen
    await expect(zen).toBeVisible();
    // header: centered logo; footer: file name left, tags middle, position right
    await expect(zen.locator('img[alt="shutterbase"]')).toBeVisible();
    await expect(zen.locator('span:text-is("Default")')).toBeVisible(); // seeded EXIF tag
    await expect(zen.locator(`span:text-is("${uniq}")`)).toBeVisible(); // custom tag, not EXIF-exported
    await expect(zen.locator("span").filter({ hasText: /\.jpg$/i }).first()).toBeVisible();
    await expect(zen.locator("span").filter({ hasText: /^\d+ \/ \d+$/ })).toBeVisible();

    await page.keyboard.press("g"); // guarded while zen is active
    await expect(zen).toBeVisible();
    expect(imageParam(page)).toBe(first);

    await page.keyboard.press("z"); // zen → back to detail
    await expect(zen).toBeHidden();
    expect(imageParam(page)).toBe(first);

    await page.keyboard.press("g"); // detail → grid
    await expect.poll(() => imageParam(page)).toBeNull();

    await page.keyboard.press("z"); // grid → zen
    await expect(zen).toBeVisible();
    await page.keyboard.press("z"); // zen → back to grid
    await expect(zen).toBeHidden();
    expect(imageParam(page)).toBeNull();
  });

  test("double-click zooms the detail image and resets again", async ({ page }) => {
    const hero = page.locator("figure img").first();
    const transformed = hero.locator("xpath=..");
    await hero.dblclick();
    await expect(transformed).toHaveAttribute("style", /scale\(2\.5\)/);
    await hero.dblclick();
    await expect(transformed).toHaveAttribute("style", /scale\(1\)/);
  });

  test("a zoomed image pans half a stage past the old bound, clearing the sidebar", async ({ page }) => {
    const hero = page.locator("figure img").first();
    const transformed = hero.locator("xpath=..");
    await hero.dblclick();
    await expect(transformed).toHaveAttribute("style", /scale\(2\.5\)/);

    // The stage is where the image is clipped while zoomed; the img keeps its
    // frozen fitted size, so the scaled width is that times the zoom.
    const stage = await hero.evaluate((img: HTMLImageElement) => ({
      width: (img.closest(".overflow-hidden") as HTMLElement).clientWidth,
      scaledWidth: img.offsetWidth * 2.5,
    }));

    // drag right, repeatedly and well past the old bound. The stage spans the
    // full window, so every press can start at the same point — kept clear of
    // the sidebar, which sits above the stage and would swallow the pointer.
    const { width, height } = page.viewportSize() ?? { width: 1440, height: 900 };
    for (let i = 0; i < 4; i++) {
      await page.mouse.move(width * 0.35, height / 2);
      await page.mouse.down();
      await page.mouse.move(width - 40, height / 2, { steps: 8 });
      await page.mouse.up();
    }

    // Panning used to stop where the content still covered the stage; it now
    // runs half a stage further, so the image can be dragged out from under the
    // detail sidebar.
    const translateX = (style: string | null) => Number(/translate\((-?[\d.]+)px/.exec(style ?? "")?.[1] ?? NaN);
    const oldBound = Math.max(0, stage.width - stage.scaledWidth);
    await expect.poll(async () => translateX(await transformed.getAttribute("style"))).toBeCloseTo(oldBound + stage.width / 2, 0);
  });

  test("switching images resets zoom, unless ctrl/cmd is held", async ({ page }) => {
    const hero = page.locator("figure img").first();
    const transformed = hero.locator("xpath=..");
    await hero.dblclick();
    await expect(transformed).toHaveAttribute("style", /scale\(2\.5\)/);

    await page.keyboard.press("ArrowRight"); // plain switch → fitted once loaded
    await expect(transformed).toHaveAttribute("style", /scale\(1\)/);

    await hero.dblclick();
    await expect(transformed).toHaveAttribute("style", /scale\(2\.5\)/);
    const before = imageParam(page);
    await page.keyboard.press("Control+ArrowLeft"); // modifier switch → zoom kept
    await expect.poll(() => imageParam(page)).not.toBe(before);
    await expect(transformed).toHaveAttribute("style", /scale\(2\.5\)/);
  });
});
