import { test, expect } from "@playwright/test";
import { loginAs, seedProjectId, meId, collectJsErrors } from "./helpers";

// Drives every route as admin (the access superset) and asserts each renders
// without an uncaught JS error or a Vite error overlay, and paints real content.
// This is the migration's render gate — a broken import/template surfaces here.
test.describe.serial("smoke", () => {
  test("every route renders without JS errors (admin)", async ({ page }) => {
    const errors = collectJsErrors(page);
    await loginAs(page, "admin");
    const pid = await seedProjectId(page);
    const uid = await meId(page);
    const { uploadId, cameraId } = await page.evaluate(async (pid: string | null) => {
      const j = async (u: string) => {
        const r = await fetch(u, { credentials: "include" });
        const b = await r.json().catch(() => ({}));
        return b?.items || b?.data || [];
      };
      const up = pid ? await j(`/api/v1/uploads?projectId=${pid}&limit=1`) : [];
      const cam = await j(`/api/v1/cameras?limit=1`);
      return { uploadId: up[0]?.id ?? null, cameraId: cam[0]?.id ?? null };
    }, pid);

    const routes: [string, string][] = [
      ["index", "/"],
      ["projects", "/projects"],
      ["project-create", "/projects/create"],
      ["images", "/images"],
      ["uploads", "/uploads"],
      ["uploads-create", "/uploads/create"],
      ["users", "/users"],
      ["change-password", "/change-password"],
      ["sandbox", "/sandbox"],
      ["notfound", "/zzz-does-not-exist"],
    ];
    if (pid)
      routes.push(
        ["project-general", `/projects/${pid}/general`],
        ["project-tags", `/projects/${pid}/tags`],
        ["project-statistics", `/projects/${pid}/statistics`],
        ["project-members", `/projects/${pid}/members`],
        ["project-danger-zone", `/projects/${pid}/danger-zone`],
      );
    if (uploadId) routes.push(["upload-edit", `/uploads/${uploadId}/edit`]);
    if (uid)
      routes.push(
        ["user-general", `/users/${uid}/general`],
        ["user-cameras", `/users/${uid}/cameras`],
        ["user-camera-create", `/users/${uid}/cameras/create`],
      );
    if (cameraId) routes.push(["camera-timeoffset", `/cameras/${cameraId}/time-offset`]);

    const failures: string[] = [];
    for (const [name, path] of routes) {
      const before = errors.length;
      await page.goto(path, { waitUntil: "networkidle" }).catch(() => {});
      await page.waitForTimeout(500);
      const actual = new URL(page.url()).pathname;
      const overlay = await page.locator("vite-error-overlay").count().catch(() => 0);
      const text = (await page.locator("body").innerText().catch(() => "")).trim();
      const jsErrs = errors.slice(before);
      if (jsErrs.length) failures.push(`${name} (${path}): JS ${jsErrs.join("; ")}`);
      if (overlay > 0) failures.push(`${name} (${path}): vite error overlay present`);
      // an authed admin should not be bounced off any of these routes (catches the
      // vacuous "redirected to /login but still has body text" pass).
      if (actual !== path) failures.push(`${name} (${path}): redirected to ${actual}`);
      if (text.length < 10) failures.push(`${name} (${path}): rendered empty (${text.length} chars)`);
      console.log(`  ${name.padEnd(22)} -> ${path}  ${jsErrs.length || overlay || actual !== path ? "FAIL" : "ok"}`);
    }
    expect(failures, `\n${failures.join("\n")}\n`).toHaveLength(0);
  });
});
