# pull — copy a PROD sample into the local DEV environment

```sh
cd api
SHUTTERBASE_PROD_API_KEY=… DATABASE_PORT=5433 go run ./cmd/pull --project FSG26 --n 100 --public
```

Source: the REST API of `--source` (default `https://shutterbase.fsg.one`) with an API key
(`--api-key` or `$SHUTTERBASE_PROD_API_KEY`; `api/.env` may carry it). Target: the local
`DATABASE_*` / `S3_*` config (`api/.env`, env vars win — the dev Postgres runs on 5433 next to
the hub stack).

What it mirrors per image: the project (created by name if missing), the photographer (local
user named after the copyright tag), camera, upload (state kept, so reviewed uploads stay
reviewed), every tag with type/order/album flag, the tag assignments (types kept), EXIF, the
original and all renditions into the local bucket under the same storage id.

Flags: `--n` sample size, `--spread` (default on) picks evenly across the whole project so every
day and photographer shows up, `--tags a,b` narrows the source to images carrying all of them,
`--public` also assigns the reserved `public` tag (what the public gallery shows),
`--parallel` concurrent copies. Re-runs skip images that already exist locally.
