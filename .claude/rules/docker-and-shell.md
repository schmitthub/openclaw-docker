---
globs: ["**/Dockerfile*", "**/compose*.yaml", "**/.env*", "**/*.sh", "internal/render/**/*.go"]
---

# Docker & Shell Artifact Rules

## Generated Artifacts
The CLI generates files to `<output>/` (default `./openclaw-deploy`), organized by service:

**Root files:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `compose.yaml` | 0644 | Envoy + gateway services on internal network |
| `.env.openclaw` | 0644 | Runtime env vars including proxy config |
| `setup.sh` | 0755 | Token gen, onboarding, config management, compose up |
| `manifest.json` | 0644 | Resolved version metadata |

**`compose/openclaw/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `Dockerfile` | 0644 | `node:22-bookworm` with `iptables` + `gosu` for egress lockdown |
| `entrypoint.sh` | 0755 | Root-owned: sets iptables rules, then drops to `node` via `gosu` |

**`compose/envoy/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `envoy.yaml` | 0644 | Envoy config: ingress (TLS+reverse proxy) + egress (CONNECT+domain ACL) |
| `server-cert.pem` | 0644 | Self-signed TLS cert for Envoy ingress listener |
| `server-key.pem` | 0600 | TLS key for Envoy ingress listener |

## Dockerfile Conventions
- Base: `node:22-bookworm` (matches official OpenClaw Docker pattern)
- Always installs `iptables` and `gosu` (required for egress security model)
- OpenClaw installed via `npm install -g openclaw@<version>` (not the install script — no need for interactive/retry logic in Docker)
- `SHARP_IGNORE_GLOBAL_LIBVIPS=1` set during npm install (matches install script behavior)
- CLI symlink: `ln -sf "$(npm root -g)/openclaw/dist/entry.js" /usr/local/bin/openclaw`
- Optional `OPENCLAW_INSTALL_BROWSER` ARG: bakes Playwright + Chromium + Xvfb (~300MB)
- `COPY entrypoint.sh /usr/local/bin/entrypoint.sh` — root-owned, 0755
- `ENTRYPOINT ["entrypoint.sh"]` runs as root to set iptables, then drops to `node`
- `CMD ["openclaw", "gateway", "--allow-unconfigured"]`
- `OPENCLAW_DOCKER_APT_PACKAGES` ARG for optional additional packages
- No dev tools (zsh, git-delta, hadolint, fzf)

## Entrypoint Security Model
The `entrypoint.sh` script enforces network-level egress isolation:
1. Resolves Envoy's IP via `getent hosts envoy`
2. Sets `iptables -P OUTPUT DROP` (default deny all outbound)
3. Allows: loopback, Docker DNS (127.0.0.11:53 UDP), established/related, traffic to Envoy IP
4. Drops to `node` user via `exec gosu node "$@"`

The `node` user cannot modify iptables rules (requires `CAP_NET_ADMIN` which only root has).
This is the primary egress enforcement — `HTTP_PROXY` env vars are convenience, not security.

## Compose Conventions
- Envoy (`envoyproxy/envoy:v1.33-latest`) is the sole ingress/egress proxy
- Envoy publishes port 443 for ingress; egress listener on port 10000 (internal only)
- Gateway builds from `compose/openclaw/` subdirectory
- Gateway uses `cap_add: [NET_ADMIN]` — required by root entrypoint for iptables setup
- Gateway has no published ports — only accessible via Envoy on internal network
- Envoy on both `openclaw-internal` (internal: true) and `openclaw-egress` networks
- Gateway on `openclaw-internal` only — all egress routes through Envoy
- `HTTPS_PROXY=http://envoy:10000` set as convenience for proxy-aware tools
- Env vars sourced from `.env.openclaw` via `env_file`

## Envoy Configuration
- **Ingress listener (:443)**: TLS termination, WebSocket upgrade, reverse proxy to gateway
- **Egress listener (:10000)**: HTTP CONNECT forward proxy with domain whitelist
- Domain ACL via virtual host `domains` field matching `host:port` from CONNECT authority
- `openclaw.ai` always included in egress whitelist
- Dynamic forward proxy cluster resolves allowed domains and establishes TCP tunnels
- Unmatched domains get 403 Forbidden response
- No SSL bump / MITM — TLS is end-to-end between gateway and API providers

## Shell Script Conventions
- Shebang: `#!/usr/bin/env bash`
- `set -euo pipefail`
- Bash 3.2 compatible (macOS default)
- Token gen: `openssl rand -hex 32` with python3 fallback
- Uses `docker compose -f "$COMPOSE_FILE" build` (not standalone docker build)

## Generation Code
Generation lives in `internal/render/render.go` and `internal/render/ca.go`:
- `Generate(opts)` — orchestrates all artifact writes
- `dockerfileFor(opts)` — Dockerfile content via `fmt.Sprintf`
- `entrypointContent()` — entrypoint.sh with iptables + gosu
- `composeFileContent(opts)` — compose YAML as string-joined lines
- `openClawEnvFileContent(opts)` — env file via `fmt.Sprintf`
- `setupScriptContent(opts)` — setup.sh via `fmt.Sprintf`
- `envoyConfigContent(opts)` — envoy.yaml with ingress + egress listeners
- `generateTLSCert(opts)` — self-signed TLS cert generation (in `ca.go`, preserves existing across re-runs)

## Validation
```bash
# Generate artifacts
go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline

# Validate compose
docker compose -f ./openclaw-deploy/compose.yaml config

# Check setup.sh and entrypoint.sh are executable
test -x ./openclaw-deploy/setup.sh
test -x ./openclaw-deploy/compose/openclaw/entrypoint.sh
```
