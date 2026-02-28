# Directory: `e2e/`

End-to-end generation tests. Package name: `test`.

## Files

| File | Purpose |
|------|---------|
| `generate_test.go` | 14 tests covering generate pipeline |
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

- File existence and non-emptiness (9 artifacts in compose/<service>/ subdirs + root)
- Output directory structure (compose/openclaw, compose/envoy; no squid/nginx)
- Dockerfile content (base image, version, iptables, iproute2, gosu, pnpm, bun, ENTRYPOINT, no proxy-preload)
- Custom apt packages
- Compose services (envoy, openclaw-gateway, openclaw-cli), networks, build contexts, Envoy volumes, cap_add NET_ADMIN
- Compose gateway: command with --bind lan, init, restart, HOME/TERM env vars, dns: 172.28.0.2
- Compose CLI: entrypoint ["openclaw"], stdin_open, tty, BROWSER echo, depends_on envoy, dns: 172.28.0.2
- Compose networking: Envoy static IP (172.28.0.2), IPAM subnet (172.28.0.0/24)
- Env file variables (no proxy env vars — iptables DNAT handles egress, no NODE_OPTIONS, no dead vars)
- Setup script permissions, shebang, onboarding flow, config set calls (auth token, trustedProxies, controlUi.allowedOrigins, dangerouslyDisableDeviceAuth), identity dir
- Custom options propagation (port, bind, allowed-domains)
- Idempotency (two runs = identical output)
- Full pipeline (npm resolve + generate)
- Envoy config content (ingress listener, egress transparent TLS proxy with SNI filter chains, TLS Inspector, domain whitelist, deny_cluster, WebSocket, XFF forwarding, DNS listener with Cloudflare resolvers; no HTTP CONNECT artifacts)
- Envoy allowed domains propagation via --allowed-domains (additive to all hardcoded domains)
- TLS cert generation (valid PEM, idempotent across re-runs)
- Entrypoint content (default route via Envoy, DOCKER_OUTPUT chain restore, iptables NAT DNAT to Envoy, FILTER OUTPUT DROP, Docker DNS, gosu node, executable)

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
