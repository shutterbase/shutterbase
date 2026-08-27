import { test, expect, request } from "@playwright/test";

// The public gallery (cmd/testserver :8090, key "e2e") after publishing the
// seed project through the shutterbase API on the SPA origin (:9000).
const API = process.env.PLAYWRIGHT_BASE_URL || "http://localhost:9000";

async function publishSeedProject(): Promise<{ projectId: string; publicId: string; hiddenId: string; publicTagId: string }> {
  const api = await request.newContext({ baseURL: API });
  const login = await api.post("/api/v1/dev/login", { data: { role: "admin" } });
  expect(login.ok()).toBeTruthy();
  const galleries = (await (await api.get("/api/v1/galleries?limit=100")).json()).items ?? [];
  let gallery = galleries.find((g: any) => g.key === "e2e");
  if (!gallery) {
    gallery = await (await api.post("/api/v1/galleries", { data: { key: "e2e", name: "E2E Public Gallery", tagline: "Seed photos", locale: "en" } })).json();
  } else {
    await api.put(`/api/v1/galleries/${gallery.id}`, { data: { active: true, locale: "en", name: "E2E Public Gallery" } });
  }
  const project = (await (await api.get("/api/v1/projects?limit=1")).json()).items[0];
  const pub = await api.put(`/api/v1/projects/${project.id}`, { data: { galleryId: gallery.id, gallerySlug: "seed-event" } });
  expect(pub.ok(), await pub.text()).toBeTruthy();
  const tags = (await (await api.get(`/api/v1/image-tags?projectId=${project.id}&limit=500`)).json()).items;
  const publicTag = tags.find((t: any) => t.name === "public");
  expect(publicTag).toBeTruthy();
  const images = (await (await api.get(`/api/v1/images?projectId=${project.id}&limit=10&sort=capturedAtCorrected&order=asc`)).json()).items;
  const publicImage = images[0];
  await api.post("/api/v1/image-tag-assignments", { data: { imageId: publicImage.id, imageTagId: publicTag.id, type: "manual" } });
  await api.dispose();
  return { projectId: project.id, publicId: publicImage.id, hiddenId: images[1].id, publicTagId: publicTag.id };
}

test.describe("public gallery", () => {
  test("landing → project → grid → detail shows only public images", async ({ page }) => {
    const fx = await publishSeedProject();
    // the gallery caches for 30s in the testserver; wait for the publication to show
    await expect.poll(async () => (await page.request.get("/")).status(), { timeout: 45_000 }).toBe(200);
    await expect.poll(async () => await (await page.request.get("/")).text(), { timeout: 45_000 }).toContain("/seed-event");

    await page.goto("/");
    await expect(page.getByRole("heading", { name: "E2E Public Gallery" })).toBeVisible();
    await page
      .getByRole("link", { name: /Formula Student Test/ })
      .first()
      .click();
    await expect(page).toHaveURL(/\/seed-event$/);
    await page.getByRole("link", { name: /All photos/ }).click();
    await expect(page).toHaveURL(/\/seed-event\/photos/);
    const tiles = page.locator("#grid > a[data-lb-src]");
    await expect(tiles).toHaveCount(1);
    await expect(page.locator(`a[href*="/seed-event/p/${fx.hiddenId}"]`)).toHaveCount(0);

    await page.goto(`/seed-event/p/${fx.publicId}`);
    await expect(page.locator('meta[property="og:title"]')).toHaveCount(1);
    await expect(page.getByRole("link", { name: /Download original/ })).toBeVisible();
    await expect(page.locator("body")).not.toContainText("aiDescription");

    const hidden = await page.request.get(`/seed-event/p/${fx.hiddenId}`);
    expect(hidden.status()).toBe(404);
    const hiddenDownload = await page.request.get(`/d/${fx.hiddenId}`);
    expect(hiddenDownload.status()).toBe(404);
  });

  test("unpublishing removes the image from the download route immediately", async ({ page }) => {
    const fx = await publishSeedProject();
    const api = await request.newContext({ baseURL: API });
    await api.post("/api/v1/dev/login", { data: { role: "admin" } });
    const assignments = (await (await api.get(`/api/v1/image-tag-assignments?imageId=${fx.publicId}&limit=50`)).json()).items ?? [];
    const pubAssignment = assignments.find((a: any) => a.tag?.id === fx.publicTagId || a.imageTagId === fx.publicTagId);
    expect(pubAssignment, "public assignment present").toBeTruthy();
    await api.delete(`/api/v1/image-tag-assignments/${pubAssignment.id}`);
    const res = await page.request.get(`/d/${fx.publicId}`);
    expect(res.status()).toBe(404);
    // restore for other specs
    await api.post("/api/v1/image-tag-assignments", { data: { imageId: fx.publicId, imageTagId: fx.publicTagId, type: "manual" } });
    await api.dispose();
  });
});
