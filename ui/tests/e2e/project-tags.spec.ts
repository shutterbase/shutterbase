import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// End-to-end CRUD through the migrated TagDialog + Table: create a tag, see it in
// the table, then delete it (deletion is immediate, no confirm). Self-cleaning.
test.describe.serial("project tags CRUD", () => {
  const TAG = "E2E Smoke Tag";

  test("create a tag via the dialog, then delete it (admin)", async ({ page }) => {
    const errors = collectJsErrors(page);
    const project = await loginAs(page, "admin");
    expect(project, "seed project should be active").not.toBeNull();

    await page.goto(`/projects/${project!.id}/tags`);
    await expect(page.getByRole("button", { name: /Add Project Tag/i })).toBeVisible();

    // --- create ---
    await page.getByRole("button", { name: /Add Project Tag/i }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Add tag")).toBeVisible();
    const textboxes = dialog.getByRole("textbox");
    await textboxes.nth(0).fill(TAG); // Name
    await textboxes.nth(1).fill("created by e2e"); // Description
    await dialog.getByRole("button", { name: "Save tag" }).click();

    // dialog closes, new row appears
    await expect(dialog).toBeHidden();
    const row = page.getByRole("row").filter({ hasText: TAG });
    await expect(row).toBeVisible();

    // --- delete (immediate) ---
    await row.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByRole("row").filter({ hasText: TAG })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });
});
