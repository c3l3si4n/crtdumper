# CRTDumper

CRTDumper is a Go application that massively scans Certificate Transparency (CT) logs to extract and save domain names datasets for later processing. It supports resuming from the last fetched index to avoid redundant processing in case of interruptions.

## Features

- Fetches certificates from CT logs.
- Extracts domain names from certificates.
- Saves progress to a cache file for resuming.
- Supports multithreading for faster processing.
- Handles retries for transient network errors.

## Installation

```go
go install github.com/c3l3si4n/crtdumper@HEAD
```

## Usage

To run the CRTDumper, you can use the following command:

```sh
crtdumper -o <output_filename> [-r] [-n <number_of_records_to_query>]
```

### Command-line Flags

- `-o <output_filename>`: (Required) The file where the parsed domain names will be saved.
- `-r`: (Optional) Resume from the last execution by loading the cache from `cache.json`.
- `-n <number_of_records_to_query>`: (Optional) The number of records to query at once. Default is 20.

### Example:

```sh
./crtdumper -o domains.txt -r -n 50
```


## Handling Interruptions

CRTDumper is designed to handle interruptions gracefully. If interrupted (e.g., by pressing `Ctrl+C`), it will save its state to `cache.json` and print a message. You can resume execution later using the `-r` flag.

## Issues and Contributions

Feel free to submit issues or pull requests if you find any bugs or want to contribute.

## Acknowledgements

This project depends on several open-source packages:

- `github.com/alphadose/haxmap`
- `github.com/go-resty/resty/v2`
- `github.com/google/certificate-transparency-go`
- `github.com/hashicorp/go-retryablehttp`

