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

  test("Ctrl+wheel sizes the grid into the detail page; wheel zooms the hero; fullscreen overlays", async ({ page }) => {
    const fx = await publishSeedProject();
    await expect.poll(async () => await (await page.request.get("/")).text(), { timeout: 45_000 }).toContain("/seed-event");
    await page.goto("/seed-event/photos");
    const tile = page.locator("#grid > a[data-lb-src]").first();
    await expect(tile).toBeVisible();
    const tileVar = () => page.evaluate(() => parseFloat(getComputedStyle(document.querySelector("#grid")!).getPropertyValue("--tile")));
    const before = await tileVar();
    await tile.hover();
    await page.keyboard.down("Control");
    await page.mouse.wheel(0, -300);
    expect(await tileVar()).toBeGreaterThan(before);
    // the +/− buttons drive the same axis
    const mid = await tileVar();
    await page.getByRole("button", { name: "Zoom out" }).click();
    expect(await tileVar()).toBeLessThan(mid);
    // keep zooming in: once the tile fills the row the axis continues on the detail page
    await tile.hover();
    for (let i = 0; i < 25 && !page.url().includes("/p/"); i++) await page.mouse.wheel(0, -300);
    await page.keyboard.up("Control");
    await expect(page).toHaveURL(new RegExp(`/seed-event/p/${fx.publicId}`));

    // plain wheel over the hero zooms in place, click-drag pans
    const hero = page.locator("[data-hero]");
    await hero.hover();
    await page.mouse.wheel(0, -300);
    await expect(hero).toHaveClass(/zoomed/);
    await expect(hero.locator("img")).toHaveAttribute("style", /scale\((1\.[0-9]+|[2-8])/);
    await page.mouse.wheel(0, 300);
    await page.mouse.wheel(0, 300);
    await expect(hero).not.toHaveClass(/zoomed/);

    // fullscreen button opens the overlay, Escape closes it
    await page.getByRole("button", { name: "Fullscreen" }).click();
    await expect(page.locator(".lightbox")).toHaveCount(1);
    await expect(page).toHaveURL(/#fs$/);
    await page.keyboard.press("Escape");
    await expect(page.locator(".lightbox")).toHaveCount(0);

    // a hard scroll-out of the fitted hero returns to the grid
    await hero.hover();
    for (let i = 0; i < 6; i++) await page.mouse.wheel(0, 250);
    await expect(page).toHaveURL(/\/seed-event\/photos/);
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
