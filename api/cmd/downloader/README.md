## Shutterbase Downloader

## Prerequisites

A shutterbase account is required to use the downloader. For authentication, the email address and password of the shutterbase account are being used.  
Furthermore, the shutterbase project id is required to download the files. The project id can be found by navigating to the project's detail page and copying the id from the URL.

## Usage

```
./downloader -url <shutterbase url> --email <email> --password <password> --project <projectId> --parallelism <workerCount=1> --whitelist <tag1,tag2> download [full|delta]
```

The `--whitelist` flag is used to specify which tags an image must have to be included. The list is logically `AND`-concatenated. More than one tag can be specified by separating them with commas. No spaces must be included.

### Example

```
./downloader --url https://shutterbase.fsg.one --email <email> --password <password> --project qagr042y62aeptz --parallelism 3 --whitelist vbo download full
```
```
./downloader --url https://shutterbase.fsg.one --email <email> --password <password> --project qagr042y62aeptz --parallelism 3 --whitelist vbo,Thursday download full
```

**Note:** Windows users shoud use `downloader.exe` instead of `./downloader`

### Blocklist

Optionally, a blocklist can be provided to exclude certain files from the download.
The blocklist is a simple text file with one computedFileName name per line.
The blocklist can be provided by using the `--blocklist` flag.

```
./downloader --url https://shutterbase.fsg.one --email <email> --password <password> --project qagr042y62aeptz --blocklist my-blocklist.txt --whitelist vbo download delta
```

### Blacklist

Additionally to the whitelist, a `--blacklist` of tags can be specified. The list is logically `OR`-concatenated. More than one tag can be specified by separating them with commas. No spaces must be included.
```
./downloader --url https://shutterbase.fsg.one --email <email> --password <password> --project qagr042y62aeptz --whitelist vbo --blacklist internal,review download delta
```
