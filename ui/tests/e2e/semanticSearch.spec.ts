import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// Semantic search entry point: typing a description and pressing Enter (or
// the "Ask" button) opens the ranked-results dialog titled with the query.
// The e2e stack has no AI server, so the dialog must degrade to an empty
// state (no uncaught error, no crash) rather than a broken overlay.
test.describe("semantic search", () => {
  test("Enter in the search field opens the ranked dialog with the query", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "admin");
    await page.goto("/images");

    const search = page.locator("#search");
    await expect(search).toBeVisible();
    await expect(page.getByRole("button", { name: "Ask" })).toBeDisabled();

    await search.fill("team celebrating in the rain");
    await expect(page.getByRole("button", { name: "Ask" })).toBeEnabled();
    await search.press("Enter");

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText("team celebrating in the rain");
    await expect(dialog.getByText(/Nothing matches|Loading/)).toBeVisible();

    await dialog.getByRole("button", { name: "previous" }).waitFor();
    expect(errors).toEqual([]);
  });
});
