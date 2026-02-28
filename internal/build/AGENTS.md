# Package: `internal/build`

Build metadata injected via ldflags at compile time.

## Variables

- `Version` — set via `-X .../build.Version=$(VERSION)`, defaults to `"DEV"`
- `Date` — set via `-X .../build.Date=$(DATE)`, empty for dev builds

## Fallback Chain (when Version is "DEV")

1. `debug.ReadBuildInfo()` main module version
2. `git describe --tags --always --dirty`
3. `git branch --show-current`
4. `"dev"`

## Build Command

```bash
make build  # uses ldflags from Makefile
# or:
go build -ldflags "-s -w -X .../build.Version=v1.0.0 -X .../build.Date=2026-02-27T00:00:00Z" .
```
