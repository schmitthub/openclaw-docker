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
- Generated file existence and non-emptiness (9 artifacts)
- Dockerfile content: base image, version, iptables, iproute2, gosu, pnpm, bun, ENTRYPOINT, no dev tools
- Entrypoint content: default route via Envoy, DOCKER_OUTPUT chain restore, iptables NAT DNAT to Envoy, FILTER OUTPUT DROP, Docker DNS allow, gosu node, executable
- Compose content: services (envoy, openclaw-gateway, openclaw-cli), networks, build directive, cap_add NET_ADMIN, gateway command with --bind lan, CLI entrypoint/stdin_open/tty, init/restart, Envoy static IP (172.28.0.2), IPAM subnet, dns config, depends_on
- Envoy config: ingress listener, egress transparent TLS proxy (tls_inspector, sni_dynamic_forward_proxy, server_names, deny_cluster), DNS listener (dns_filter, Cloudflare 1.1.1.2/1.0.0.2, port 53 UDP), domain whitelist, WebSocket, use_remote_address, xff_num_trusted_hops; no HTTP CONNECT artifacts
- Env file content: expected variables, no proxy env vars (HTTP_PROXY/HTTPS_PROXY/NO_PROXY removed), no NODE_OPTIONS
- Setup script: executable perms, shebang, onboarding flow, config set calls (auth token, trustedProxies, dangerouslyDisableDeviceAuth, controlUi.allowedOrigins), identity dir
- Custom options propagation (ports, bind, apt packages, allowed-domains)
- Idempotency (two runs produce identical output)
