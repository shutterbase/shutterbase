## Shutterbase Downloader

## Prerequisites

An API key is required to use the downloader. Mint one via `POST /api/v1/api-keys`
(or the UI); the secret is shown **once** in the form `<keyId>.<secret>` — store it
safely. Authentication uses the `Authorization: ApiKey <keyId>.<secret>` header.

The shutterbase project id is also required. It can be found by navigating to the
project's detail page and copying the id from the URL.

## Usage

```
./downloader --url <shutterbase API base URL> --api-key <keyId.secret> --project <projectId> --parallelism <workerCount=1> --whitelist <tag1,tag2> download [full|delta]
```

`--url` is the API base, e.g. `https://shutterbase.fsg.one/api/v1`.

The `--whitelist` flag specifies which tags an image must have to be included. The
list is logically `AND`-concatenated (applied server-side). More than one tag can
be specified by separating them with commas. No spaces must be included.

The API key may also be supplied via the `SHUTTERBASE_API_KEY` environment variable
(and `--url`/`--project` via `SHUTTERBASE_API_URL`/`SHUTTERBASE_PROJECT_ID`).

### Example

```
./downloader --url https://shutterbase.fsg.one/api/v1 --api-key abc123def456ghi.0n9m8k7j6h5g4f3d2s1a0p9o8i7u6y5t --project qagr042y62aeptz --parallelism 3 --whitelist vbo download full
```
```
./downloader --url https://shutterbase.fsg.one/api/v1 --api-key abc123def456ghi.0n9m8k7j6h5g4f3d2s1a0p9o8i7u6y5t --project qagr042y62aeptz --parallelism 3 --whitelist vbo,Thursday download full
```

**Note:** Windows users should use `downloader.exe` instead of `./downloader`

### Blocklist

Optionally, a blocklist can be provided to exclude certain files from the download.
The blocklist is a simple text file with one computedFileName per line, provided via
the `--blocklist` flag.

```
./downloader --url https://shutterbase.fsg.one/api/v1 --api-key <keyId.secret> --project qagr042y62aeptz --blocklist my-blocklist.txt --whitelist vbo download delta
```

### Blacklist

Additionally to the whitelist, a `--blacklist` of tags can be specified. The list is
logically `OR`-concatenated and applied client-side: any image carrying one of these
tags is excluded.

```
./downloader --url https://shutterbase.fsg.one/api/v1 --api-key <keyId.secret> --project qagr042y62aeptz --whitelist vbo --blacklist internal,review download delta
```
