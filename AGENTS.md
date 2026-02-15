# AGENTS.md

This file provides guidance for AI agents working with this Go codebase.

## Build, Lint & Test Commands

```bash
# Build and run locally (uses v4l2 ffmpeg)
make run                     # Builds and starts the server immediately

# Build for local development without running (uses v4l2 ffmpeg)
make dev                     # Creates scripts/stream.sh and petcam binary
./petcam                     # Run the server manually (can run multiple times)

# Clean up local build artifacts
make clean                   # Removes petcam and scripts/stream.sh

# Build for ARM Linux (Raspberry Pi deployment) - uses rpicam-vid
make                         # Builds ARM binary, deploys to rpi:~/petcam/, cleans up locally

# Deploy to a different remote host or user
make REMOTE_HOST=pi REMOTE_USER=pi    # Deploy to pi@pi instead of default mk@rpi
# Or set environment variables: REMOTE_HOST=pi REMOTE_USER=pi make

# Run all tests with race detection
make test                    # Equivalent to: go test -race ./...

# Run a single test
go test -race -run TestSingleUserFlow ./streamer/

# Run all tests in a specific package
go test -race ./streamer/

# Standard Go build for current platform
go build

# Format code
go fmt ./...

# Vet code for issues
go vet ./...
```

## Project Structure

- `main.go` - HTTP server entry point, serves HLS playlist and segments
- `streamer/` - Package managing FFmpeg process lifecycle
  - `streamer.go` - Core streaming logic with goroutine management
  - `process.go` - FFmpeg process wrapper
  - `settings.go` - Configuration struct
  - `*_test.go` - Test files
- `scripts/` - Shell scripts for streaming commands
  - `stream_pi.sh` - Raspberry Pi camera pipeline (rpicam-vid + ffmpeg)
  - `stream_dev.sh` - Development stream (v4l2 + ffmpeg)
  - `stream.sh` - Generated script (not in repo, created by Makefile)
- `Makefile` - Build automation for ARM cross-compilation
- `go.mod` - Go 1.25.6, no external dependencies

## Code Style Guidelines

### Formatting

- Use `go fmt` for all Go files (enforced)
- Max line length: aim for ~100 characters where practical
- Use tabs for indentation (Go standard)

### Imports

- Group imports: standard library first, then third-party, then local
- Use blank line between import groups
- Example:
  ```go
  import (
      "fmt"
      "log"
      "net/http"

      "github.com/some/package"

      "github.com/mateuszkowalczyk/petcam/streamer"
  )
  ```

### Naming Conventions

- **Exported**: PascalCase (e.g., `NewStreamer`, `EnsureStreaming`)
- **Unexported**: camelCase (e.g., `streamLoop`, `keepAlive`)
- **Interfaces**: -er suffix (e.g., `io.Reader`)
- **Constants**: ALL_CAPS for exported, camelCase for unexported
- **Test functions**: `Test` + descriptive name (e.g., `TestSingleUserFlow`)
- **Test helpers**: descriptive names with `_test.go` suffix

### Types & Structs

- Use named structs with descriptive field names
- Document exported types and functions with comments
- Use unexported fields with getters/setters when encapsulation needed
- Example:
  ```go
  type Streamer struct {
      settings Settings
      process  *process
      // ...
  }
  ```

### Error Handling

- Return errors as values, check with `if err != nil`
- Fatal errors in `main()`: `log.Fatalf("context: %v\n", err)`
- Store errors in struct fields for async operations (see `Streamer.err`)
- Always include context in error messages

### Concurrency Patterns

- Use `sync.WaitGroup` for coordinating goroutines
- Use `sync.Once` for one-time initialization/cleanup
- Channel naming: use descriptive names (e.g., `keepAlive`, `stopped`)
- Buffered channels for non-blocking signal sends (size 1)
- Always clean up goroutines on shutdown

### Comments

- Package comment at top of file explaining purpose
- Exported functions/types must have doc comments
- Use `// TODO: ` for incomplete items
- Example: `// EnsureStreaming ensures the stream is active...`

### Testing

- Use standard `testing` package
- Test function format: `TestXxx(t *testing.T)`
- Use `t.TempDir()` for temporary files (auto-cleanup)
- Test behavior, not implementation details
- Use race detector: always run with `-race` flag
- Helper functions in `*_test.go` files, marked with `t.Helper()`

### Architecture Patterns

- Clear separation between packages (`main` vs `streamer`)
- Constructor function pattern: `NewXxx(settings Settings) *Xxx`
- Lifecycle methods: `Start()`, `Stop()`, `Wait()`
- Configuration via struct (see `Settings`)
- No global state; pass dependencies explicitly

## Environment

- Go version: 1.25.6
- Target platform: ARM Linux (Raspberry Pi)
- No external dependencies (stdlib only)
- FFmpeg required at runtime

## Pre-commit Checklist

Before committing changes:

1. Run `go fmt ./...` - ensure formatting
2. Run `go vet ./...` - check for issues
3. Run `go test -race ./...` - all tests pass
4. Run specific tests for modified packages
5. Verify build: `go build`
6. Ensure `scripts/stream_dev.sh` and `scripts/stream_pi.sh` are executable: `chmod +x scripts/stream_dev.sh scripts/stream_pi.sh`
