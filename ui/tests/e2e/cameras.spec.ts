import { test, expect } from "@playwright/test";
import { loginAs, meId, collectJsErrors } from "./helpers";

// Cameras are soft-deleted on the server (photos keep their camera ref), so the
// page must offer Delete on every camera, gate it behind a confirm dialog, and
// the freed name must be usable again immediately.
test.describe.serial("cameras", () => {
  const CAM_NAME = "e2e soft-delete cam";

  test("create a camera, delete it, reuse the name (plain user)", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "user", { activate: false });
    const userId = await meId(page);
    expect(userId, "logged-in user id").toBeTruthy();

    // --- create ---
    await page.goto(`/users/${userId}/cameras`);
    await page.getByRole("button", { name: "Add Camera" }).click();
    await page.getByLabel("Name").fill(CAM_NAME);
    await page.getByRole("button", { name: "Create" }).click();
    await expect(page.getByRole("heading", { name: CAM_NAME }), "created camera must appear").toBeVisible();

    // --- delete: cancel keeps it, confirm removes it ---
    await page.getByRole("button", { name: `Delete ${CAM_NAME}` }).click();
    await expect(page.getByText("Delete camera")).toBeVisible();
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByRole("heading", { name: CAM_NAME }), "cancel must keep the camera").toBeVisible();

    await page.getByRole("button", { name: `Delete ${CAM_NAME}` }).click();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("heading", { name: CAM_NAME })).toHaveCount(0);

    // server-side, not just optimistic UI
    await page.reload();
    await expect(page.getByRole("heading", { name: CAM_NAME })).toHaveCount(0);

    // --- the name is freed by the soft delete ---
    await page.getByRole("button", { name: "Add Camera" }).click();
    await page.getByLabel("Name").fill(CAM_NAME);
    await page.getByRole("button", { name: "Create" }).click();
    await expect(page.getByRole("heading", { name: CAM_NAME }), "freed name must be reusable").toBeVisible();

    // cleanup so reruns start from a clean list
    await page.getByRole("button", { name: `Delete ${CAM_NAME}` }).click();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("heading", { name: CAM_NAME })).toHaveCount(0);

    expect(errors, errors.join("\n")).toHaveLength(0);
  });
});
