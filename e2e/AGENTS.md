# Directory: `e2e/`

End-to-end generation tests. Package name: `test`.

## Files

| File | Purpose |
|------|---------|
| `generate_test.go` | 14 tests covering render and full generate pipeline |
| `harness/harness.go` | Test harness: isolated FS + Cobra CLI execution |

## Running

```bash
go test ./e2e/...          # all e2e tests
go test ./e2e/... -v       # verbose
go test ./e2e/... -run TestRenderDockerfileContent  # single test
```

## Test Manifest Seeding

Tests that use `render` (not `generate`) seed a manifest via `seedManifest(t, cacheDir)`:
```go
versionsFile := seedManifest(t, setup.CacheDir)
// Writes test JSON to <cacheDir>/openclaw-docker/versions.json
```

## Tests That Need npm

`TestGenerateFullPipeline` skips if `npm` is not in PATH (`exec.LookPath("npm")`).
All other tests use seeded manifests and don't require network access.

## Current Test Coverage

- File existence and non-emptiness (9 artifacts including CA cert/key)
- Dockerfile content (base image, version, forbidden content)
- Custom apt packages
- Compose services/networks/build directive, squid mounts, NODE_EXTRA_CA_CERTS, named volumes
- Env file variables and proxy config
- Setup script permissions, shebang, expected content
- Custom options propagation (port, bind)
- Idempotency (two runs = identical output)
- Full pipeline (npm resolve + render)
- Squid.conf content (SSL bump, sslcrtd_program, deny all, openclaw.ai)
- Squid allowed domains propagation via --squid-allowed-domains
- openclaw.json content (gateway, mode, bind, auth, token placeholder)
- CA cert generation (valid PEM, idempotent across re-runs)

## Test Pattern

```go
func TestExample(t *testing.T) {
    h := &harness.Harness{T: t}
    setup := h.NewIsolatedFS()
    versionsFile := seedManifest(t, setup.CacheDir)
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
