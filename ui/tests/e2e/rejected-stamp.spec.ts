import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// The rejected stamp: applying the reserved "rejected" tag slams an oversized
// stamp onto the on-screen image; it stays until the user moves to another
// image. Needs the review flow, so the spec toggles it on the seed project
// and restores everything afterwards — other specs assume review is off.

const imageParam = (page: Page) => new URL(page.url()).searchParams.get("image");

async function apiFetch(page: Page, method: string, url: string, body?: unknown) {
  return page.evaluate(
    async ({ method, url, body }) => {
      const r = await fetch(url, {
        method,
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      return { status: r.status, body: await r.json().catch(() => null) };
    },
    { method, url, body },
  );
}

async function setReviewEnabled(page: Page, projectId: string, enabled: boolean) {
  const project = await apiFetch(page, "GET", `/api/v1/projects/${projectId}`);
  expect(project.status).toBe(200);
  const updated = await apiFetch(page, "PUT", `/api/v1/projects/${projectId}`, { ...project.body, uploadReviewEnabled: enabled });
  expect(updated.status, `set uploadReviewEnabled=${enabled}`).toBeLessThan(300);
}

test.describe("rejected stamp", () => {
  let errors: string[];
  let projectId: string;
  let rejectedImageId: string | null = null;

  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
    const project = await loginAs(page, "admin");
    expect(project).not.toBeNull();
    projectId = project!.id;
    await setReviewEnabled(page, projectId, true);
  });

  test.afterEach(async ({ page }) => {
    // un-reject the image and switch the review flow back off
    if (rejectedImageId) {
      const image = await apiFetch(page, "GET", `/api/v1/images/${rejectedImageId}`);
      const assignment = (image.body?.tags ?? []).find((a: any) => a?.tag?.name === "rejected");
      if (assignment) await apiFetch(page, "DELETE", `/api/v1/image-tag-assignments/${assignment.id}`);
      rejectedImageId = null;
    }
    await setReviewEnabled(page, projectId, false);
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("stamps on apply, stays on the image, clears on navigation", async ({ page }) => {
    await page.goto("/images");
    await page.locator('[id^="grid-tile-"]').first().click();
    await expect(page.getByText("Image Tags")).toBeVisible();
    const stamp = page.getByTestId("rejected-stamp");
    await expect(stamp).toHaveCount(0);
    const first = imageParam(page);
    rejectedImageId = first;

    // reject via the tagging dialog
    await page.keyboard.press("t");
    const search = page.getByPlaceholder("Search tag...");
    await expect(search).toBeVisible();
    await search.fill("rejected");
    await page.locator('li:has-text("rejected")').first().click();
    await page.keyboard.press("Escape");
    await expect(search).toBeHidden();

    // the stamp is on the image and survives closing the dialog
    await expect(stamp).toBeVisible();

    // moving to the next image clears it ...
    await page.keyboard.press("ArrowRight");
    await expect.poll(() => imageParam(page)).not.toBe(first);
    await expect(stamp).toHaveCount(0);

    // ... and coming back does not re-stamp: only the act of rejecting stamps
    await page.keyboard.press("ArrowLeft");
    await expect.poll(() => imageParam(page)).toBe(first);
    await expect(stamp).toHaveCount(0);
  });
});
