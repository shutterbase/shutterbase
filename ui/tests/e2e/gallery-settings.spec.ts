import { test, expect } from "@playwright/test";
import { loginAs, seedProjectId, collectJsErrors } from "./helpers";

// Publishing a project onto a public gallery: a platform admin creates the
// gallery, a project admin publishes the seed project and marks an image
// public through the normal tagging flow; an editor is offered neither.
test.describe("public gallery settings", () => {
  const key = `e2e-${Date.now().toString(36)}`;

  test("admin creates a gallery and edits its branding", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "admin", { activate: false });
    await page.goto("/galleries");
    await expect(page.getByRole("heading", { name: /galler/i }).first()).toBeVisible();
    await page.getByRole("button", { name: /add/i }).click();
    await expect(page).toHaveURL(/\/galleries\/create/);
    await page.getByLabel("Key").fill(key);
    await page.getByLabel("Name").fill("E2E Media");
    await page.getByRole("button", { name: "Create" }).click();
    await expect(page).toHaveURL(/\/galleries\/[a-z0-9]+$/);
    await page.getByLabel("Tagline").fill("Every lap.");
    await page.getByLabel("Imprint URL").fill("https://example.org/imprint");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Saved")).toBeVisible();
    expect(errors, `js errors: ${errors.join("; ")}`).toHaveLength(0);
  });

  test("project admin publishes the seed project; editor cannot", async ({ page }) => {
    await loginAs(page, "admin", { activate: false });
    // the gallery from the first test may not exist in a filtered run: ensure one
    const galleryId = await page.evaluate(async (key) => {
      const list = await (await fetch("/api/v1/galleries?limit=100", { credentials: "include" })).json();
      const existing = (list.items ?? []).find((g: any) => g.key === key);
      if (existing) return existing.id;
      const r = await fetch("/api/v1/galleries", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, name: "E2E Media" }),
      });
      return (await r.json()).id;
    }, key);
    expect(galleryId).toBeTruthy();

    await loginAs(page, "projectAdmin");
    const pid = await seedProjectId(page);
    await page.goto(`/projects/${pid}/general`);
    const section = page.getByTestId("public-gallery");
    await expect(section).toBeVisible();
    await section.getByTestId("gallery-select").selectOption(galleryId);
    await section.getByTestId("gallery-slug").fill("seed-event");
    await section.getByTestId("gallery-save").click();
    await expect(section.getByText(/Published since/)).toBeVisible();

    // the reserved public tag exists now and only a project admin may assign it
    const publicTag = await page.evaluate(async (pid) => {
      const list = await (await fetch(`/api/v1/image-tags?projectId=${pid}&limit=500`, { credentials: "include" })).json();
      return (list.items ?? []).find((t: any) => t.name === "public");
    }, pid);
    expect(publicTag, "public tag materialized").toBeTruthy();

    await loginAs(page, "projectEditor");
    await page.goto(`/projects/${pid}/general`);
    await expect(page.getByTestId("gallery-select")).toBeDisabled();
    const status = await page.evaluate(
      async ({ pid, tagId }) => {
        const images = await (await fetch(`/api/v1/images?projectId=${pid}&limit=1`, { credentials: "include" })).json();
        const r = await fetch("/api/v1/image-tag-assignments", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ imageId: images.items[0].id, imageTagId: tagId, type: "manual" }),
        });
        return r.status;
      },
      { pid, tagId: publicTag.id },
    );
    expect(status, "editor cannot publish an image").toBe(403);
  });
});
