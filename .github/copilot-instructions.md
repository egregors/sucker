# Copilot Instructions for Sucker

## Project Overview

Sucker is a file downloader written in Go that extracts and downloads files from HTML pages piped through stdin. It uses concurrent workers to download files efficiently with progress bars.

## Technology Stack

- **Language**: Go 1.24
- **Key Dependencies**:
  - `github.com/vbauerster/mpb/v7` - Progress bar library
  - `golang.org/x/net/html` - HTML parsing
- **Dependency Management**: Go modules with vendored dependencies

## Build and Test Commands

- **Build**: `make build` - Builds the binary using vendored modules with CGO disabled
- **Test**: `make test` - Runs tests with race detector and coverage
- **Benchmark**: `make bench` - Runs benchmark tests
- **Clean**: `make clean` - Removes downloaded files directory

## Project Structure

- `main.go` - Single file containing all application logic
- `vendor/` - Vendored dependencies
- `Makefile` - Build and test commands
- `.github/workflows/go.yml` - CI/CD pipeline for build and test

## Coding Conventions

- **Error Handling**: Use wrapped errors with `fmt.Errorf` and `%w` verb
- **File Permissions**: Use octal notation (e.g., `0o600`, `os.ModePerm`)
- **Concurrency**: Uses goroutines with `sync.WaitGroup` for worker pool pattern
- **Context**: Pass `context.Context` for cancellation support
- **Logging**: Use standard `log` package with prefixes like `[ERROR]`

## Key Features

1. **HTML Parsing**: Reads HTML from stdin and extracts links with specific file extensions
2. **Concurrent Downloads**: Uses 5 worker goroutines for parallel downloads
3. **Progress Tracking**: Multi-bar progress display showing individual file downloads and total progress
4. **History**: Tracks downloaded files in `history.gob` to avoid re-downloading
5. **File Types**: Currently supports `webm` and `mp4` file extensions

## Important Implementation Details

- Downloads are saved to `sucker_downloads/` directory in the current working directory
- Uses `http.Get` for downloading files
- Progress bars are configured with custom styling using Unicode characters
- History is stored using Go's `encoding/gob` for serialization
- File existence is checked before downloading to avoid duplicates

## Development Workflow

1. Make code changes in `main.go`
2. Run `make build` to compile
3. Run `make test` to verify tests pass
4. Use `make clean` to remove downloaded files during testing

## CI/CD

GitHub Actions workflow runs on every push:
- Sets up Go 1.24
- Vendors dependencies
- Builds the project
- Runs tests

## Notes for AI Assistants

- This is a single-file Go application - all logic is in `main.go`
- Dependencies are vendored - use `go mod vendor` after adding new dependencies
- The application expects HTML input from stdin (e.g., piped from clipboard)
- Focus on maintaining the concurrent worker pattern when making changes
- Preserve the progress bar functionality and user experience
