## Shutterbase Downloader

## Prerequisites

An API Key is required to use the downloader.  
The API Key can be obtained by navigating to the the "My User" page and clicking on the "GET NEW API KEY" button.  
**Note:** Only one API Key can be active for your account at a time. Repeat the process to generate a new key. This will invalidate the previous key.

## Usage

```
./downloader --url <shutterbase api url> --key <api key> --project <project uuid> download [full|delta] <tag>
```

### Example

```
./downloader --url https://shutterbase.mxcd.de/api/v1 --key <INSERT API KEY> --project c3f2ea22-2e15-44d6-a926-69db76a60c86 download full vbo
```

**Note:** Windows users shoud use `downloader.exe` instead of `./downloader`

### Blocklist

Optionally, a blocklist can be provided to exclude certain files from the download.
The blocklist is a simple text file with one file name path per line.
The blocklist can be provided by using the `--blocklist` flag.

```
./downloader --url https://shutterbase.mxcd.de/api/v1 --key <INSERT API KEY> --project c3f2ea22-2e15-44d6-a926-69db76a60c86 --blocklist my-blocklist.txt download delta vbo
```
