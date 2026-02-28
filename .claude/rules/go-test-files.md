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
- `result := h.Run("generate", "--dangerous-inline", ...)` — execute CLI through Cobra
- `result.Err` / `result.ExitCode` — check outcome

### Test Pattern (e2e/generate_test.go)
```go
func TestExample(t *testing.T) {
    h := &harness.Harness{T: t}
    setup := h.NewIsolatedFS()
    seedManifest(t, setup.BaseDir) // write test manifest + set env var
    outputDir := filepath.Join(setup.BaseDir, "deploy")
    result := h.Run("generate", "--dangerous-inline",
        "--output", outputDir,
    )
    if result.Err != nil {
        t.Fatalf("generate failed: %v", result.Err)
    }
    // assert on generated files in outputDir
}
```

### seedManifest helper
Writes a test `manifest.json` and sets `OPENCLAW_DOCKER_VERSIONS_FILE` env var
so `generate` reads it instead of resolving from npm:
```go
seedManifest(t, setup.BaseDir)
// writes manifest.json to baseDir, sets env var
```

### Key test flags
- Always use `--dangerous-inline` in tests (skip write prompts)
- Tests that need npm (e.g., `TestGenerateFullPipeline`) should `t.Skip()` if `npm` not in PATH

## What to Test
- Generated file existence and non-emptiness
- Dockerfile content: base image, version, iptables, gosu, ENTRYPOINT, no dev tools
- Entrypoint content: iptables OUTPUT DROP, Docker DNS allow, Envoy allow, gosu node, executable
- Compose content: services (envoy, gateway), networks, build directive, cap_add NET_ADMIN
- Envoy config: ingress/egress listeners, domain whitelist, CONNECT, WebSocket
- Env file content: expected variables, proxy config (envoy:10000), no NODE_OPTIONS
- Setup script: executable perms, shebang, compose/token logic
- Custom options propagation (ports, bind, apt packages)
- Idempotency (two runs produce identical output)
