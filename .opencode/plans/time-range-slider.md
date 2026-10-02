# Time-range slider in the Time popover

> Status: **shipped**. This document records the design as BUILT. Where the
> implementation deliberately diverged from the original sketch, the entry says
> so and says why — do not read the "planned" framing as current behaviour.

## Behavior

- Two-thumb slider domain = `[earliest, latest] capturedAtCorrected` of everything matching the current filters **minus the time range itself** (domain stays stable while dragging).
- Thumbs set `from`/`to`; existing datetime-local inputs remain directly below as the **manual override** — typing moves the thumbs, dragging updates them. Single source of truth: the route-driven filter state.
- Drag updates thumbs locally and previews into the inputs; commits happen on **release** (`change`, not `input`) → no request storm. The keyboard path commits per keypress (no drag to interrupt).
- An untouched thumb commits as `null`, so an open-ended range ("everything after 10:00") stays open instead of being silently closed at the domain max.
- The `to` bound is widened to the last **millisecond** of the entered minute in both producers (inputs and slider). The backend compares `to` with an inclusive `LTE` and both controls carry minute precision, so an unwidened bound dropped every photo in the final minute.
- Slider hidden when the domain spans less than a minute; greyed (`disabled`) while the chip-level suspension (`timeRangeSuspended`) is active.
- Domain and ticks are fetched on **popover open** (`timeBoundsNeeded`), not on page mount — the memo key is there precisely to make the second open free.

## Backend

1. `api/internal/repository/image.go`:
   - `ImageTimeBounds{Min,Max *time.Time}` + `GetImageTimeBounds` — two ordered picks (not `MIN`/`MAX` aggregates: SQLite hands aggregates back as untyped strings, and both queries are index-covered anyway).
   - `GetImageTimeTicks(ctx, parameters, maxTicks)` — sampled `capturedAtCorrected` for the density strip.
   - Both **strip `From`/`ToCapturedAtCorrected` on a COPY** of the parameters. Mutating the caller's struct dropped the range from any later reuse of the same `*GetImageParameters`.
   - Tick sampling is bounded by `maxTicks`, not by gallery size: `≤ maxTicks` images do one ordered scan; above that, a `COUNT` plus `maxTicks` indexed `OFFSET … LIMIT 1` seeks spread over `[0, total-1]` **inclusive**, so the oldest AND the newest matching photo are always on the strip. A plain `int(i * step)` sampler never reached `len-1` and silently dropped the most recent ticks.
2. `api/internal/server/images_controller.go`:
   - `api.GET("/images/time-bounds", …)` and `api.GET("/images/time-ticks", …)`; both share `parseImageFilterParams`, whose `CanViewProject` gate is their only authorization (covered by a non-member 403 test).
   - The bounds response IS `repository.ImageTimeBounds` — it already carries the wire tags, so there is no second shape to keep in sync.
   - `ticks` is always a JSON array, never `null`: the SPA reads `.length` unguarded, and a `null` there throws, resets the memo key and turns the failure into a refetch loop.
3. Tests:
   - `TestGetImageTimeBounds` — domain excludes the range; repo test proves the strip.
   - `TestGetImageTimeTicks` — ascending, exact count, and the endpoints asserted **exactly** (`sampled[last] == ticks[last]`); the previous `|| .After(ticks[0])` clause was tautological and could not catch the dropped-last-tick bug.
   - `TestImageSliderEndpointsRequireProjectAccess` — a non-admin, non-assigned caller gets 403 on both endpoints.

## Frontend

4. `ui/src/api/images.ts`: `ImageTimeBounds` + `timeBounds(params)`, `timeTicks(params)`.
5. `ui/src/pages/image/imageQueryLogic.ts`:
   - `timeBounds` / `timeTicks` refs with `loadTimeBounds()` / `loadTimeTicks()` mirroring `loadTagFacets`' key memo — plus a **latest-wins sequence guard**. Both loaders set the memo key before awaiting and used to assign unconditionally, so a slow response for an older filter overwrote the newer domain.
   - `resetTransientFilters` clears `rangeScopeAll` and `routeSortOrder` alongside the window. Leaving the context flag set with no window kept search/tags/orientation suspended behind a chip row that rendered nothing — filters silently off, no on-screen explanation.
   - `routeSortOrder` is the **view-local** sort, falling back to the persisted `preferredImageSortOrder`.
