# CRTDumper

CRTDumper is a Go application that massively scans Certificate Transparency (CT) logs to extract and save domain names datasets for later processing. It supports resuming from the last fetched index to avoid redundant processing in case of interruptions.

## Features

- Fetches certificates from CT logs.
- Extracts domain names from certificates.
- Saves progress to a file for resuming.
- Supports multithreading for faster processing.
- Handles retries for transient network errors.

## Installation

```go
go install github.com/Kagee/crtdumper@HEAD 
```

## TODOs
* Actually implement increasing sleep time on HTTP 429 Too Many Requests, Retry-After
* Option for limiting the HTTP client to IPv4/IPv6 (some logs are dead/slow on IPv6?)

## Usage

CRTDumper has two primary modes, dumping all dmains from one to multiple logs, and
a mode for listing avaliable operators, logs or the domains extracted from a spesific
log entry (most useful for debugging)

### Examples

#### List all operators
```sh
./crtdumper -list
```

#### List all logs from Let's Encrypt (case insensitive)
```sh
./crtdumper -list -operator "lEt'S ENCrypt"
```

#### List the domains in www.cia.gov's certificate for 2024-2025
```sh
./crtdumper -list -operator "Cloudflare" -log https://ct.cloudflare.com/logs/nimbus2025 --entry 38779142
```

#### Write all domains from all active operator logs to ./log-output.log
```sh
./crtdumper
```

#### Write all domains from all active operator logs to stdout, limiting to the .no TLD, not resuming from previous runs
```sh
./crtdumper --output - --required-postfix ".no" -no-resume
```

### Command-line Flags
#### For scraping logs
- `-output <filename or ->`:
        Output filename. Use - for stdout. (default ./output.log) (default "./output.log")
- `-num-entries <number>`:
        number of entries to query at once (default 20)
- `-no-resume`:
        Do not use resume file (default false, i.e. use resume file)
- `-resume-file string`:
        Resume filename (default ./log-resume.json)

#### Limiting output
- `-require-postfix <string>`:
        Postfixe to require. (will not output domains not matching postfix)
- `-only-log <string>`:
        URL of the single log to process
- `-only-operator <string>`:
        Name of the only operator to process
- `-skip-log <value>`:
        URLs of logs to skip (can be repeated)
- `-skip-operator <value>`:
        URLs of operators to skip (can be repeated)

#### For listings
- `-list`:
        List something defined by operator/log/entry (default lists all operators)
- `-operator string`:
        Operator for listing (if only this, will lists logs for this operator)
- `-log string`:
        Log for listing (does not make sense without --entry)
- `-entry uint`:
        Entry to list from --operator's --log

#### Common options
- `-loglevel string`:
        Application loglevel (INFO (default), DEBUG, ERROR, WARN) (default "INFO")
- `-include-precert`:
        Include data from precertificates in output

## Handling Interruptions

CRTDumper is designed to handle interruptions gracefully. If interrupted (e.g., by
pressing `Ctrl+C`), it will save its state to `log-resume.json` and print a message. 
Unless you specify `-no-resume`, it will be used automatically.

## Issues and Contributions

Feel free to submit issues or pull requests if you find any bugs or want to contribute.

## Acknowledgements

The base for this was project taken from github.com/c3l3si4n/crtdumper, and heavily modified.

This project depends on several open-source packages:

- `github.com/alphadose/haxmap`
- `github.com/go-resty/resty/v2`
- `github.com/google/certificate-transparency-go`
- `github.com/hashicorp/go-retryablehttp`

