# Time-range slider in the Time popover

## Behavior

- Two-thumb slider domain = `[earliest, latest] capturedAtCorrected` of everything matching the current filters **minus the time range itself** (domain stays stable while dragging).
- Thumbs set `from`/`to`; existing datetime-local inputs remain directly below as the **manual override** — typing moves the thumbs, dragging updates them. Single source of truth: the route-driven filter state.
- Drag updates thumbs locally; commits happen on **release** (`change`, not `input`) → no request storm. Live tile-count preview deliberately out of scope v1.
- Slider hidden when the domain has <2 distinct timestamps; greyed while the chip-level suspension (`timeRangeSuspended`) is active.

## Backend (mirrors the facets endpoint)

1. `api/internal/repository/image.go` — add:
   ```go
   type ImageTimeBounds struct {
       Min *time.Time `json:"min"`
       Max *time.Time `json:"max"`
   }
   func (r *Repository) GetImageTimeBounds(ctx, parameters) (*ImageTimeBounds, error)
   ```
   - STRIPS `From/ToCapturedAtCorrected` defensively at entry (range can't be part of its own domain; repo test proves it)
   - `buildImagePredicates` + `Aggregate(ent.As(ent.Min(FieldCapturedAtCorrected,"min"), ent.As(ent.Max(...),"max")))` → `Scan` into struct slice
2. `api/internal/server/images_controller.go`:
   - route `api.GET("/images/time-bounds", s.getImageTimeBounds)` next to `/images/position`
   - handler: shared `parseImageFilterParams`; `emptyResult` → `{min:null,max:null}`; else call repo, return bounds
3. Tests:
   - extend `TestGetImagesTimeRange`: unbounded project → Min = cluster start, Max = newest base photo; with `Search "FSG_90"` → Min/Max exactly cluster edges; passing From/To params changes nothing (strip proof)

## Frontend

4. `ui/src/api/images.ts`: `interface ImageTimeBounds {min,max}` + `timeBounds(params)` GET
5. `ui/src/pages/image/imageQueryLogic.ts`: `timeBounds` ref + `loadTimeBounds()` mirroring `loadTagFacets`' key cache; params from `currentFilterInput()` minus time fields
6. New `ui/src/components/image/TimeRangeSlider.vue`:
   - two overlapped native `<input type="range">` (keyboard-accessible; no Quasar components anywhere in the app — off-pattern)
   - minute-step model over `[Date(min), Date(max)]`, selected-segment track div, domain-end date labels
   - props `min/max/from/to/disabled`; emits `change(from,to)` on release only; local thumbs init from props else domain edges; clamp manual inputs into domain
7. `ui/src/components/image/ImagesHeader.vue`: slider above the From/To inputs in the popover; popover open emits `timeBoundsNeeded` (same pattern as Tags' `facetsNeeded`); new prop `timeBounds`
8. `ui/src/pages/image/Images.vue`: `:time-bounds` + `@time-bounds-needed="loadTimeBounds()"`

## Tests

- e2e extend `time-range.spec.ts`:
  - popover opens → thumbs sit at expected domain ends under a narrowed filter
  - simulate drag (`evaluate` set value + input/change events) → URL gains from/to, tile count changes
  - then type into the From input → URL shows the typed instant (manual override wins)
- vitest unaffected (pure helpers unchanged); run full go/vitest/playwright suites

## Out of scope (v1)

- live count preview during drag; snap-to-photo stepping
