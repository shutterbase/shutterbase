import { test, expect } from "@playwright/test";
import { devLogin } from "./helpers";

test.describe("authentication", () => {
  test("unauthenticated access to a protected route redirects to login", async ({ page }) => {
    await page.context().clearCookies();
    await page.goto("/images");
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole("button", { name: /^Sign in$/ })).toBeVisible();
  });

  test("dev login lands in the app shell with primary nav", async ({ page }) => {
    await devLogin(page, "admin");
    await page.goto("/");
    await expect(page.locator("header").first()).toBeVisible();
    await expect(page.locator("header nav").getByText("Projects", { exact: true })).toBeVisible();
  });

  test("logout returns to the login screen", async ({ page }) => {
    await devLogin(page, "admin");
    await page.goto("/logout");
    await expect(page).toHaveURL(/\/login$/, { timeout: 10_000 });
  });
});
