// Keeps stale browser tabs working across deploys. See util/updateGuard.ts
// for the decision logic and the reasoning; this file is the wiring only.
import { boot } from "quasar/wrappers";
import { API_BASE } from "src/boot/axios";
import { isVersionChange, RELOAD_COOLDOWN_MS, shouldReloadOnChunkError, VERSION_CHECK_INTERVAL_MS } from "src/util/updateGuard";

const RELOAD_KEY = "shutterbase-chunk-reload-at";

async function fetchServerVersion(): Promise<string | null> {
  try {
    const response = await fetch(`${API_BASE}/health`, { credentials: "include" });
    if (!response.ok) return null;
    return ((await response.json()) as { version?: string }).version ?? null;
  } catch {
    return null;
  }
}

export default boot(({ router }) => {
  // --- reactive: stale chunk failed to load ---------------------------------
  window.addEventListener("vite:preloadError", (event) => {
    const last = Number(sessionStorage.getItem(RELOAD_KEY)) || null;
    if (!shouldReloadOnChunkError(last, Date.now())) return; // let it surface
    event.preventDefault(); // suppress the error — we recover instead
    sessionStorage.setItem(RELOAD_KEY, String(Date.now()));
    window.location.reload();
  });

  // --- proactive: server runs a newer build than this tab -------------------
  let baseline: string | null = null;
  let updatePending = false;

  const check = async () => {
    if (updatePending) return;
    const version = await fetchServerVersion();
    if (baseline === null) {
      baseline = version;
    } else if (isVersionChange(baseline, version)) {
      updatePending = true;
    }
  };

  void check(); // capture the baseline at app start
  setInterval(() => void check(), VERSION_CHECK_INTERVAL_MS);
  let lastFocusCheck = 0;
  document.addEventListener("visibilitychange", () => {
    // returning to the tab is the moment stale tabs resurface — re-check, but
    // no more often than the reload cooldown
    if (document.visibilityState === "visible" && Date.now() - lastFocusCheck > RELOAD_COOLDOWN_MS) {
      lastFocusCheck = Date.now();
      void check();
    }
  });

  router.beforeEach((to) => {
    if (!updatePending) return true;
    // Full document load to the target route: the user sees an ordinary page
    // transition and lands in the new build.
    window.location.href = router.resolve(to).href;
    return false;
  });
});
