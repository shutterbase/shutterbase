# shutterbase UI — design system & migration spec

The visual language is **locked**. This file is the single source of truth for migrating
every remaining page/component onto it. When restyling a file: **preserve all logic,
props, emits, refs, structure, accessibility and behavior — change only classes/markup
for styling.** Never touch the logo/wordmark assets or the potato/ghost mascot images.

## Identity
"Photographic instrument" — editorial, high-contrast, generous negative space, a confident
cobalt accent on a cool-neutral canvas. Dark is the default theme; light is first-class.
Inter for UI, JetBrains Mono as the technical "readout" voice.

## Tokens (Tailwind) — the ONLY colors allowed
- `primary` (50→950): cool-neutral. Surfaces, borders, structure, body/muted text.
- `accent` (50→950): cobalt. Actions, selection, links, focus, active state. **One** accent.
- `surface` / `surface-dark` / `surface-muted` / `surface-dark-muted`: raised panels/cards/inputs.
- `success` / `warning` / `error`: semantic only (with icon/text, never color-alone).

**BANNED — replace on sight (these are the migration targets):**
- `gray-*` → `primary-*`
- `secondary-*` → does not exist; was rendering invisible. Buttons → see button recipes; text → `primary-*`.
- `indigo|red|green|blue|slate|zinc-*` → the matching token (`red`→`error`, `green`→`success`, `indigo|blue`→`accent`, `slate|zinc`→`primary`).
- `bg-white` / `bg-black` where a `dark:` flip is needed → `bg-surface dark:bg-surface-dark` (Quasar ships `.bg-white !important`; Tailwind `bg-white` can't be flipped). `text-white` is fine ONLY as static text on a colored/dark fill.

## Type
- Page heading: `<h1 class="display text-3xl text-primary-900 dark:text-white">`. (`.display` = tight statement sans.)
- Optional kicker above a heading: `<p class="label-mono text-accent-600 dark:text-accent-400">Section</p>`.
- Field label / column header / metadata caption: `class="label-mono text-primary-500 dark:text-primary-400"` (`.label-mono` = mono uppercase tracked).
- Numbers/IDs/times/EXIF/counts: add `.font-data` (mono tabular).
- Body: `text-primary-700 dark:text-primary-300`. Muted: `text-primary-500 dark:text-primary-400`.
- Links: `text-accent-600 hover:text-accent-500 dark:text-accent-400`.

## Component recipes (copy these class strings)

**Text input / `<select>` / `<textarea>`**
```
h-10 w-full rounded-md border border-primary-200 bg-surface px-3 text-sm text-primary-900 placeholder:text-primary-400 transition-colors hover:border-primary-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-100 dark:placeholder:text-primary-500 dark:hover:border-primary-600
```
(textarea: drop `h-10`, add `py-2.5`.)

**Primary button (accent)**
```
inline-flex items-center justify-center gap-1.5 rounded-md bg-accent-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-accent-500 active:bg-accent-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:ring-offset-2 focus-visible:ring-offset-surface dark:focus-visible:ring-offset-primary-950 disabled:opacity-50
```

**Secondary button (quiet bordered)**
```
inline-flex items-center justify-center gap-1.5 rounded-md border border-primary-200 bg-surface px-4 py-2 text-sm font-medium text-primary-700 transition-colors hover:border-primary-300 hover:text-primary-900 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200 dark:hover:border-primary-600 dark:hover:text-white
```

**Destructive button**
```
inline-flex items-center justify-center gap-1.5 rounded-md border border-error-300 bg-error-50 px-4 py-2 text-sm font-medium text-error-700 transition-colors hover:bg-error-100 dark:border-error-800/70 dark:bg-error-950/40 dark:text-error-300 dark:hover:bg-error-950/70
```
(solid danger when it's the primary action: `bg-error-600 text-white hover:bg-error-500`.)

**Card / panel**
```
rounded-lg border border-primary-200 bg-surface p-6 shadow-panel dark:border-primary-800 dark:bg-surface-dark dark:shadow-panel-dark
```

**Dialog**: scrim `fixed inset-0 bg-primary-950/60 backdrop-blur-sm`; panel = the card recipe, `max-w-md`, with a `label-mono` kicker + `display` title. Provide a visible close (X) and Esc/cancel.

**Page container**: `mx-auto max-w-7xl w-full px-4 sm:px-6 lg:px-8` (forms/detail: `max-w-3xl`). Vertical rhythm in 4/8 steps; section gaps `space-y-6`/`space-y-8`.

**Badge/chip** (e.g. tags): `inline-flex items-center gap-1 rounded-md border border-primary-200 bg-surface px-2 py-0.5 text-xs font-medium text-primary-700 dark:border-primary-700 dark:bg-surface-dark dark:text-primary-200`; accent variant swaps to `accent`.

**Empty state**: viewfinder-framed art (`<div class="relative"><CornerMarks/><img …/></div>`), `label-mono` kicker, `display` headline, muted sub. Keep the mascots.

## Already migrated (do NOT edit — reference these for patterns)
`Login.vue`, `Images.vue`, `image/ImagesHeader.vue`, `image/ImageGridTile.vue`, `image/ImagesFooter.vue`, `Table.vue`, `upload/ImageUploadList.vue`, `MainLayout.vue`, `layout/navbar/DarkMode.vue`, `CornerMarks.vue`, `tailwind.config.js`, `css/tailwind.css`.

## Checklist per file
- [ ] No banned tokens remain (`grep gray-/secondary-/indigo-/red-/...`).
- [ ] Inputs/buttons/cards/labels use the recipes above.
- [ ] Works in BOTH themes (every surface/text/border has a `dark:` pair); AA contrast.
- [ ] Logic/props/emits/behavior unchanged; mascots/logo untouched.
- [ ] Focus-visible states present; clickable elements `cursor-pointer`.
