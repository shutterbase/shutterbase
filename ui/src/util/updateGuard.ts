// Pure decision logic for the SPA update guard (boot/updateGuard.ts wires it
// to the router and window events). Two recovery paths after a deploy:
//
//  1. Proactive: the app remembers the server version from its first /health
//     call; when a later check sees a different version, the NEXT router
//     navigation becomes a full document load — the user gets the new build
//     disguised as an ordinary page transition.
//  2. Reactive: if a lazy chunk still fails (vite:preloadError), reload the
//     page — rate-limited so a genuinely broken asset can't reload-loop.

export const RELOAD_COOLDOWN_MS = 30_000;
export const VERSION_CHECK_INTERVAL_MS = 5 * 60 * 1000;

// isVersionChange: true only when both sides are known and differ — an
// unreachable health endpoint (empty/undefined) must never trigger reloads.
export function isVersionChange(baseline: string | null, current: string | null | undefined): boolean {
  return !!baseline && !!current && baseline !== current;
}

// shouldReloadOnChunkError: allow at most one automatic reload per cooldown
// window. lastReloadAt is the persisted (sessionStorage) timestamp of the
// previous automatic reload, as epoch ms.
export function shouldReloadOnChunkError(lastReloadAt: number | null, now: number): boolean {
  return lastReloadAt === null || now - lastReloadAt > RELOAD_COOLDOWN_MS;
}
