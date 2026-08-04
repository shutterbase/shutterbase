import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// The upload-batch filter on the gallery (?upload=) and its two entry points:
// the "View in images" button on the upload edit page and the photo icon on a
// kanban card. All three funnel into the same route query, which Images.vue
// maps onto the images list request.

async function seedUploadId(page: Page, projectId: string): Promise<string | null> {
  return page.evaluate(async (pid) => {
    const r = await fetch(`/api/v1/uploads?projectId=${pid}&limit=50`, { credentials: "include" });
    const b = await r.json().catch(() => ({}));
    return (b?.items || []).find((u: any) => u.name === "seed upload")?.id ?? null;
  }, projectId);
}

test.describe("upload batch filter", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("gallery picker filters by upload and clears again", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images");
    await expect(page.getByPlaceholder("Search images")).toBeVisible();

    await page.getByRole("button", { name: "Upload", exact: true }).click();
    await expect(page.getByRole("option", { name: "All uploads" })).toBeVisible();
    await page.getByRole("option", { name: "seed upload" }).click();

    await expect(page).toHaveURL(/[?&]upload=/);
    // button now carries the selected batch name and the filtered grid has rows
    await expect(page.getByRole("button", { name: "seed upload" })).toBeVisible();
    await expect(page.locator('[id^="grid-tile-"]').first()).toBeVisible();

    await page.getByRole("button", { name: "seed upload" }).click();
    await page.getByRole("option", { name: "All uploads" }).click();
    await expect.poll(() => page.url()).not.toMatch(/[?&]upload=/);
  });

  test("upload edit page links into the filtered gallery", async ({ page }) => {
    const project = await loginAs(page, "admin");
    expect(project).not.toBeNull();
    const uploadId = await seedUploadId(page, project!.id);
    expect(uploadId, "seeded upload must exist").not.toBeNull();

    await page.goto(`/uploads/${uploadId}/edit`);
    await page.getByRole("link", { name: "View in images" }).click();

    await expect(page).toHaveURL(new RegExp(`/images\\?.*upload=${uploadId}`));
    await expect(page.getByRole("button", { name: "seed upload" })).toBeVisible();
    await expect(page.locator('[id^="grid-tile-"]').first()).toBeVisible();
  });

  test("kanban card links into the filtered gallery", async ({ page }) => {
    const project = await loginAs(page, "admin");
    expect(project).not.toBeNull();
    const uploadId = await seedUploadId(page, project!.id);

    const setReview = (enabled: boolean) =>
      page.evaluate(
        async ({ pid, enabled }) => {
          const r = await fetch(`/api/v1/projects/${pid}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ uploadReviewEnabled: enabled }),
          });
          if (!r.ok) throw new Error(`project update: ${r.status}`);
          // the store reads uploadReviewEnabled off the persisted active project,
          // so pin the fresh full object — {id,name} alone hides the kanban
          localStorage.setItem("activeProject", JSON.stringify(await r.json()));
        },
        { pid: project!.id, enabled },
      );

    await setReview(true);
    try {
      await page.goto("/uploads");
      const card = page.locator("article").filter({ hasText: "seed upload" }).first();
      await expect(card).toBeVisible();
      await card.getByRole("link", { name: "Show this upload's images" }).click();

      await expect(page).toHaveURL(new RegExp(`/images\\?.*upload=${uploadId}`));
      await expect(page.locator('[id^="grid-tile-"]').first()).toBeVisible();
    } finally {
      await setReview(false);
    }
  });
});
