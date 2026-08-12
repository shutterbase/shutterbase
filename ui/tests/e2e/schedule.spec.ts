import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// S15 e2e: the schedule tab end-to-end — the projectAdmin defines a pool item
// through the dialog, a photographer claims it via the assignment popover
// (occupancy empty -> full), the admin assigns/removes people through the
// popover + modal, and edits live behind the pen icon. Self-cleaning; runs
// serially against the shared dev stack.
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

    // Tag suggestions are picked through the searchable combobox (hundreds of
    // tags in a real project — no chip cloud): type, pick, chip appears.
    await dialog.getByPlaceholder(/Search tags/).fill("podi");
    await dialog.getByRole("option", { name: /Podium/ }).click();
    await expect(dialog.getByRole("button", { name: "Remove tag Podium" })).toBeVisible();

    await dialog.getByRole("button", { name: "Add item" }).click();
    await expect(dialog).toBeHidden();

    // The item renders in the calendar and survives a reload (backend persisted).
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();
    await page.reload();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("photographer claims the item via the popover and the scope filters", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "projectEditor");

    await page.goto("/schedule");
    // The persisted scope may still be "mine" from a previous run.
    await page.getByRole("button", { name: "Everything" }).click();
    const item = page.getByRole("button").filter({ hasText: TITLE });
    await expect(item).toBeVisible();

    // A photographer gets NO edit pen — that's the admin's.
    await expect(item.getByRole("button", { name: `Edit ${TITLE}` })).toHaveCount(0);

    // Normal click opens the assignment popover, not the edit dialog.
    await item.click();
    const pop = page.getByTestId("schedule-popover");
    await expect(pop).toBeVisible();
    await expect(pop.getByText("Nobody yet — be the first.")).toBeVisible();
    await pop.getByRole("button", { name: "Add to my schedule" }).click();
    await expect(pop.getByText("Fully covered")).toBeVisible(); // cardinality 1 reached
    await page.keyboard.press("Escape");
    await expect(pop).toBeHidden();

    // "My schedule" now contains it.
    await page.getByRole("button", { name: "My schedule" }).click();
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toBeVisible();

    // Leave again — the mine scope empties.
    await page.getByRole("button").filter({ hasText: TITLE }).click();
    await pop.getByRole("button", { name: "Leave" }).click();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("button").filter({ hasText: TITLE })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("projectAdmin assigns and removes a photographer through the popover", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "projectAdmin");
    await page.goto("/schedule");
    await page.getByRole("button", { name: "Everything" }).click();

    const item = page.getByRole("button").filter({ hasText: TITLE });
    await item.click();
    const pop = page.getByTestId("schedule-popover");
    await expect(pop.getByText("Nobody yet — be the first.")).toBeVisible();

    // Assign via the modal (search + pick).
    await pop.getByRole("button", { name: "Assign" }).click();
    const modal = page.getByRole("dialog");
    await expect(modal.getByText("Assign photographer")).toBeVisible();
    await modal.getByLabel("Search members").fill("");
    const firstCandidate = modal.getByRole("button").first();
    const candidateName = (await firstCandidate.innerText()).trim();
    await firstCandidate.click();
    await expect(modal).toBeHidden();
    await expect(pop.getByText(candidateName)).toBeVisible();

    // Remove again via the row's x.
    await pop.getByRole("button", { name: `Remove ${candidateName}` }).click();
    await expect(pop.getByText("Nobody yet — be the first.")).toBeVisible();
    await page.keyboard.press("Escape");

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("projectAdmin edits via the pen and deletes the item", async ({ page }) => {
    await loginAs(page, "projectAdmin");
    await page.goto("/schedule");
    await page.getByRole("button", { name: "Everything" }).click();

    const item = page.getByRole("button").filter({ hasText: TITLE });
    await expect(item).toBeVisible();
    await item.getByRole("button", { name: `Edit ${TITLE}` }).click();

    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Edit schedule item")).toBeVisible();
    // The tag suggestion picked at creation persisted and renders as a chip.
    await expect(dialog.getByRole("button", { name: "Remove tag Podium" })).toBeVisible();
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

  // Add a tag lane through the searchable picker; the lane spans the full
  // image axis and, being user-added, carries a delete "x".
  await page.getByPlaceholder(/Add tag lane/).fill("podi");
  await page.getByRole("option", { name: /Podium/ }).click();
  const removeLane = page.getByRole("button", { name: "Remove lane Podium" });
  await expect(removeLane).toBeVisible();

  // Applied lanes must come back labelled: the persisted timeline carries only
  // tag ids, so a reload that seeds before the tag list arrives used to render
  // every lane as the generic "Tag" placeholder.
  await page.getByRole("button", { name: "Apply tags" }).click();
  await expect(page.getByText(/Tags applied/)).toBeVisible();
  await page.reload();
  await expect(page.getByRole("button", { name: "Remove lane Podium" })).toBeVisible();

  // Drop the lane again and apply, so the upload is left as seeded.
  await page.getByRole("button", { name: "Remove lane Podium" }).click();
  await page.getByRole("button", { name: "Apply tags" }).click();
  await expect(page.getByRole("button", { name: "Remove lane Podium" })).toHaveCount(0);

  expect(errors, errors.join("\n")).toHaveLength(0);
});
