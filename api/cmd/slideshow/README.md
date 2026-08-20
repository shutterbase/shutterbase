# Shutterbase Slideshow

Renders a slideshow MP4 from shutterbase images: Ken Burns pan/zoom per image,
crossfade transitions, and a blurred background fill for images that don't
match the output aspect ratio.

## Prerequisites

- **ffmpeg on the PATH** (`brew install ffmpeg` / `apt install ffmpeg`).
- For API mode: an API key (`<keyId>.<secret>`, minted via `POST /api/v1/api-keys`
  or the UI — the secret is shown once) and the project id from the project
  detail page URL.

## Usage

```
./slideshow --url <API base URL> --api-key <keyId.secret> --project <projectId> [flags]
./slideshow --input-dir <folder of images> [flags]
```

| Flag | Env | Default | Description |
|---|---|---|---|
| `--url, -u` | `SHUTTERBASE_API_URL` | — | API base, e.g. `https://shutterbase.fsg.one/api/v1` |
| `--api-key, -k` | `SHUTTERBASE_API_KEY` | — | `<keyId>.<secret>` |
| `--project` | `SHUTTERBASE_PROJECT_ID` | — | project id |
| `--whitelist` | `SHUTTERBASE_WHITELIST` | — | comma-separated tags, AND-applied server-side |
| `--blacklist` | `SHUTTERBASE_BLACKLIST` | — | comma-separated tags, OR-excluded server-side |
| `--input-dir` | `SHUTTERBASE_INPUT_DIR` | — | render a local folder (`.jpg/.jpeg/.png`, sorted by filename) instead of the API |
| `--output, -o` | `SHUTTERBASE_OUTPUT` | `slideshow.mp4` | output file |
| `--limit` | — | all | render at most N images in slideshow order |
| `--shuffle` | — | off | randomize image order (applied before `--limit`, so the limit takes a random sample) |
| `--resolution` | — | `1920x1080` | output size `WIDTHxHEIGHT` |
| `--fps` | — | `30` | frame rate |
| `--show-seconds` | — | `6` | seconds each image is fully visible |
| `--transition-seconds` | — | `1.5` | crossfade length, additive (0 = hard cuts) |
| `--ken-burns` | — | `subtle` | `subtle` \| `medium` \| `strong` \| `off` |
| `--ken-burns-variants` | — | all | subset of `zoom-in,pan-right,zoom-out,pan-left`; one is picked randomly per image, never twice in a row |
| `--background` | — | `blur` | `blur` \| `black` fill behind non-matching aspect ratios |
| `--parallelism` | `SHUTTERBASE_PARALLELISM` | `min(4, cores/2)` | parallel downloads / ffmpeg runs |

Timing: each image is fully visible for `--show-seconds`; crossfades add
`--transition-seconds` on top, so a new image starts every
`show + transition` seconds (matching the web app's slideshow).

API mode plays images in `capturedAtCorrected` ascending order and uses the
2048px renditions.

### Examples

```
./slideshow --url https://shutterbase.fsg.one/api/v1 --api-key <keyId.secret> \
  --project qagr042y62aeptz --whitelist vbo --blacklist review \
  --show-seconds 5 --transition-seconds 1 -o vbo.mp4
```

```
./slideshow --input-dir ./downloads/vbo --ken-burns strong --background black \
  --resolution 3840x2160 -o vbo-4k.mp4
```

**Note:** Windows users should use `slideshow.exe` instead of `./slideshow`

## Known limitations

- Images tagged `internal` (including as a combo-tag part like `trip|internal`)
  are always skipped in API mode, exactly like the web slideshow.
- Local files with an EXIF orientation tag render unrotated (API renditions are
  already pixel-upright).
- Images smaller than the output resolution are upscaled and will look soft.
- No audio track; the output is silent.
