# Directory: `e2e/`

End-to-end generation tests. Package name: `test`.

## Files

| File | Purpose |
|------|---------|
| `generate_test.go` | 15 tests covering generate pipeline |
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

- File existence and non-emptiness (10 artifacts in compose/<service>/ subdirs + root)
- Output directory structure (compose/openclaw, compose/envoy; no squid/nginx)
- Dockerfile content (base image, version, iptables, gosu, ENTRYPOINT, no proxy-preload)
- Custom apt packages
- Compose services (envoy, gateway), networks, build contexts, Envoy volumes, cap_add NET_ADMIN
- Env file variables and proxy config (envoy:10000, no NODE_OPTIONS, no dead vars)
- Setup script permissions, shebang, expected content, envoy service start
- Custom options propagation (port, bind)
- Idempotency (two runs = identical output)
- Full pipeline (npm resolve + generate)
- Envoy config content (ingress/egress listeners, domain whitelist, CONNECT, WebSocket)
- Envoy allowed domains propagation via --allowed-domains
- openclaw.json content (gateway, mode, bind, auth, token placeholder)
- TLS cert generation (valid PEM, idempotent across re-runs)
- Entrypoint content (iptables OUTPUT DROP, Docker DNS, Envoy allow, gosu node, executable)

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
