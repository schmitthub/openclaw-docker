# Directory: `e2e/`

End-to-end generation tests. Package name: `test`.

## Files

| File | Purpose |
|------|---------|
| `generate_test.go` | 16 tests covering generate pipeline |
| `harness/harness.go` | Test harness: isolated FS + Cobra CLI execution |

## Running

```bash
go test ./e2e/...          # all e2e tests
go test ./e2e/... -v       # verbose
go test ./e2e/... -run TestGenerateDockerfileContent  # single test
```

## Test Manifest Seeding

Tests seed a manifest via `seedManifest(t, baseDir)` which writes `manifest.json`
and sets `OPENCLAW_DOCKER_VERSIONS_FILE` env var to bypass npm resolution:
```go
seedManifest(t, setup.BaseDir)
// Writes manifest.json to baseDir, sets env var
```

## Tests That Need npm

`TestGenerateFullPipeline` skips if `npm` is not in PATH (`exec.LookPath("npm")`).
All other tests use seeded manifests and don't require network access.

## Current Test Coverage

- File existence and non-emptiness (13 artifacts in compose/<service>/ subdirs + root)
- Output directory structure (compose/openclaw, compose/squid, compose/nginx)
- Dockerfile content (base image, version, forbidden content)
- Custom apt packages
- Compose services (nginx, squid, gateway), networks, build contexts, volume mounts, NODE_EXTRA_CA_CERTS, named volumes
- Env file variables and proxy config
- Setup script permissions, shebang, expected content
- Custom options propagation (port, bind)
- Idempotency (two runs = identical output)
- Full pipeline (npm resolve + generate)
- Squid.conf content (SSL bump, sslcrtd_program, deny all, openclaw.ai)
- Squid allowed domains propagation via --squid-allowed-domains
- openclaw.json content (gateway, mode, bind, auth, token placeholder)
- CA cert generation (valid PEM, idempotent across re-runs)
- nginx.conf content (upstream, proxy_pass, SSL, WebSocket upgrade, commented mTLS)
- nginx cert generation (valid PEM, signed by CA, idempotent across re-runs)

## Test Pattern

```go
func TestExample(t *testing.T) {
    h := &harness.Harness{T: t}
    setup := h.NewIsolatedFS()
    seedManifest(t, setup.BaseDir)
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
