import { test, expect } from "@playwright/test";
import { loginAs, meId, collectJsErrors } from "./helpers";

// The API-key page is the only way to mint a key without curl. The whole point of
// the feature is the one-time secret, so that is what this drives: mint it, prove
// it is displayed, prove a reload never shows it again, then revoke.
//
// Runs as the plain "user" persona on purpose — minting is self-service ("admin or
// self" on the server), so a non-admin must be able to do all of this.
test.describe.serial("api keys", () => {
  const KEY_NAME = "e2e downloader key";

  test("mint a key, see the secret once, then revoke it (plain user)", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "user", { activate: false });
    const userId = await meId(page);
    expect(userId, "logged-in user id").toBeTruthy();

    await page.goto(`/users/${userId}/api-keys`);
    await expect(page.getByRole("button", { name: "Create API key" })).toBeVisible();

    // --- mint ---
    await page.locator("#createApiKey").click();
    await page.locator("#apiKeyName").fill(KEY_NAME);
    await page.locator("#submitApiKey").click();

    const panel = page.getByTestId("minted-token");
    await expect(panel, "the secret must be shown after minting").toBeVisible();
    const token = ((await panel.locator("code").textContent()) || "").trim();
    // "<keyId>.<secret>" — the shape the Authorization header expects.
    expect(token, "minted token shape").toMatch(/^\S+\.\S+$/);

    // --- shown exactly once ---
    await page.reload();
    await expect(page.getByTestId("minted-token"), "the secret must not survive a reload").toHaveCount(0);
    expect(await page.content(), "the secret must appear nowhere after a reload").not.toContain(token);

    // the key itself persisted, and has never been used yet
    const row = page.getByRole("listitem").filter({ hasText: KEY_NAME });
    await expect(row, "the minted key must persist").toBeVisible();
    await expect(row).toContainText("unused");

    // --- revoke ---
    await row.getByRole("button", { name: `Revoke ${KEY_NAME}` }).click();
    await expect(row).toContainText("revoked");
    // revoking keeps the row (the server flips a flag, it does not delete), and
    // the state must survive a reload — not just optimistic UI.
    await page.reload();
    const revoked = page.getByRole("listitem").filter({ hasText: KEY_NAME });
    await expect(revoked).toContainText("revoked");
    await expect(revoked.getByRole("button", { name: /Revoke/ }), "a revoked key cannot be revoked again").toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });

  test("another user's keys are not shown or manageable", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "user", { activate: false });

    // A non-admin hitting someone else's page: the list endpoint would scope to
    // the caller, so the page must refuse rather than label MY keys as theirs.
    await page.goto(`/users/00000000-0000-0000-0000-000000000000/api-keys`);
    await expect(page.getByText("You can only see your own API keys.")).toBeVisible();
    await expect(page.getByRole("button", { name: "Create API key" })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });
});