6. `ui/src/components/image/TimeRangeSlider.vue` — **not** native `<input type="range">`. The shipped control is `div` thumbs with manual pointer drag plus explicit ARIA:
   - `role="slider"`, `tabindex="0"`, `aria-valuemin/max/now/text`, and Arrow / PageUp / PageDown / Home / End handling. The original sketch's "keyboard-accessible native inputs" was the design intent; the pointer rewrite dropped it and this is the fix.
   - `touch-none` on the track — without it the browser claims a horizontal drag for page scrolling and cancels the gesture, so the slider never moved on a phone.
   - `setPointerCapture` plus `pointercancel` / `blur` teardown and an `onUnmounted` cleanup. The window listeners used to be removed only in `pointerup`: a cancelled gesture pinned `dragging` to a thumb, leaked both listeners and left a half-dragged range on screen that was never applied.
   - Geometry is measured once per gesture (with a `ResizeObserver` for popover sizing) instead of `getBoundingClientRect()` on every `pointermove`, which forced a reflow per mouse event because each `preview` emit dirties the panel layout.
   - Track and thumb centres are inset by the thumb radius so the end thumbs stay inside the popover panel; the domain is rounded to whole minutes so a thumb cannot land outside the strip the server drew.
7. `ui/src/components/image/ImagesHeader.vue`:
   - slider above the From/To inputs; popover open emits `timeBoundsNeeded`; new props `timeBounds`, `timeTicks`, `timeRangeSuspended`.
   - The inputs start **empty** when no range is applied. They used to be prefilled with the domain bounds, so editing only "From" silently committed `?to=<last photo>` on a panel whose "Clear time range" button was hidden because no range existed.
   - Typing is debounced (400 ms) and commits as a history `replace`, and an inverted range is clamped rather than sent (the backend answers `400 invalid_time_range`, which renders the grid as an error page).
   - A debounced keystroke carries a token: an inbound sync (props change or slider preview) bumps it, so a keystroke already in flight knows it was overtaken and does not write its stale value.
8. `ui/src/pages/image/Images.vue`:
   - `:time-bounds` / `:time-ticks` / `:time-range-suspended` + `@time-bounds-needed`.
   - `?from=` / `?to=` are validated as real instants before use. An unparseable bound used to reach the API and error the whole page; it now degrades to "no bound on that side".
   - `showTimespanAround` sets `?sort=oldestFirst` **in the route** instead of writing `preferredImageSortOrder`. Writing the persisted preference rewrote the user's global sort for every project, browser-back could not undo it, and the write tripped `watch(preferredImageSortOrder)` — firing a `loadImages` under the OLD range before `applyRoute` fired its own under the new one.
   - ONE chip row for every context, with a single `filters-pill`. The person view and the timespan view each rendered their own, so a URL carrying both showed two identical "Filters" pills and any testid selector matched two elements.

## Tests

- e2e `time-range.spec.ts` drives the real control: `data-testid="range-start-thumb"` / `range-end-thumb` with `page.mouse` (there are no `<input aria-label="Range start">` elements), plus a keyboard case.
- `playwright.config.ts` pins `timezoneId` (default `Europe/Berlin`). The Time popover works in local wall clock, so a spec asserting the ISO behind a typed "2026-08-01T00:00" otherwise passes only on a CEST host.
- The chip tests navigate with `&rangeScope=all`: the chip row is context-gated and a bare `?from=&to=` URL renders no chip to assert on.
- `resetTransientFilters` clearing the context flag is asserted by re-setting a window through the UI after clearing — a `page.goto` would remount into the active state and prove nothing.
- vitest: `tests/dateTimeUtil.spec.ts` keeps the four time-offset suites (`timeOffsetUpToDate`, `toWasmTimeOffsets`, `appliedTimeOffset`, `backendTimeToUnixSeconds`) alongside the new `datetime-local` ones. The BigInt-on-Postgres-microseconds regression guard in `toWasmTimeOffsets` killed the whole browser upload pipeline once; it stays.
- vitest: `applyPersonPause` (including the time range) is asserted once, in `personViewPause.spec.ts`, over the shared fixture — it used to be duplicated across two files with different coverage.

## Out of scope (v1)

- live count preview during drag; snap-to-photo stepping

## Open work

- The slider's `preview` emit only mirrors the thumbs into the inputs; a live tile count during the drag is still not implemented.
