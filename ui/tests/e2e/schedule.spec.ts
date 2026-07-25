import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// S15 e2e: the schedule tab end-to-end — the projectAdmin defines a pool item
// through the dialog, a photographer pulls it into their schedule (occupancy
// goes empty -> full), the "My schedule" scope filters, and the admin cleans
// up. Self-cleaning; runs serially against the shared dev stack.
test.describe.serial("schedule module", () => {
  const TITLE = "E2E Endurance";

  // Today at fixed local hours, as datetime-local strings.
  const day = new Date().toISOString().slice(0, 10);
  const START = `${day}T09:00`;
  const END = `${day}T11:00`;

  test("projectAdmin creates a schedule item via the dialog", async ({ page }) => {
    const errors = collectJsErrors(page);
    const project = await loginAs(page, "projectAdmin");
    expect(project, "seed project should be active").not.toBeNull();

    await page.goto("/schedule");
    await page.getByRole("button", { name: "Add item" }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Add schedule item")).toBeVisible();
    await dialog.getByLabel("Title").fill(TITLE);
    await dialog.getByLabel("Start").fill(START);
    await dialog.getByLabel("End").fill(END);
    await dialog.getByLabel("Cardinality").fill("1");
    await dialog.getByRole("button", { name: "Add item" }).click();
    await expect(dialog).toBeHidden();

    // The item renders in the calendar and survives a reload (backend persisted).
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();
    await page.reload();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("photographer pulls the item into their schedule and the scope filters", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "projectEditor");

    await page.goto("/schedule");
    // "Everything" scope shows the pool; the persisted scope may still be "mine"
    // from a previous run, so switch explicitly.
    await page.getByRole("button", { name: "Everything" }).click();
    const item = page.getByRole("button").filter({ hasText: TITLE });
    await expect(item).toBeVisible();

    // Join through the dialog (transcript: "Add to my schedule").
    await item.click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "Add to my schedule" }).click();
    await expect(dialog.getByText("Fully covered")).toBeVisible(); // cardinality 1 reached
    await dialog.getByRole("button", { name: "Close" }).last().click();

    // "My schedule" now contains it.
    await page.getByRole("button", { name: "My schedule" }).click();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();

    // Leave again — the mine scope empties.
    await page.getByRole("button").filter({ hasText: TITLE }).click();
    await dialog.getByRole("button", { name: "Leave" }).click();
    await dialog.getByRole("button", { name: "Close" }).last().click();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("projectAdmin deletes the item again", async ({ page }) => {
    await loginAs(page, "projectAdmin");
    await page.goto("/schedule");
    await page.getByRole("button", { name: "Everything" }).click();

    const item = page.getByRole("button").filter({ hasText: TITLE });
    await expect(item).toBeVisible();
    await item.click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toHaveCount(0);
  });
});

// The upload page rebuild: dropzone -> timeline -> tile grid. The seeded
// upload has three images, so the grid and the (empty) timeline must render
// without JS errors for the owner.
test("upload detail page shows timeline and tile grid", async ({ page }) => {
  const errors = collectJsErrors(page);
  await loginAs(page, "projectEditor");

  await page.goto("/uploads");
  // Navigate to the seeded upload via the API to stay layout-independent.
  const uploadId = await page.evaluate(async () => {
    const r = await fetch("/api/v1/uploads?limit=1", { credentials: "include" });
    const b = await r.json();
    return b?.items?.[0]?.id ?? null;
  });
  expect(uploadId).not.toBeNull();

  await page.goto(`/uploads/${uploadId}/edit`);
  await expect(page.getByRole("heading", { name: "Tagging timeline" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Images", exact: true })).toBeVisible();
  // The three seeded images render as tiles.
  await expect(page.locator("figure")).toHaveCount(3, { timeout: 10_000 });

  expect(errors, errors.join("\n")).toHaveLength(0);
});
