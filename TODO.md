# TODOs for FSG

## Bad

- close tagging dialog when clicking outside of it
- multiselect in grid view
- filter for upload with query parameter, etc

## Not so bad

- delete cameras
- add pagination or infiniscroll to project tags
- tag ui broken when tag description too long
- applied tag shows up locally as "manual" even though it might be "custom"
- verify S3 completeness after Hetzner transfer finishes: compare bucket originals (`XX/<storageId>.jpg`) against the 29108 `images.storage_id` rows in prod (mc ls -r diff or `import --verify` re-run)
- ui: `vue-tsc --noEmit` crashes ("isSatisfiesExpression is not defined") — vue-tsc/@vue/language-core incompatible with the pinned typescript; templates currently have no typecheck gate. Bump vue-tsc + @vue/language-core (or typescript) until it runs clean
- ci: `Server Build` job always fails on fork PRs — the docker push step runs with the fork's read-only GITHUB_TOKEN ("denied: installation not allowed to Write organization package"). Guard the push with `if: github.event_name != 'pull_request'` (build-only for PRs) so fork contributions can go green
- e2e: `project-tags.spec.ts` is flaky on main — which test fails moves between runs (`:51` display-name fallback, `:94` template tag) and the failing assertion moves within a test (persist-after-reload vs. the later clear check), so it looks like a race between the save dialog closing and the tag table refresh rather than a broken feature. Verified pre-existing: `:51` fails identically with `ui/src` stashed. Every other spec passes (38/38). Reproduce with `bunx playwright test project-tags.spec.ts`
- schema hardening: all other `name` fields (upload, imageTag, personName, camera, project, apiKey) are still unlimited varchar — same class of bug as the download-config name (#90). Before capping them, check prod for existing rows longer than the target limit; an ent auto-migrate column shrink fails on oversized rows and stalls the rollout
- upload pause: pausing does not abort the in-flight WASM S3 transfer — the current image's remaining presigned PUTs finish in the background and the resumed run re-uploads under a fresh storageId, orphaning the first object set (same orphan class as a failed create). Fix needs a cancellation signal plumbed into `process_file`'s upload loop (abort the active XHR, check before each presign/PUT)
- upload pause: an ERROR tile from a lost create *response* (row inserted, response dropped) is removable/retryable and the retry 409s on the unique computedFileName; a page reload shows the persisted image and resolves it. Rare network-drop window; if it ever bites, reconcile by storageId before allowing the retry
- ws resilience: `websocket.ts` reconnects only from `onerror` — a clean close frame (proxy drain, server read-deadline eviction at `event/websocket.go:170`) fires only `onclose` and the shared socket stays dead until a consumer remounts; no backoff on the 1s retry; `ws:open` has zero subscribers so nothing refetches after a reconnect gap (frozen AI counters); `QrTimeCode` keeps extrapolating from `performance.now()` with no staleness indicator when the tick socket dies — that's the camera-calibration surface. Also `disconnect()` on QrTimeCode/TimeOffsetCreate unmount closes the singleton for every other consumer
- upload resilience during a blip: the WASM S3 PUT and presign are single-shot (no retry, `image-wasm/src/upload.rs`), a failed `POST /images` lands the tile in ERROR, and ERROR is not in `RESETTABLE_STATES` — pause/resume cannot re-queue it, only remove + re-pick the file. No `beforeunload` guard while a queue is in flight, so the updateGuard's post-deploy hard navigation (or a tab close) silently discards pending uploads. Also `boot/axios.ts` has no timeout — a hung connection during pod shutdown waits on the browser TCP timeout
- build: `ui/public/image-wasm/` is dead weight (~9 MB in every SPA build → embedded in the Go binary). Vite bundles the wasm from the `image-wasm` package dependency (`assets/image_wasm_bg-*.wasm`), so the public copy is never fetched — and `image-wasm/hack/build.sh`'s `cp -r pkg ui/public/image-wasm` nests a second stale copy into the existing dir instead of replacing it. Drop the cp + delete `ui/public/image-wasm/`, verify upload + time-offset pages still init the wasm in a built binary (not just quasar dev)
