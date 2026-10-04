# Shutterbase seeder

`cmd/seed` fills a development database with **volume fixtures**: thousands of
photos spread over a real time span, so the time-range density strip, the tag
facets and the slideshow have structure to render instead of three rows.

It is not the test fixture builder. `seed.Seed` (internal/seed) creates the
smallest deterministic set a test needs to be deterministic — one project, one
upload, three images — and that is what the Go tests and `cmd/testserver` (the
Playwright harness) call directly. `cmd/seed` calls `seed.Seed` first and then
hands the returned manifest to the bulk loaders (`seed.LoadPhotos`) when you ask
for photos.

The loaders write through the raw ent client and **never go through the upload
pipeline**: no WASM, no S3 upload, no thumbnail generation, no AI queue, and no
`image_service.addDefaultTags`. That is why the seeder derives and assigns the
date and weekday tags itself — see [Tags](#tags). Every seeded image row carries
a synthetic `storageId`; no bytes are written to object storage. That is fine for
the gallery list, which reads the database, but there is no thumbnail or original
to open.

## Requirements

Postgres, reachable through the `DATABASE_*` config values. `just up` from the
repo root starts the matching stack and runs the seeder with no arguments.

`cmd/seed` always connects as Postgres — `DATABASE_TYPE` is not consulted here.

Every flag guard runs in one pre-flight pass **before the connection is opened**, so
a refused command needs no database at all and leaves no rows and no manifest
behind. Anything that gets past it does need a live database, `--dry-run`
included: the plan reports whether the database is already seeded.

## Quick start

```
# Base fixture only — what `just up` runs from the repo root
just up

# Base fixture plus 2000 clustered photos, plan first, then run
cd api
go run ./cmd/seed --photos 2000 --shape burst --dry-run
go run ./cmd/seed --photos 2000 --shape burst

# Evenly spread over the last 30 days instead of clustered
go run ./cmd/seed --photos 5000 --shape uniform --from -30d --to now

# A second, different fixture from the same command
go run ./cmd/seed --photos 2000 --seed 7

# A different manifest path and your own tag set
go run ./cmd/seed ./tmp/fixtures.json --photos 1000 --tags-file ./tags.tsv
```

Two things about the last example. `mkdir -p tmp` first: the manifest is written
next to its path, and a missing directory is reported only after the base fixture
is already committed. And a database that already has users takes the load path,
which needs a manifest at that exact path — `./tmp/fixtures.json` against an
already-seeded database is fatal, not a fresh seed.

The manifest path is the one optional positional argument. Flags and the path may
be written **in either order**:

```
go run ./cmd/seed ./seed-manifest.json --photos 500     # same run as below
go run ./cmd/seed --photos 500 ./seed-manifest.json
```

Plain `flag.Parse` stops at the first non-flag argument and would leave
`--photos 500` as a stray positional, silently discarding every loader flag. This
CLI re-enters the parser after each positional, so order does not matter. More
than one positional is refused.

## Usage

| Flag | Default | What it does | Constraint |
|---|---|---|---|
| `--photos N` | `0` | Seed N photos after the base fixture. `0` seeds the fixture only. | `N` must not be negative; a negative count is refused, not treated as `0`. `N > seedSoftCeiling` needs `--force`; `N > seedHardCeiling` is refused outright. The ceilings are checked for every `N`, not only above zero. |
| `--from` | 7 days before the resolved `--to` | Window start. RFC3339, `now`, or a relative offset such as `-7d`, `-36h`, `-90m`. | Requires `--photos`. `d` takes whole days only; write a fraction in hours (`-36h`) or minutes. A zero offset (`-0d`, `-0.0d`, `-0h`, `-0m`) is refused. A window longer than 365 days and a window that ends in the future both need `--force`. |
| `--to` | `now` | Window end. Same formats as `--from`. | Same as `--from`. |
| `--shape` | `burst` | How photos are distributed over the window: `burst` (5 golden-hour events per day of the window) or `uniform` (even spread landing on both bounds). | Requires `--photos`. Anything else is refused at the edge, never defaulted. |
| `--seed N` | `-1` (no seed) | Extra salt for the draw. Omit it and a re-run reproduces the previous run's draw; set it for a second, different fixture. | Requires `--photos`. `N >= -1`. |
| `--tags-file` | — | TSV of tag rows to seed **instead of** the generated set. | Requires `--photos`. An unreadable file, an empty set or a row naming a tag the seeder owns is an error, never a fallback to the generated set. |
| `--tag-count N` | `0` | Extra pool tags per photo. `0` keeps the documented 30/50/20 split over 1, 2 and 3; `1`-`3` pins every photo. | Requires `--photos` when it is above `0`. `0`-`3`; outside that the flag is refused, not clamped. |
| `--dry-run` | `false` | Print the plan (window start/end, day count, photos to add, whether the database is already seeded) and exit without writing anything. | Every flag guard still applies, so a dry run refuses exactly what a real run would — including the flags that need `--photos`. |
| `--force` | `false` | Required past the soft photo ceiling, past the 365-day window ceiling, and to seed a window that ends in the future. | Does **not** buy the hard photo ceiling. |

## Guards and refusals

Each of these refuses loudly — the tool never half-seeds and exits 0.

Every flag guard below runs in one pre-flight call **above the database
connection**, so a refusal costs nothing and leaves no rows and no manifest behind.
They are ordered — a flag's own value first, then the flags that need `--photos`,
then `--tags-file`, then the **photo ceilings**, then the window — and the first
refusal wins, so a command with two mistakes hears about the cheaper one.

The photo ceilings come before the window is even resolved, on purpose: they read
nothing but the count, so nothing about them has to wait for a window to exist, and
a refusal `--force` cannot buy outranks one it can. `seed --photos 300000 --from
2016-01-01` therefore hears about the hard ceiling instead of being told to pass
`--force` for a 4000-day window — advice that could never work, since 300000 is past
the hard ceiling whatever `--force` says.

| Guard | Refused when | Message | Runs at |
|---|---|---|---|
| `--shape` value | anything other than `burst` or `uniform` | `unknown --shape "gauss": want "burst" or "uniform"` | value check |
| `--tag-count` range | outside 0-3 | `--tag-count 40 is out of range: 0 keeps the 30/50/20 split, 1-3 pins every photo to that many` | value check |
| `--seed` range | below `-1` | `--seed -5 is not a valid seed: use -1 or higher (-1 means "no seed")` | value check |
| `--photos` sign | `--photos` below `0` | `--photos -1 is not a count: 0 is legitimate and means "seed the base fixture only", but a negative count would skip the load and report success anyway — pass --photos N with N above zero, or drop the flag to seed the base fixture alone` | value check |
| `--tags-file` pairing | given without `--photos` | `--tags-file needs --photos: it names the tag set a load draws from, and a load of zero photos draws none` | pairing check |
| Draw-flag pairing | `--shape`, `--seed` or `--tag-count` given without `--photos` | `--shape/--seed/--tag-count needs --photos: they shape the draw a load makes, and a load of zero photos makes none — pass --photos N, or drop --shape and --seed and --tag-count to seed the base fixture alone` (both lists name the flags actually passed, so `--shape uniform` on its own reads `drop --shape`) | pairing check |
| Window pairing | `--from` and/or `--to` given without `--photos` | `--from needs --photos: a window only says WHERE a load spreads the photos it was asked for, and zero photos means there is nothing to spread them over — pass --photos N, or drop the window to seed the base fixture alone` (the prefix names the flags actually passed: `--from`, `--to` or `--from/--to`) | pairing check |
| `--tags-file` parse | a missing file, a short row, an empty name, an empty description, a duplicate name, a reserved name, or a file holding no rows | `--tags-file: tags.tsv line 9: got 2 tab-separated column(s), want 3 — each row is name<TAB>displayName<TAB>description` (all seven forms are listed under [Tags](#tags)) | `--tags-file` parse |
| Time bound format | a bound that is not RFC3339, `now`, or an offset | `--from: "yesterday" is neither RFC3339, "now", nor a relative offset like -7d` | window |
| Time bound unit | a suffix other than `d`, `h` or `m` | `--from: "-5y": unknown unit "y" — use d (days), h (hours) or m (minutes)` | window |
| Fractional `d` offset | a `d` offset that is not a whole number of days | `--from: "-1.5d" is not a whole number of days: the d offset is applied as a whole number of days, so -1.5d would quietly mean -1d and -0.5d an empty window — spell the fraction in hours as -36h instead` | window |
| Zero offset | `-0d`, `-0.0d`, `-0h` or `-0m` | `--from: "-0d" is a zero offset: this bound would land on the other one and leave an empty window — drop the flag for the default 7-day window, or pass an offset that is not zero, like -7d` (the prefix names the bound actually passed, so a zero on `--to` reads `--to:`) | window |
| Window shape | `--to` before `--from`, or equal to it | `window ends -96h0m0s before it starts: 2026-03-05 00:00:00 +0000 UTC .. 2026-03-01 00:00:00 +0000 UTC`, and for the equal case `window is empty: 2026-03-05 00:00:00 +0000 UTC .. 2026-03-05 00:00:00 +0000 UTC` | window |
| Future window | `--to` lands after now, without `--force` | `window ends 24h0m0s in the future (2026-10-04T12:00:00Z > 2026-10-03T12:00:00Z) — pass --force to seed it anyway` | future window |
| Window length | the window spans more than 365 days, without `--force` | `window spans 366 days (2015-01-01T00:00:00Z .. 2016-01-02T00:00:00Z) — 367 calendar tags to write, all in ONE transaction that holds image_tags locks until it commits — past the ceiling of 365 days; pass --force to seed it anyway, or narrow --from/--to` | window length |
| Soft ceiling | `--photos` above 50 000 without `--force` | `60000 photos exceeds the soft ceiling of 50000; pass --force to proceed anyway` | photo ceiling |
| Hard ceiling | `--photos` above 250 000, with or without `--force` | `300000 photos exceeds the hard ceiling of 250000 — refusing regardless of --force` | photo ceiling |
| Positional args | more than one manifest path | `too many arguments — usage: seed [manifestPath] [loader flags]` | argument parse |
| Manifest on a load | a loader run against an already-seeded database finds no manifest at the path | `no manifest found — the loaders need the fixture identities it records. Delete it to re-seed from scratch.` | after the connection |
| Manifest on a load | the manifest exists but cannot be parsed | `cannot read the existing manifest — refusing to overwrite it. Delete it to re-seed from scratch.` | after the connection |

The manifest rows only exist on the load path: a database that already has users
with loader flags passed needs that file, and the same run against an empty
database does not.

Two messages in that table carry values no run reproduces: the **future window** and
the **window length** refusals print the run's own clock and the window's zone, so
their numbers are illustrative and only their shape is exact. The same holds for the
`--from`/`--to` shown in the window-shape row, which are ordinary sample bounds.
Everything else is byte-for-byte what the run prints.

`--shape`, `--seed` and `--tag-count` are refused without `--photos`, and so is
`--dry-run` when it is passed alongside them. The guards sit above the dry-run
branch rather than beside the `*photos > 0` they protect: a dry run reports the
refusals a real run would give, and a plan for a load that cannot happen describes
no run at all. The case that is worth a dry run — the base fixture on its own —
needs no loader flag, so bare `seed --dry-run` and `seed --dry-run --photos N` both
still work:

```
$ seed --dry-run --photos 10          # against an empty database
dry run — would seed the base fixture, then the photos

$ seed --shape uniform --dry-run
FATAL[invalid flag] --shape needs --photos: they shape the draw a load makes, and a
load of zero photos makes none — pass --photos N, or drop --shape to seed the base
fixture alone
```

The first line reads `dry run — loaders would extend the existing fixture` against a
database that already has users.

## Re-running

This is the property that makes the seeder safe to run again.

**Every entry point is idempotent.** Photos are matched by `computedFileName`
within the project (`FSG_LW%05d.jpg` for the burst shape, `FSG_W%05d.jpg` for the
uniform one), so an existing photo is skipped rather than inserted, and tag
assignments are matched by the `(image, tag)` pairs already recorded, so an
existing assignment is skipped rather than conflicting. Nothing is truncated and
no photo is deleted; the one delete a run performs is the stale reserved-name
cleanup, and it converges — see [Stale reserved tags](#stale-reserved-tags-are-cleared-on-every-run).

- **The same arguments again add nothing.** The photo names, the layout and the
  per-photo tag draws are all derived from the window and the photo's index, never
  from the wall clock, so the second run is a no-op against the database. It still
  rewrites the manifest file, with the same content.
- **Omitting `--seed` reproduces the previous draw on purpose.** Do not
  "randomise" it: the default `-1` is forwarded as a real salt, so the derivation
  is byte-identical every time and the re-run is genuinely a no-op. A random default
  would make every `just seed` reshuffle the fixture, and the run you are looking at
  would never be the run you just did.
- **`--seed N` is how you get a second, different fixture from the same command.**
  It salts the layout and the draws, producing photos with the same names-and-count
  shape but different timestamps and tag sets. Note that `0` is a real salt value
  distinct from the unset default: internally `0` means "no salt" (the unsalted
  legacy stream), `-1` means the default salted stream. Both are stable across runs.
- **A top-up with a larger `--photos` grows the set, it does not duplicate it.**
  Indexes 0..N-1 keep their existing rows and only the new tail is inserted. A
  *smaller* `--photos` is not a shrink: the extra photos from the previous run stay
  in the database.

The "already seeded" test is simply "the `users` table has at least one row", and
the run mode follows from it:

| Database | Loader flags | What runs |
|---|---|---|
| empty | none | base fixture, manifest written |
| empty | any | base fixture first (the loaders need its project, users and Default tag), then the loaders |
| has users | none | nothing — `database already has users and no loader flags — skipping seed` |
| has users | any | the loaders, against the manifest already on disk |

"Any loader flag" is not a second condition, though: the pairing guards above have
already refused every request-shaped flag that arrived without a count, so by the
time the run mode is chosen a request exists exactly when `--photos` is above
zero. `--photos 0` is the same as no `--photos` at all — the base fixture only —
and a run that passes `--shape`, `--seed`, `--tag-count`, `--tags-file`, `--from`
or `--to` without a count never reaches this table.

## The window

`--from` and `--to` each accept three forms:

| Form | Examples | Meaning |
|---|---|---|
| RFC3339 | `2026-03-05T00:00:00Z` | an absolute instant |
| `now` | `now` | the current time |
| relative offset | `-7d`, `-36h`, `-90m` | that far **before** the reference instant. `d` days (whole numbers only), `h` hours, `m` minutes |

Offsets are subtractive only — the value must start with `-` — and the unit
suffix is mandatory, and the amount must not be zero. `m` means minutes, not
months, because in this domain `m` reads as months.

`h` and `m` take a fraction and take it exactly, because a duration is exact to
the digit: `-1.5h` is ninety minutes, `-0.5m` is thirty seconds. `d` cannot. A
day offset is applied as a whole number of days, so `-1.5d` used to mean `-1d`
and `-0.5d` an empty window — a complaint about a window nobody wrote, for a
request that was perfectly well formed. It is refused rather than rounded, and
the refusal names the spelling that works:

```
$ seed --photos 10 --from -1.5d
FATAL[invalid flag] --from: "-1.5d" is not a whole number of days: the d offset
is applied as a whole number of days, so -1.5d would quietly mean -1d and -0.5d an
empty window — spell the fraction in hours as -36h instead
```

**A zero offset is refused for the same reason, and reaches the same empty window
without a fraction in it.** `-0d`, `-0.0d`, `-0h` and `-0m` all resolve to the
reference instant, so `--from -0h` puts the two bounds on the same timestamp and
the run is turned away by `window is empty: …` — a complaint about a window
nobody wrote, naming no flag at all. The bound is named instead:

```
$ seed --photos 10 --from -0h
FATAL[invalid flag] --from: "-0h" is a zero offset: this bound would land on the
other one and leave an empty window — drop the flag for the default 7-day window,
or pass an offset that is not zero, like -7d
```

Dropping the flag is the fix it names, because an unset bound falls back to the
default seven days rather than to the other bound.

Half-open flags fall back rather than erroring: `--from -7d` on its own keeps the
default end (`now`), and `--to now` on its own keeps the default start (seven days
back). The default window is therefore **the seven days ending now**.

**A relative `--from` resolves against the resolved `--to`, not against now.**
That is what makes the mixed form coherent:

```
--from -2d --to 2026-03-05T00:00:00Z   # the two days ending 5 March
```

Resolving `-2d` against now instead would mean "2 days before today .. 5 March",
which for any `--to` in the past is an inverted window and is refused.

The window must be non-empty and non-inverted, and must not end in the future
without `--force` — the loaders spread backwards from the window end precisely so
nothing is dated ahead of now, which recency ordering, slideshows and EXIF export
all read.

**A window may not span more than 365 days without `--force`.** The photo count
does not bound this: a window is walked once per calendar date it spans, whatever
`--photos` says, so `--photos 100 --from -366d` asks for a hundred photos and 367
date tags. Base fixture plus 100 photos held constant, measured on the dev machine
against Postgres 18 over localhost:

| Window | Wall clock | Calendar tags | Where the time went |
|---|---|---|---|
| 7 days | 2.05s | 8 | — |
| 365 days | 2.95s | 366 | +0.9s inside the calendar transaction |
| 3 900 days | 17.63s | 3 901 | +15.6s inside **one** transaction holding `image_tags` row locks from the first INSERT to the COMMIT |

The wall clock matters less than the lock: the walk holds `image_tags` locks for
its whole duration, so it blocks the app's own tag writes and reports nothing
until it commits. The ceiling is an order of magnitude above the case it bounds —
a real event fixtures a handful of days and the longest window anyone documents is
`-30d` — and **there is no hard tier above it**, for the same reason the
future-window guard has none: what a long window costs is a slow transaction
holding locks, not a destroyed database, so a decade of density data is a
legitimate thing to want and `--force` is the way to ask for it.

The comparison is on the **span**, so the boundary sits where the flag value is:
`-365d` passes, `-366d` does not. A 365-day span still writes 366 date tags, both
endpoints included, and the refusal reports that count, so the number the run pays
in is never a number the operator has to guess at.

**Both endpoints count, so a window that crosses midnight gets both dates.** The
walk steps by calendar DATE, not by elapsed hours, and truncates to the date
before it starts: a window from 22:45 to 00:45 is 2 hours long and spans 2 dates,
so it writes 2 date tags and every photo in its final hour carries one. Stepping
24h from the window's own time of day instead would walk 22:45 → 22:45 tomorrow,
stop as soon as that passed the end, and never visit the date the window ends on
— a photo captured in the last hour would have come back with a weekday and no
date at all, next to its neighbours in the same run that had one.

```
$ seed --photos 2 --shape uniform --from 2026-03-04T22:45:00Z --to 2026-03-05T00:45:00Z
```

Both photos land on a bound, and each carries its own date — the one captured at
`2026-03-05 00:45:00+00` gets `20260305`, not `20260304`:

```
computed_file_name | captured_at_corrected  | date tag
FSG_W00001.jpg     | 2026-03-04 22:45:00+00 | 20260304
FSG_W00000.jpg     | 2026-03-05 00:45:00+00 | 20260305
```

A sub-day window that stays on one side of midnight spans one date and writes one
date tag; `--from -1h --to now` therefore writes one, or two once the run is
past 23:00 local and the hour crosses midnight.

**`burst` (default)** places 5 golden-hour events per day across however many whole
days the window spans (floored at 1), centred on 07:00, 08:00, 12:00, 17:00 and
18:00 local plus up to a minute of jitter, each spanning 30-90 minutes. Raw burst
sizes (10-50 photos) are normalised to exactly `--photos N` by largest-remainder
apportionment, so a count lands where you asked for it and the most recent day is
not starved. When `N` is smaller than the number of bursts, the largest bursts are
kept — the busiest shooting days — rather than truncating to the oldest ones. Every
photo is clamped into `[--from, --to]`.

**`uniform`** spreads photos evenly and lands on **both** bounds: the interval is
`span / (N - 1)`, not `span / N`, so nothing is left a gap short at the far end.
A single photo sits at the newest end (`--to`).

In both shapes `capturedAtCorrected` is the value the window places, and
`capturedAt` is 37 seconds earlier — the seeded camera's clock drift.

## Tags

Three sources feed every seeded photo. `--tags-file` replaces the generated set;
the calendar tags are always added on top.

**1. The generated team tags (default).** 80 tags, 4 per country across 20
countries, built deterministically from tables in `internal/seed/teamtags.go`.
Names are pipe-triples (`car_000|tid_000|AT Daxstein HS`), display names are
`car_000`, descriptions name a team and its institution, and every row is
`type=manual`. They exist so the facets have 80 names to group by — one
interesting group and eight flat ones is not a fixture.

**2. `--tags-file`.** A TSV of `name<TAB>displayName<TAB>description`, one tag per
line. Blank lines — including ones holding only whitespace — and lines starting
with `#` are skipped.

**All three columns are required.** A row with fewer is refused rather than padded,
because each way of padding was a real failure: a missing description became an ent
validator error from inside the loader, after the base fixture was already
committed, and a missing `displayName` became a blank chip in the tag filter with
nothing in the file to explain it. Columns *beyond* the third are ignored, so a file
exported with a trailing delimiter still loads.

`displayName` may be present but **empty** — the schema makes it optional — and then
falls back to the name. Only the missing column is an error. `description` may NOT be
empty: `image_tags.description` is `NotEmpty` in the schema. `name` may not be empty.

**A name the seeder owns is refused too.** The seeder is not the only writer, and a
tag file is not the only tag set, so five shapes of name are reserved:

| Reserved name | Why |
|---|---|
| `Default` | the seeder creates it and assigns it to every photo as a `type=default` assignment |
| `internal` | it marks photos kept out of slides and EXIF exports, and `internal/exif` strips it from every export |
| any `YYYYMMDD` | it is a calendar tag the seeder derives from each photo's capture time |
| any English weekday | the same, derived from each photo's own capture time |
| any `$`-prefixed name | a `$` name is a template the app renders on upload, not a tag a file can seed |

The `$` rule refuses **any** name starting with `$`, not only the `$DATE` and
`$WEEKDAY` this package ships, and the reason is two-sided. `addDefaultTags`
renders every `type=template` row and skips a name it cannot render, while
`ensureTags` hardcodes `type=manual` on create — so a file row would be a tag the
app never renders — and on a name that already exists as a template the
find-or-create would silently overwrite that template's description, which is the
one the app reads. A `$` inside a name is fine: `my$tag` is not reserved.

The calendar shapes are refused whether or not the date is inside the window: a
row naming a date a later, wider run would derive is the same collision waiting
to happen. Names that only look reserved are not: `near2026`, `2026100` (seven
digits) and `Thursdayish` all load.

A file row and the seeder's own path would both write the same `(image, tag)` pair
for a reserved name, `imagetagassignment` carries a unique index on it, and the
second write dies on the first 500-row chunk — long after the file looked correct.
Refused rather than dropped, like every other rule here: a row quietly skipped is a
tag set that is not the file the operator wrote.

A duplicate name is refused rather than
deduped, because the writer is find-or-create and the second row would silently
overwrite the first. A missing file, a file holding no rows, a short row, an empty
name, a reserved name, an empty description and a duplicate name all fail:

```
--tags-file: read tag file: open ./tags.tsv: no such file or directory
--tags-file: tags.tsv line 12: empty tag name
--tags-file: tags.tsv line 9: got 2 tab-separated column(s), want 3 — each row is name<TAB>displayName<TAB>description
--tags-file: tags.tsv line 8: "fsa_beta" has an empty description — image_tags.description is NotEmpty
--tags-file: tags.tsv line 7: "Podium" already defined on line 4 — a tag set cannot hold it twice
--tags-file: tags.tsv line 6: "Default" is reserved — the seeder creates it and assigns it to every photo as a type=default assignment; drop the row
--tags-file: tags.tsv holds no tag rows — refusing to seed an empty set
```

The `--tags-file: ` prefix is cmd/seed's; the message after it comes from the parser. All
seven are raised before anything is written, so a bad file costs nothing. The
reserved check runs before the duplicate check, so a file that repeats `Default`
gets the reserved message rather than "already defined on line 1".

A file that cannot be read is an **error**, never a silent fallback to the
generated 80 tags.

**3. The date and weekday tags.** Every photo also carries the day it was shot
(`20261002`) and the weekday (`Thursday`), derived from the photo's own corrected
capture instant rather than drawn. `seed.Seed` ships both the `$DATE` and the
`$WEEKDAY` template tags, and the seeder creates the derived rows as
`type=default`.

A date name carries its own date, so its description is the long form —
`Wednesday, 4 March 2026`. A weekday name does not: `Thursday` is not a date, so
its description is the short form rendered from a **fixed reference week**
(2024-01-01, a Monday), which means the same seven strings in every window, in
every zone, on every machine:

| Name | Description |
|---|---|
| `Monday` | `Mon 01 Jan 2024` |
| `Tuesday` | `Tue 02 Jan 2024` |
| `Wednesday` | `Wed 03 Jan 2024` |
| `Thursday` | `Thu 04 Jan 2024` |
| `Friday` | `Fri 05 Jan 2024` |
| `Saturday` | `Sat 06 Jan 2024` |
| `Sunday` | `Sun 07 Jan 2024` |

It used to be rendered from the window instead, and fell back to the bare name
whenever the window held no day matching that weekday — which is most of the
time, so the promised label almost never appeared. Reading the date off a fixed
week also removes the other half of the problem: a label that moved between runs
would make the calendar pass re-describe all seven weekday rows on every run and
they would never settle. **Existing weekday rows are therefore re-described once,
on the next run**, and settle after that.

That type is load-bearing. `image_service.findOrCreateDefaultTag` filters on
`type=default`, and `image_tags` carries a unique index on `(name, project_id)` — so
a calendar tag created as anything else would be invisible to the next real upload,
whose `$DATE` render would then INSERT the same name and die on that unique index.
Creating them the way the app would is what lets the two agree. The seeder promotes
an existing same-named row to `type=default` if it finds one.

The date and weekday tags do **not** count against `--tag-count`. That flag pins
the *random* pool at 1-3 so a run has a predictable tag load; the date and the
weekday are unconditional facts about the photo, so counting them would mean
`--tag-count 1` yields one tag that is sometimes the date and sometimes not.

So a default photo carries: the `Default` tag (a `type=default` assignment), its
date, its weekday, and 1-3 tags drawn from the pool of 80 generated team tags plus
`Tag00`-`Tag09`. With `--tags-file` the pool is the file's rows plus `Tag00`-`Tag09`.
Each photo's draw comes from a stream keyed on that photo's own index — or on its
image id for the backfill — so re-running the same command against the same
database reproduces it exactly.

A pinned `--tag-count` is a target, not a floor: after the load, every image in the
project holding fewer pool tags than the run targets is topped up — including the
base fixture's three, which would otherwise sit in the gallery carrying only
`Default` next to thousands with a full tag set. Two groups are skipped: the
midnight control cluster (`FSG_90xx`, deliberately untagged) and the loader's own
photos from this run. The backfill is append-only, so it can only reach a photo an
earlier run left short; it can never lower an over-pinned photo.

### Where the calendar names come from

Both names are derived **in the window's own location**, and through one helper
shared by the loader and the tag backfill, so the two paths that *write* a date
cannot disagree about a photo. The zone is load-bearing: the loader holds the
instant it just computed in the window's zone, while a photo read back out of
Postgres arrives in the session zone (`DATABASE_TIMEZONE`, `UTC` by default). For
a photo within an hour or two of midnight those two readings format as different
calendar dates, which is why the zone is part of the name and not an
afterthought.

A relative offset resolves against `time.Now()`, so the window's location is the
machine's local zone and the two readings agree — that is the ordinary case, and it
is where the divergence is easiest to miss. The divergence appears as soon as a
bound carries an explicit offset:

```
$ seed --photos 2 --shape uniform --from 2026-10-03T22:30:00Z --to 2026-10-03T22:50:00Z
# machine TZ Europe/Berlin, DATABASE_TIMEZONE UTC
computed_file_name | captured_at_corrected | in Berlin | calendar tags
FSG_W00001.jpg     | 2026-10-03 22:30:00+00 | 2026-10-04 00:30 | none
FSG_W00000.jpg     | 2026-10-03 22:50:00+00 | 2026-10-04 00:50 | none
```

Both photos came out with **no date and no weekday**, because the window's zone
says 2026-10-04 and the rule the cleanup below applies reads 2026-10-03, so the
tags the loader had just written looked stale and were deleted.

Under relative bounds the same boundary hour keeps both tags: `--photos 400
--shape uniform --from -2d --to -1d` on this machine put `FSG_W00002.jpg` at
`2026-10-02 23:59:25+00`, which is Berlin `2026-10-03 01:59:25`, and it came out
carrying `20261003` and `Saturday`. **So write the window's bounds with no offset
when you want every photo dated.**

One difference from a real upload is deliberate: the seeder reads the **raw**
corrected instant and does **not** apply the app's `DATE_TAG_HOUR_OFFSET`
(default `-3h`). That shift exists because an event's late-night photos belong to
the previous day — a real photography concern — and a fixture has no reason to
model it, so applying it would make the seeder disagree with the plain reading of
its own timestamps. A photo captured before 03:00 local therefore gets the
previous day from a real upload and its own date from the seeder.

### Stale reserved tags are cleared on every run

Before it measures anything, the tag backfill deletes every assignment to a
reserved name that this build's own writers would not have put there, and rebuilds
that photo's `imageTags` read model. It runs automatically on **every** invocation
that runs the loaders — any `--photos N` with `N` above zero — with no flag, and
it decides by name:

| Assignment | Kept? |
|---|---|
| `Default` | always — every create path writes it, and nothing would put it back |
| the photo's **own** date `YYYYMMDD` | kept |
| the photo's **own** weekday | kept |
| any other `YYYYMMDD` or weekday | deleted — a calendar name on a photo that is not its own came from the draw |
| `internal` on `FSG_0000.jpg`..`FSG_0002.jpg` | kept — `seed.Seed` puts it there on purpose, and an e2e spec counts on it |
| `internal` anywhere else | deleted |
| anything not in the reserved set | untouched |

The photo's own date and weekday are read from the instant as it comes back out
of Postgres, **not** normalised into the window's location — so a window carrying
an explicit offset does not agree with them, and the loader's own calendar tags
are deleted rather than kept. See
[Where the calendar names come from](#where-the-calendar-names-come-from) for a
reproduction. That is the one place where the cleanup is stricter than intended;
everything else about it behaves as the table says.

**Why it deletes rather than documenting the divergence:** `internal/exif/inject.go`
strips `internal` from every export, so a photo left holding it is excluded from
every EXIF export and every slideshow. Nothing else in the seeder removes an
assignment — the backfill only ever inserts — so without this pass **no seeder
invocation can clear it**, and the pool filter cannot either: that stops new writes
and does not touch a row already there. A database seeded by an older build
converges on the next run instead.

**What an operator can lose:** a photo hand-tagged `internal` in a seeded dev
database, to keep it out of a slideshow, loses the tag on the next run. That is the
price of a name the seeder documents as its own, and `--tags-file` refuses the row
outright for the same reason. The blast radius is bounded by construction — only
reserved names, only where the current build's writers would not put them, only in
the seed project — and a second run finds nothing stale and writes nothing.

## Manifests

The manifest is a JSON file recording every id a test or a later run needs, plus
the `referenceNow` the fixtures derive from:

| Field | What it holds |
|---|---|
| `referenceNow` | the instant every time-sensitive fixture value derives from; the default window ends here |
| `project` | the seed project id — every photo row carries it |
| `users` | role key → user id (`admin`, `user`, `projectAdmin`, `projectEditor`, `projectViewer`) |
| `roles` | role key → project-scoped role id |
| `cameras` | `fresh` / `stale` → camera id; the loaders file photos against `fresh` |
| `tags` | tag name → tag id, for every tag the run created or resolved, including the team tags and the calendar tags, so a name on a photo is always resolvable |
| `offsets` | `fresh` / `stale` → time offset id |
| `upload` | the upload every photo row is filed under |
| `images` | every image id, base fixture and loaded |
| `driftSeconds` | the seeded camera's clock offset |

Default path `./seed-manifest.json`, relative to the working directory
(`api/seed-manifest.json` under `just up`). It is gitignored, along with
`testserver-manifest.json`. Pass a different path as the one positional argument.

It is written **after every phase**: once before the loaders run (so the base
fixture is on disk even if a later phase dies) and again after they finish. The
write goes to a temp file and is renamed into place, so an interrupted run leaves
either the old manifest or the new one, never half a document.

**A loader run requires that file.** On a database that already has users, with any
loader flag, a missing or unparseable manifest is fatal. The loaders need the
project, the upload, `cameras["fresh"]`, `users["projectEditor"]` and the `Default`
tag id from it; rebuilding those from the database would mean re-deriving the base
seed's identities — the work `seed.Seed` does — inside a path whose whole purpose
is to extend an existing file. A bad manifest says so by name:

```
no "Default" tag in the manifest — the seed project is missing its Default tag, or the manifest was merged from a hand-edited file
the manifest records no upload — every photo row carries all four, and an empty one is an empty-string foreign key that aborts the chunk with a bare constraint error
```

The upload is checked for *ownership*, not just existence: a stale manifest whose
project was recreated resolves an upload row, and the photos would land in one
project while being filed under another's upload, so a gallery filtering by upload
would show a different set than one filtering by project.

Note that an identical re-run leaves the database unchanged and does not grow the
manifest either: an image id already in `images` is not recorded a second time, so
the array holds one entry per photo and matches the row count. Measured over
`--photos 20`, the same command again, then `--photos 40`: 43 entries in the
manifest, 43 distinct, 43 rows in `images`.

## Guardrails and limits

The photo ceilings are measured, not guessed: 15 023 photos seeded in 2m50s on the
dev machine. The soft ceiling sits an order of magnitude above a comfortable run
(50 000) and the hard ceiling well beyond anything a dev database needs (250 000).
**Raise them with measurement, never with optimism.** The 365-day window ceiling is
measured the same way — see [The window](#the-window).

Every ceiling is checked in the pre-flight, **above the connection and above
anything that writes**, so a refused oversized run leaves nothing behind: no rows,
no manifest, no half-seeded base fixture. That order is the point of the guard. The
ceilings used to sit below `seed.Seed` and below the first manifest write, so
`--photos 60000` on an empty database seeded the whole base fixture, published a
manifest for it, and only then refused — a run reporting failure over a database it
had just changed.

Work is committed in chunks of 500 photos, one transaction per chunk. A chunk that
fails leaves photos created without assignment rows, which the assignment
backfill makes safe to resume — re-run the same command.

What a run writes per photo: one `images` row (with its denormalized `imageTags`
read model), one `type=default` assignment for `Default`, and one
`type=manual` assignment per extra tag. Two names stay untouched by
deliberation. The `internal` tag, which `seed.Seed` puts on the third base image
so it stays in the gallery but reaches neither a slideshow nor an EXIF export, is
kept **on `FSG_0000.jpg`..`FSG_0002.jpg` only** and deleted everywhere else — see
[Stale reserved tags](#stale-reserved-tags-are-cleared-on-every-run). And the
`FSG_90xx` name prefix, which the tag backfill skips so the time-range filter keeps
an untagged control group. Nothing in `internal/seed` creates an `FSG_90xx` photo —
the prefix is the only handle the skip has, and it keeps working on a project seeded
from somewhere else entirely.
