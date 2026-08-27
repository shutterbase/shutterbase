import { test, expect } from "@playwright/test";
import { loginAs, collectJsErrors } from "./helpers";

// The "ask" filter: Enter in the search field (or the Ask button) narrows the
// normal grid via ?ask=, shows a chip with the query, and the chip's × restores
// the unfiltered view. The e2e stack's stub AI server ranks nothing, so the
// grid shows its empty state — the point is the filter plumbing, not results.
test.describe("ask filter", () => {
  test("Enter applies ?ask=, the chip shows the query, × clears it", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "admin");
    await page.goto("/images");

    const search = page.locator("#search");
    await expect(search).toBeVisible();
    await expect(page.getByRole("button", { name: "Ask" })).toBeDisabled();

    await search.fill("team celebrating in the rain");
    await expect(page.getByRole("button", { name: "Ask" })).toBeEnabled();
    await search.press("Enter");

    await expect(page).toHaveURL(/[?&]ask=team(\+|%20)celebrating/);
    const chip = page.getByTestId("ask-chip");
    await expect(chip).toContainText("team celebrating in the rain");

    await chip.locator("button").click();
    await expect(chip).toHaveCount(0);
    await expect(page).not.toHaveURL(/ask=/);
    expect(errors).toEqual([]);
  });
});
