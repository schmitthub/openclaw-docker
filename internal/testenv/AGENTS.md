# Package: `internal/testenv`

Isolated filesystem environments for tests. Creates temp dirs and sets env vars with automatic cleanup.

## Usage

```go
env := testenv.New(t)
env.Dirs.Base   // temp root (parent of all dirs)
env.Dirs.Cache  // cache dir (set as OPENCLAW_DOCKER_CACHE_DIR)

// With config:
env := testenv.New(t, testenv.WithConfig(yamlString))
env.Config // *config.FileConfig
```

## What It Does

1. Creates temp directory (resolves macOS symlinks: `/var` -> `/private/var`)
2. Creates `cache/` subdirectory under temp root
3. Sets `OPENCLAW_DOCKER_CACHE_DIR` env var (restored via `t.Setenv`)
4. Optionally parses a YAML config string

## Key Detail

`OPENCLAW_DOCKER_CACHE_DIR` is checked first in `defaultVersionsFilePath()` (in `internal/cmd/root.go`),
so tests using testenv automatically get isolated manifest paths without needing `--versions-file`.

## Types

- `IsolatedDirs{Base, Cache, Deploy}` — resolved temp paths
- `Env{Dirs, Config}` — full test environment
- `Option func(t, e)` — functional options (e.g. `WithConfig`)
