import { chromium, FullConfig } from "@playwright/test";

// Reseed the backend to a known fixture before the suite runs, so every spec
// starts from the deterministic seed state (1 project, 3 tags, 1 upload, 3 images,
// 2 cameras, 5 personas). See api/internal/seed/seed.go.
export default async function globalSetup(config: FullConfig) {
  const base = config.projects[0]?.use?.baseURL || "http://localhost:9000";
  const browser = await chromium.launch();
  const page = await browser.newPage();

  const resp = await page.goto(`${base}/login`).catch(() => null);
  if (!resp) {
    await browser.close();
    throw new Error(
      `Dev stack not reachable at ${base}.\n` +
        `Start it first: postgres (sb-pg) + 'go run ./cmd/server serve' (:8080, DEV=true) + 'bun run dev' (:9000).`,
    );
  }

  // Boot admin carries ForcePasswordChange on a fresh DB; clearing it is required
  // before reseed. On an already-seeded DB the seeded admin has no force flag and
  // the change-password call simply fails (wrong currentPassword) — harmless, ignored.
  await page.evaluate(async () => {
    await fetch("/api/v1/dev/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ role: "admin" }),
    });
    await fetch("/api/v1/auth/change-password", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ currentPassword: "changeme123", newPassword: "Devpassw0rd!", newPasswordConfirm: "Devpassw0rd!" }),
    }).catch(() => {});
  });

  const reseed = await page.evaluate(async () => (await fetch("/api/v1/dev/reseed", { method: "POST", credentials: "include" })).status);
  await browser.close();
  if (reseed >= 400) throw new Error(`reseed failed (HTTP ${reseed}). Is the server in DEV mode?`);
  console.log(`[global-setup] backend reseeded (HTTP ${reseed})`);
}
