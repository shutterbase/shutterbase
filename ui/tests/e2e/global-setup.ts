import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { chromium, FullConfig } from "@playwright/test";

// The shape /dev/reseed returns, narrowed to the fields the suite asserts on.
// See api/internal/seed/seed.go — Manifest.
interface SeedManifest {
  images: string[];
  timeRangeImages: string[];
  timeRangeStart: string;
  timeRangeEnd: string;
}

// The counts the suite depends on. A spec that hardcodes 8 or 11 can only fail
// with "expected 8, got N", which sends you hunting through the seeder. Naming
// the invariant turns fixture drift into a one-line diagnosis.
const EXPECTED = {
  baseImages: 3,
  clusterImages: 8,
  totalImages: 11,
} as const;

// Published so specs can derive their tile counts instead of hardcoding them.
//
// Through a FILE, not a module-level variable: globalSetup is loaded by the
// Playwright config process while the specs run in worker processes, so they are
// separate module instances and can never share memory. (The first version of
// this exported the variable directly and every consumer failed with
// "seedManifest() called before globalSetup ran".)
const MANIFEST_PATH = join(".playwright", "seed-manifest.json");

/** The manifest globalSetup captured, or a throw naming the likely cause. */
export function seedManifest(): SeedManifest {
  if (!existsSync(MANIFEST_PATH)) {
    throw new Error(
      `${MANIFEST_PATH} is missing — globalSetup did not run, or failed before writing it.\n` +
        `Run the suite through playwright (bun run test:e2e) rather than invoking one spec file directly.`,
    );
  }
  return JSON.parse(readFileSync(MANIFEST_PATH, "utf8")) as SeedManifest;
}

export default async function globalSetup(config: FullConfig) {
  const base = config.projects[0]?.use?.baseURL || "http://localhost:9000";
  const browser = await chromium.launch();
  const page = await browser.newPage();

  const resp = await page.goto(`${base}/login`).catch(() => null);
  if (!resp) {
    await browser.close();
    throw new Error(`Dev stack not reachable at ${base}.\n` + `Start it first: postgres (sb-pg) + 'go run ./cmd/server serve' (:8080, DEV=true) + 'bun run dev' (:9000).`);
  }
  if (resp.status() >= 400) {
    // A dev server that answers 404 or 500 still resolves goto(), so without
    // this the message above stays silent and the real cause is the "wrong
    // port" one you would otherwise suspect.
    await browser.close();
    throw new Error(`${base}/login answered HTTP ${resp.status()} — the port is reachable but is not the SPA.`);
  }

  // Boot admin carries ForcePasswordChange on a fresh DB; clearing it is required
  // before reseed. On an already-seeded DB the seeded admin has no force flag and
  // the change-password call simply fails (wrong currentPassword) — harmless, ignored.
  const raw = await page.evaluate(async () => {
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
    const r = await fetch("/api/v1/dev/reseed", { method: "POST", credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  });
  await browser.close();
  if (raw.status >= 400) throw new Error(`reseed failed (HTTP ${raw.status}). Is the server in DEV mode?`);

  const m = raw.body as SeedManifest | null;
  if (!m || !Array.isArray(m.images) || !Array.isArray(m.timeRangeImages)) {
    throw new Error(`reseed returned no usable manifest (got ${JSON.stringify(raw.body)?.slice(0, 200)}).`);
  }

  // The cluster is this branch's fixture and its count is the number
  // time-range.spec.ts asserts against, so check it here where the message can
  // name the field — once — instead of surfacing there as "expected 8, got 7".
  const count = (o: Record<string, unknown> | string[]) => Object.keys(o ?? {}).length;
  // images holds EVERY seeded photo — the three base ones AND the cluster — so
  // the base count is the remainder, not the length. (Getting this wrong is how
  // the first version of this check reported "baseImages: expected 3, got 11".)
  const total = count(m.images);
  const cluster = count(m.timeRangeImages);
  const actual = {
    baseImages: total - cluster,
    clusterImages: cluster,
    totalImages: total,
  };
  const drift = Object.entries(EXPECTED)
    .filter(([k, want]) => actual[k as keyof typeof actual] !== want)
    .map(([k, want]) => `  ${k}: expected ${want}, got ${actual[k as keyof typeof actual]}`);
  if (drift.length > 0) {
    throw new Error(
      `Seed fixture shape changed:\n${drift.join("\n")}\n\n` +
        `Update EXPECTED in ui/tests/e2e/global-setup.ts and the tile counts in\n` +
        `time-range.spec.ts, or fix api/internal/seed/seed.go if the change is unintended.`,
    );
  }

  mkdirSync(dirname(MANIFEST_PATH), { recursive: true });
  writeFileSync(MANIFEST_PATH, JSON.stringify(m));
  console.log(`[global-setup] backend reseeded (HTTP ${raw.status}) — ${actual.totalImages} images (${actual.clusterImages} cluster)`);
}
