import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// The gallery toolbar is the product's signature surface: density modes, search,
// tag filter, sort and orientation. Exercised as admin with the seed project active.
test.describe("gallery toolbar", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
    await loginAs(page, "admin"); // activates seed project + primes projectTags
    await page.goto("/images");
    await expect(page.getByPlaceholder("Search images")).toBeVisible();
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("density control switches between modes", async ({ page }) => {
    const density = () => page.evaluate(() => localStorage.getItem("image-grid-density"));
    await page.locator('button[title="Gallery view"]').click();
    await expect.poll(density).toBe("gallery");
    await page.locator('button[title="Dense view"]').click();
    await expect.poll(density).toBe("dense");
    await page.locator('button[title="Grid view"]').click();
    await expect.poll(density).toBe("comfortable");
  });

  test("search field accepts input and clears via the X button", async ({ page }) => {
    const search = page.getByPlaceholder("Search images");
    await search.fill("podium");
    await expect(search).toHaveValue("podium");
    const clear = page.getByRole("button", { name: "Clear search" });
    await expect(clear).toBeVisible();
    await clear.click();
    await expect(search).toHaveValue("");
  });

  test("tags filter lists project tags (template excluded) and tracks selection", async ({ page }) => {
    await page.getByRole("button", { name: "Tags" }).click();
    await expect(page.getByPlaceholder("Filter tags…")).toBeVisible();
    await expect(page.getByText("Podium", { exact: true })).toBeVisible();
    await expect(page.getByText("Default", { exact: true })).toBeVisible();
    // $DATE is a template tag and must not appear in the filter
    await expect(page.getByText("$DATE")).toHaveCount(0);
    // toggling a tag surfaces the "Clear N selected" affordance
    await page.getByRole("button", { name: /Podium/ }).click();
    await expect(page.getByText(/Clear 1 selected/)).toBeVisible();
  });

  test("sort listbox opens with ordering options", async ({ page }) => {
    await page.getByRole("button", { name: /Latest first/ }).click();
    await expect(page.getByRole("option", { name: /Oldest first/ })).toBeVisible();
  });

  test("orientation listbox opens with portrait/landscape options", async ({ page }) => {
    await page.getByRole("button", { name: /All orientations/ }).click();
    await expect(page.getByRole("option", { name: "Portrait" })).toBeVisible();
    await expect(page.getByRole("option", { name: "Landscape" })).toBeVisible();
  });
});
