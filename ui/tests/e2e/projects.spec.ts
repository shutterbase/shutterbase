import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

// Activating a project used to change nothing visible (user feedback: "did it
// work?") — now it must confirm itself unmissably: modal + confetti.
test("activating a project confirms with a modal", async ({ page }) => {
  const project = await loginAs(page, "projectAdmin", { activate: false });
  expect(project).toBeNull(); // activation happens through the UI below

  await page.goto("/projects");
  const row = page.getByRole("row").filter({ has: page.getByRole("button", { name: "Activate" }) }).first();
  const projectName = (await row.getByRole("cell").first().innerText()).trim();
  await row.getByRole("button", { name: "Activate" }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("Project activated")).toBeVisible();
  await expect(dialog.getByText(projectName)).toBeVisible();

  await dialog.getByRole("button", { name: "Let's go" }).click();
  await expect(dialog).toHaveCount(0);

  // the activation actually stuck (same persisted keys the store writes)
  const activeId = await page.evaluate(() => localStorage.getItem("activeProjectId"));
  expect(activeId).toBeTruthy();
});
