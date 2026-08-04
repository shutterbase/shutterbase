import { Page, expect } from "@playwright/test";

export const PERSONAS = ["admin", "user", "projectAdmin", "projectEditor", "projectViewer"] as const;
export type Persona = (typeof PERSONAS)[number];

// Defensive list-shape parser — the REST endpoints wrap results differently in places.
const listOf = (b: any): any[] => b?.items || b?.data || b?.results || (Array.isArray(b) ? b : []);

/** Authenticate as a seeded persona via the DEV-only login route (role === seeded username). */
export async function devLogin(page: Page, role: Persona): Promise<void> {
  await page.goto("/login");
  const status = await page.evaluate(async (role) => {
    const r = await fetch("/api/v1/dev/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ role }),
    });
    return r.status;
  }, role);
  expect(status, `dev login as ${role}`).toBeLessThan(300);
}

/**
 * Make the seed project active for this persona, the same way clicking "Activate"
 * in the projects list does (writes the store's persisted keys). `load()` only
 * overwrites activeProjectId when the server returns a server-side active project,
 * so this sticks for every persona. Returns null for personas with no project (user).
 */
export async function activateSeedProject(page: Page): Promise<{ id: string; name: string } | null> {
  return page.evaluate(async () => {
    const j = async (u: string) => {
      const r = await fetch(u, { credentials: "include" });
      const b = await r.json().catch(() => ({}));
      return b?.items || b?.data || b?.results || (Array.isArray(b) ? b : []);
    };
    const p = (await j("/api/v1/projects?limit=1"))[0];
    if (!p) return null;
    localStorage.setItem("activeProjectId", p.id);
    localStorage.setItem("activeProject", JSON.stringify({ id: p.id, name: p.name }));
    const tags = await j(`/api/v1/image-tags?projectId=${p.id}&limit=100`);
    localStorage.setItem("projectTags", JSON.stringify(tags));
    return { id: p.id, name: p.name };
  });
}

/** Login + (optionally) activate the seed project. Returns the active project (or null). */
export async function loginAs(page: Page, role: Persona, opts: { activate?: boolean } = {}) {
  await devLogin(page, role);
  return opts.activate === false ? null : await activateSeedProject(page);
}

/** Resolve the seed project id without changing active state (for building URLs). */
export async function seedProjectId(page: Page): Promise<string | null> {
  return page.evaluate(async () => {
    const r = await fetch("/api/v1/projects?limit=1", { credentials: "include" });
    const b = await r.json().catch(() => ({}));
    const list = b?.items || b?.data || b?.results || [];
    return list[0]?.id ?? null;
  });
}

/** Resolve the logged-in user's own id (for /users/:id/... routes). */
export async function meId(page: Page): Promise<string | null> {
  return page.evaluate(async () => {
    const r = await fetch("/api/v1/users/me", { credentials: "include" });
    const b = await r.json().catch(() => ({}));
    return b?.id ?? b?.data?.id ?? null;
  });
}

/**
 * Attach a JS-error collector. We deliberately ignore network-resource failures
 * (gravatar `?d=404` avatar fallback, pre-auth /users/me probes) — those are
 * expected and not migration/render bugs. Only uncaught JS exceptions count.
 */
export function collectJsErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(`pageerror: ${e.message || e}`));
  return errors;
}

export { listOf };
