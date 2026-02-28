# Package: `e2e/harness`

Wraps `internal/testenv` for CLI integration testing.

## Types

- `Harness{T}` — test harness, holds `*testing.T`
- `SetupResult{BaseDir, CacheDir}` — resolved paths from `NewIsolatedFS()`
- `RunResult{ExitCode, Err}` — CLI execution outcome

## Usage

```go
h := &harness.Harness{T: t}
setup := h.NewIsolatedFS()      // creates isolated temp dirs
result := h.Run("render", "--dangerous-inline", "--versions-file", vf, "--output", dir)
if result.Err != nil {
    t.Fatalf("failed: %v", result.Err)
}
```

## How Run Works

1. Creates a fresh `cmd.NewRootCmd("test", "test")`
2. Sets args via `rootCmd.SetArgs(args)`
3. Calls `rootCmd.Execute()`
4. Returns exit code (0 or 1) and error

## Key Details

- Each `Run()` call creates a new root command, so tests are isolated from each other's flag state
- The harness does NOT capture stdout/stderr — tests assert on generated files, not CLI output
- `NewIsolatedFS()` delegates to `testenv.New(t)` which sets `OPENCLAW_DOCKER_CACHE_DIR`
