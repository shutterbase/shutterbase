import { test, expect, Page } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// Permalinks: a shared /images?image=<id> URL must open the detail view for the
// recipient (jump-to-context / solo resolution in imageQueryLogic.jumpToImage),
// degrade with a toast for dead links, and survive the login redirect.

const imageParam = (page: Page) => new URL(page.url()).searchParams.get("image");

async function anyImageId(page: Page, projectId: string): Promise<string> {
  return page.evaluate(async (pid) => {
    const r = await fetch(`/api/v1/images?projectId=${pid}&limit=1`, { credentials: "include" });
    const b = await r.json();
    return b.items[0].id as string;
  }, projectId);
}

test.describe("image permalinks", () => {
  let errors: string[];
  test.beforeEach(async ({ page }) => {
    errors = collectJsErrors(page);
  });
  test.afterEach(() => {
    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("deep link opens the detail view directly", async ({ page }) => {
    const project = await loginAs(page, "admin");
    const id = await anyImageId(page, project!.id);
    await page.goto(`/images?image=${id}`);
    await expect(page.getByText("Image Tags")).toBeVisible();
    expect(imageParam(page)).toBe(id);
  });

  test("copy link puts the canonical permalink on the clipboard", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    const project = await loginAs(page, "admin");
    const id = await anyImageId(page, project!.id);
    await page.goto(`/images?image=${id}`);
    await page.getByText("copy link", { exact: true }).click();
    await expect(page.getByText("Link copied")).toBeVisible();
    const copied = await page.evaluate(() => navigator.clipboard.readText());
    expect(copied).toContain(`/images?image=${id}`);
  });

  test("dead link falls back to the grid with a toast", async ({ page }) => {
    await loginAs(page, "admin");
    await page.goto("/images?image=no-such-image");
    await expect(page.getByText("This image no longer exists")).toBeVisible();
    await expect.poll(() => imageParam(page)).toBeNull(); // ?image stripped from the URL
  });

  test("unauthenticated deep link is carried through the login redirect", async ({ page }) => {
    await page.goto("/images?image=some-id");
    await expect(page).toHaveURL(/\/login\?redirect=/);
    expect(new URL(page.url()).searchParams.get("redirect")).toBe("/images?image=some-id");
  });
});
