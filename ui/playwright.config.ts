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
    baseURL: "http://localhost:9000",
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
