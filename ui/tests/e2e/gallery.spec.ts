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
    // including a tag surfaces the "Clear N selected" affordance
    await page.getByRole("button", { name: "Include Podium" }).click();
    await expect(page.getByText(/Clear 1 selected/)).toBeVisible();
  });

  test("tag filter pins include/exclude chips above the list, immune to the text filter", async ({ page }) => {
    await page.getByRole("button", { name: "Tags" }).click();
    await page.getByRole("button", { name: "Exclude Podium" }).click();
    await page.getByRole("button", { name: "Include Default" }).click();

    // both chips pinned; a selected tag leaves the pick list below
    const podiumChip = page.getByRole("button", { name: "Remove filter Podium" });
    const defaultChip = page.getByRole("button", { name: "Remove filter Default" });
    await expect(podiumChip).toBeVisible();
    await expect(defaultChip).toBeVisible();
    await expect(page.getByRole("button", { name: "Include Podium" })).toHaveCount(0);

    // the text filter narrows the pick list but never the pinned chips
    await page.getByPlaceholder("Filter tags…").fill("zzz-no-such-tag");
    await expect(page.getByText("No tags found")).toBeVisible();
    await expect(podiumChip).toBeVisible();
    await expect(defaultChip).toBeVisible();

    // chip click removes a single filter; the tag returns to the pick list
    await page.getByPlaceholder("Filter tags…").fill("");
    await podiumChip.click();
    await expect(podiumChip).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Include Podium" })).toBeVisible();
    await expect(page.getByText(/Clear 1 selected/)).toBeVisible();
  });

  test("image detail groups tags into category rows", async ({ page }) => {
    await page.locator('[id^="grid-tile-"]').first().click();
    await expect(page.getByText("Image Tags")).toBeVisible();
    // seeded images carry the Default tag (type default) → it renders under the
    // "template" category row with its caption
    await expect(page.getByText("template", { exact: true })).toBeVisible();
    await expect(page.getByText("Default", { exact: true })).toBeVisible();
  });

  test("detail sidebar collapses details behind more, revealing the applied offset", async ({ page }) => {
    await page.locator('[id^="grid-tile-"]').first().click();
    await expect(page.getByText("Corrected capture time")).toBeVisible();
    await expect(page.getByText("Updated", { exact: true })).toBeVisible();
    await expect(page.getByText("Original file name")).toBeHidden();
    await page.getByText("more", { exact: true }).click();
    await expect(page.getByText("Original file name")).toBeVisible();
    await expect(page.getByText("Applied time offset")).toBeVisible();
    await page.getByText("less", { exact: true }).click();
    await expect(page.getByText("Original file name")).toBeHidden();
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
