import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// End-to-end CRUD through the migrated TagDialog + Table: create a tag, see it in
// the table, then delete it (deletion is immediate, no confirm). Self-cleaning.
test.describe.serial("project tags CRUD", () => {
  const TAG = "E2E Smoke Tag";
  // "$" prefixed: that is what makes a template tag render at image-create time.
  // Deleted at the end of its test so it cannot default-tag images in later specs.
  const TEMPLATE_TAG = "$E2ETEMPLATE";

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
    await dialog.getByLabel("Name", { exact: true }).fill(TAG);
    await dialog.getByLabel("Description", { exact: true }).fill("created by e2e");
    await dialog.getByLabel(/^Order/).fill("7");
    await dialog.getByRole("button", { name: "Save tag" }).click();
    await expect(dialog).toBeHidden();

    // new row appears AND survives a reload — proves the backend create persisted,
    // not just optimistic UI state.
    await expect(page.getByRole("row").filter({ hasText: TAG })).toBeVisible();
    await page.reload();
    const row = page.getByRole("row").filter({ hasText: TAG });
    await expect(row, "created tag must persist after reload").toBeVisible();
    await expect(row, "and keep its order rank").toContainText("7");

    // --- delete ---
    await row.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByRole("row").filter({ hasText: TAG })).toHaveCount(0);
    // stays gone after reload — proves the backend DELETE persisted (guards the
    // fire-and-forget regression where optimistic removal hides a failed delete).
    await page.reload();
    await expect(page.getByRole("row").filter({ hasText: TAG }), "deleted tag must stay gone after reload").toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  // Display name: shown wherever the tag is rendered, name is the fallback.
  // Create with one, see it in the table; clear it via edit, see the name again.
  test("create a tag with a display name, clear it, see the name fallback (admin)", async ({ page }) => {
    const errors = collectJsErrors(page);
    const NAME = "e2e-displayname-tag";
    const DISPLAY = "E2E Pretty Label";
    const project = await loginAs(page, "admin");
    expect(project, "seed project should be active").not.toBeNull();

    await page.goto(`/projects/${project!.id}/tags`);
    await page.getByRole("button", { name: /Add Project Tag/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name", { exact: true }).fill(NAME);
    await dialog.getByLabel(/^Display name/).fill(DISPLAY);
    await dialog.getByLabel("Description", { exact: true }).fill("created by e2e");
    await dialog.getByRole("button", { name: "Save tag" }).click();
    await expect(dialog).toBeHidden();

    await page.reload();
    const row = page.getByRole("row").filter({ hasText: NAME });
    await expect(row, "display name must persist after reload").toContainText(DISPLAY);

    // --- clear the display name ---
    await row.getByRole("button", { name: "Edit" }).click();
    await dialog.getByLabel(/^Display name/).fill("");
    await dialog.getByRole("button", { name: "Save tag" }).click();
    await expect(dialog).toBeHidden();
    await page.reload();
    await expect(page.getByRole("row").filter({ hasText: NAME }), "cleared display name must be gone").not.toContainText(DISPLAY);

    await page
      .getByRole("row")
      .filter({ hasText: NAME })
      .getByRole("button", { name: "Delete" })
      .click();
    await page.reload();
    await expect(page.getByRole("row").filter({ hasText: NAME })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  // Regression: the dialog offered type=template while the API refused it
  // outright ("template tags are not creatable via the API"), so a project admin
  // could never add $COPYRIGHT/$WEEKDAY — the "$" prefix IS how a template tag is
  // declared, and the seed/import was the only way one could exist.
  test("create a $-prefixed template tag via the dialog, then delete it (admin)", async ({ page }) => {
    const errors = collectJsErrors(page);
    const project = await loginAs(page, "admin");
    expect(project, "seed project should be active").not.toBeNull();

    await page.goto(`/projects/${project!.id}/tags`);
    await page.getByRole("button", { name: /Add Project Tag/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name", { exact: true }).fill(TEMPLATE_TAG);
    await dialog.getByLabel("Description", { exact: true }).fill("template created by e2e");
    await dialog.getByLabel("Type", { exact: true }).selectOption("template");
    await dialog.getByRole("button", { name: "Save tag" }).click();
    await expect(dialog, "the API must accept a template tag, not 400 the dialog open").toBeHidden();

    await page.reload();
    const row = page.getByRole("row").filter({ hasText: TEMPLATE_TAG });
    await expect(row, "template tag must persist after reload").toBeVisible();
    await expect(row, "and keep its type").toContainText("template");

    await row.getByRole("button", { name: "Delete" }).click();
    await page.reload();
    await expect(page.getByRole("row").filter({ hasText: TEMPLATE_TAG })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });
});
