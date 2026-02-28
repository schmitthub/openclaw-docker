---
globs: ["**/*_test.go", "e2e/**/*.go", "internal/testenv/**/*.go", "e2e/harness/**/*.go"]
---

# Go Testing Rules

## Running Tests
- `go test ./...` — all tests
- `go test ./e2e/...` — e2e generation tests only
- `make test` — same as `go test ./...`
- `make check` — test + vet + lint

## Test Infrastructure

### testenv (`internal/testenv/testenv.go`)
Creates isolated temp directories with cleanup:
- `testenv.New(t)` — returns `*Env` with `Dirs.Base` and `Dirs.Cache`
- Sets `OPENCLAW_DOCKER_CACHE_DIR` env var (restored on cleanup)
- Optional: `testenv.WithConfig(yamlString)` for config-backed tests
- Resolves macOS symlinks (`/var` -> `/private/var`) for path consistency

### harness (`e2e/harness/harness.go`)
Wraps testenv for CLI integration tests:
- `h := &harness.Harness{T: t}` — create harness
- `setup := h.NewIsolatedFS()` — get isolated dirs (`setup.BaseDir`, `setup.CacheDir`)
- `result := h.Run("render", "--dangerous-inline", ...)` — execute CLI through Cobra
- `result.Err` / `result.ExitCode` — check outcome

### Test Pattern (e2e/generate_test.go)
```go
func TestExample(t *testing.T) {
    h := &harness.Harness{T: t}
    setup := h.NewIsolatedFS()
    versionsFile := seedManifest(t, setup.CacheDir) // write test manifest
    outputDir := filepath.Join(setup.BaseDir, "deploy")
    result := h.Run("render", "--dangerous-inline",
        "--versions-file", versionsFile,
        "--output", outputDir,
    )
    if result.Err != nil {
        t.Fatalf("render failed: %v", result.Err)
    }
    // assert on generated files in outputDir
}
```

### seedManifest helper
Writes a test `versions.json` to the harness cache dir so `render` can find it without npm:
```go
versionsFile := seedManifest(t, setup.CacheDir)
// writes to: <cacheDir>/openclaw-docker/versions.json
```

### Key test flags
- Always use `--dangerous-inline` in tests (skip write prompts)
- Use `--versions-file` to point to seeded manifest
- Tests that need npm (e.g., `TestGenerateFullPipeline`) should `t.Skip()` if `npm` not in PATH

## What to Test
- Generated file existence and non-emptiness
- Dockerfile content: base image, version, no firewall/entrypoint/dev tools
- Compose content: services, networks, build directive (not image tag)
- Env file content: expected variables, proxy config
- Setup script: executable perms, shebang, compose/token logic
- Custom options propagation (ports, bind, apt packages)
- Idempotency (two runs produce identical output)
