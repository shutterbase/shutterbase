import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

// The viewer posts the picked file to /api/v1/exif/inspect (in-memory exiftool
// read, nothing stored) and renders key facts + all raw groups.
test.describe("exif viewer", () => {
  test("inspects a local image and shows key facts and raw groups", async ({ page }) => {
    await loginAs(page, "user", { activate: false });
    await page.goto("/exif-viewer");
    await expect(page.getByTestId("exif-drop-zone")).toBeVisible();

    await page.locator('input[type="file"]').setInputFiles("tests/e2e/fixtures/20240817-0B8A0042.jpg");

    await expect(page.getByTestId("exif-key-facts")).toBeVisible();
    await expect(page.getByTestId("exif-file-name")).toHaveText("20240817-0B8A0042.jpg");
    // real camera JPEG: fine-grained groups must be present in the accordion
    const groups = page.getByTestId("exif-raw-groups");
    await expect(groups.locator("summary", { hasText: "ExifIFD" })).toBeVisible();
    // groups start collapsed; expanding one reveals its tag table (the fixture
    // is a stripped 5 KB thumbnail — DateTimeOriginal survives, exposure data
    // doesn't). Exact match: Composite carries SubSecDateTimeOriginal, which a
    // substring locator would resolve to first — inside a collapsed group.
    await groups.locator("summary", { hasText: "ExifIFD" }).click();
    await expect(groups.getByRole("cell", { name: "DateTimeOriginal", exact: true })).toBeVisible();
  });

  test("requires authentication", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/exif-viewer");
    await expect(page).toHaveURL(/\/login/);
  });
});
