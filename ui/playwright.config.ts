import { defineConfig, devices } from "@playwright/test";

// E2E suite for the shutterbase SPA. Requires the dev stack running:
//   postgres (sb-pg) + `go run ./cmd/server serve` (:8080, DEV=true) + `bun run dev` (:9000)
// global-setup reseeds the DB to a known fixture before the suite runs.
//
// Tests share one backend + DB and some mutate it, so the suite runs serially
// (workers: 1, fullyParallel: false) for deterministic state.
export default defineConfig({
  testDir: "./tests/e2e",
  globalSetup: "./tests/e2e/global-setup.ts",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: 0,
  timeout: 30_000,
  expect: { timeout: 7_000 },
  reporter: [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://localhost:9000",
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    // Pinned, not inherited from the host. The `datetime-local` inputs and the
    // Time popover work in local wall clock, so a spec that types "2026-08-01
    // T00:00" and asserts the resulting ISO only holds under a known zone —
    // without this the suite passes on a CEST developer machine and fails
    // everywhere else.
    timezoneId: process.env.PLAYWRIGHT_TIMEZONE || "Europe/Berlin",
    locale: "en-GB",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
