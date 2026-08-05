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

    await page.keyboard.press("z"); // detail → zen
    await expect(zen).toBeVisible();
    await expect(zen.locator('span:text-is("Default")')).toBeVisible(); // seeded EXIF tag in the bottom bar

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
});
