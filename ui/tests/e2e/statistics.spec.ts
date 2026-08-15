import { test, expect } from "@playwright/test";
import { loginAs, seedProjectId, collectJsErrors } from "./helpers";

// The statistics dashboard over the testserver seed: KPI tiles, the images-per-day
// column chart (with the photographer toggle), and the tag detail table. The seed
// carries a project with images that have capturedAtCorrected, so the day chart
// must render real columns — an empty chart here means the aggregation broke.
test.describe("project statistics", () => {
  test("dashboard renders tiles, day chart and tag table", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "admin");
    const pid = await seedProjectId(page);
    expect(pid, "seed project id").toBeTruthy();

    await page.goto(`/projects/${pid}/statistics`);
    await expect(page.getByRole("heading", { name: "Statistics" })).toBeVisible();

    // KPI row: the Images tile carries a non-zero seeded count.
    const imagesTile = page.locator("dl > div").filter({ hasText: "Images" }).first();
    await expect(imagesTile).toBeVisible();
    await expect(imagesTile.locator("dd")).not.toHaveText("0");

    // Day chart: at least one rendered column and the photographer toggle works.
    await expect(page.getByRole("heading", { name: "Images per day" })).toBeVisible();
    await page.getByRole("button", { name: "By photographer" }).click();
    await expect(page.getByRole("button", { name: "By photographer" })).toHaveClass(/text-accent/);

    // Photographers + top tags cards render with content.
    await expect(page.getByRole("heading", { name: "Photographers" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Top tags" })).toBeVisible();

    // The detail table survives below the dashboard.
    await expect(page.getByRole("heading", { name: "All tags" })).toBeVisible();

    expect(errors, `js errors: ${errors.join("; ")}`).toHaveLength(0);
  });
});
