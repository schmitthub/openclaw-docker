---
globs: ["**/Dockerfile*", "**/compose*.yaml", "**/.env*", "**/*.sh", "internal/render/**/*.go"]
---

# Docker & Shell Artifact Rules

## Generated Artifacts
The CLI generates files to `<output>/` (default `./openclaw-deploy`), organized by service:

**Root files:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `compose.yaml` | 0644 | 3 services: envoy, openclaw-gateway, openclaw-cli |
| `.env.openclaw` | 0644 | Runtime env vars (token, ports, bind settings) |
| `setup.sh` | 0755 | Build, onboard, configure (via CLI), compose up |
| `manifest.json` | 0644 | Resolved version metadata |

**`compose/openclaw/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `Dockerfile` | 0644 | `node:22-bookworm` + `iptables` + `iproute2` + `gosu` + `pnpm` + `bun` |
| `entrypoint.sh` | 0755 | Root-owned: sets iptables rules, then drops to `node` via `gosu` |

**`compose/envoy/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `envoy.yaml` | 0644 | Envoy config: ingress (TLS+reverse proxy+XFF) + egress (transparent TLS proxy+SNI whitelist) |
| `server-cert.pem` | 0644 | Self-signed TLS cert for Envoy ingress listener |
| `server-key.pem` | 0600 | TLS key for Envoy ingress listener |

## Dockerfile Conventions
- Base: `node:22-bookworm` (matches official OpenClaw Docker pattern)
- Always installs `iptables`, `iproute2`, and `gosu` (required for egress security model)
- `pnpm` installed via `corepack enable && corepack prepare pnpm@latest --activate`
- `bun` installed via `BUN_INSTALL=/usr/local curl -fsSL https://bun.sh/install | bash` (global, accessible to node user)
- OpenClaw installed via `npm install -g openclaw@<version>`
- `SHARP_IGNORE_GLOBAL_LIBVIPS=1` set during npm install
- CLI symlink: `ln -sf "$(npm root -g)/openclaw/dist/entry.js" /usr/local/bin/openclaw`
- Optional `OPENCLAW_INSTALL_BROWSER` ARG: bakes Playwright + Chromium + Xvfb (~300MB)
- `COPY entrypoint.sh /usr/local/bin/entrypoint.sh` — root-owned, 0755
- `ENTRYPOINT ["entrypoint.sh"]` runs as root to set iptables, then drops to `node`
- `CMD ["openclaw", "gateway", "--allow-unconfigured"]` (overridden by compose `command`)
- `OPENCLAW_DOCKER_APT_PACKAGES` ARG for optional additional packages
- No dev tools (zsh, git-delta, hadolint, fzf)

## Entrypoint Security Model
The `entrypoint.sh` script enforces transparent egress isolation via iptables:
1. Resolves Envoy's IP via `getent hosts envoy`
2. Adds default route via Envoy (`ip route add default via $ENVOY_IP`) — required because `internal: true` networks have no gateway, so the kernel rejects connections to external IPs with "Network is unreachable" before iptables can DNAT them
3. Flushes existing rules, then restores Docker's `DOCKER_OUTPUT` chain jump (Docker DNS uses this chain to DNAT port 53 to a high port — without it, all DNS fails)
4. NAT table: DNAT all outbound TCP to Envoy's transparent proxy listener (skip Envoy's own IP to avoid loop)
5. FILTER table: `OUTPUT DROP` default policy. Allows: loopback, Docker DNS (127.0.0.11:53 UDP), established/related, Envoy IP
6. Drops to `node` user via `exec gosu node "$@"`

Apps are unaware of the proxy — they connect normally and iptables rewrites the destination.
The `node` user cannot modify iptables rules (requires `CAP_NET_ADMIN` which only root has).
No `HTTP_PROXY`/`HTTPS_PROXY` env vars are needed — iptables DNAT handles all routing transparently.

## Compose Conventions
- **3 services:** `envoy`, `openclaw-gateway`, `openclaw-cli`
- Envoy (`envoyproxy/envoy:v1.33-latest`) is the sole ingress/egress proxy and DNS forwarder
- Envoy has static IP `172.28.0.2` on `openclaw-internal` (IPAM subnet `172.28.0.0/24`)
- Envoy runs as `user: "0"` (required for binding port 53)
- Envoy publishes port 443 for ingress; egress listener on port 10000, DNS listener on port 53 UDP (internal only)
- Gateway and CLI services use `dns: [172.28.0.2]` so Docker DNS forwards external queries to Envoy
- Gateway builds from `compose/openclaw/` subdirectory
- Gateway has explicit `command: ["openclaw", "gateway", "--bind", "lan", "--port", "18789"]` (ensures LAN binding for Docker network reachability)
- Gateway uses `cap_add: [NET_ADMIN]` — required by root entrypoint for iptables + routing setup
- Gateway has `init: true`, `restart: unless-stopped`, `HOME`/`TERM` env vars
- Gateway has no published ports — only accessible via Envoy on internal network
- CLI service shares the same image but overrides `entrypoint: ["openclaw"]` for direct CLI access
- CLI service has `stdin_open: true`, `tty: true`, `init: true`, `BROWSER: echo`
- CLI service has `depends_on: [envoy]` (prevents static IP conflicts during startup)
- CLI service has no restart policy (run-and-exit)
- Envoy on both `openclaw-internal` (internal: true) and `openclaw-egress` networks
- Gateway and CLI on `openclaw-internal` only — all egress routes through Envoy
- No proxy env vars — iptables DNAT provides transparent egress routing
- Env vars sourced from `.env.openclaw` via `env_file`

## Envoy Configuration
- **Ingress listener (:443)**: TLS termination, `use_remote_address: true`, `xff_num_trusted_hops: 0`, WebSocket upgrade, reverse proxy to gateway
- **Egress listener (:10000)**: Transparent TLS proxy with SNI-based domain whitelist. All outbound TCP from gateway is DNAT'd here by iptables.
- **DNS listener (:53 UDP)**: Forwards DNS queries to Cloudflare malware-blocking resolvers (1.1.1.2 / 1.0.0.2). Uses `envoy.filters.udp.dns_filter` with c-ares resolver. Required because Docker's embedded DNS cannot forward external queries on `internal: true` networks.
- TLS Inspector listener filter reads SNI from ClientHello without terminating TLS (no MITM)
- Domain ACL via `filter_chain_match.server_names` matching TLS SNI
- Non-TLS traffic (SSH, plain HTTP, raw TCP) is categorically denied — no SNI to inspect
- **Hardcoded always-allowed (infrastructure):** `clawhub.com`, `registry.npmjs.org`
- **Hardcoded always-allowed (AI providers):** `api.anthropic.com`, `api.openai.com`, `generativelanguage.googleapis.com`, `openrouter.ai`, `api.x.ai`
- `--allowed-domains` is additive to all hardcoded domains
- `sni_dynamic_forward_proxy` resolves whitelisted domains via DNS and forwards to port 443
- `deny_cluster` (STATIC, no endpoints) immediately resets non-whitelisted connections
- SNI spoofing is useless: Envoy resolves the domain independently, connecting to the real IP

## Shell Script Conventions
- Shebang: `#!/usr/bin/env bash`
- `set -euo pipefail`
- Bash 3.2 compatible (macOS default)
- Token gen: `openssl rand -hex 32` with python3 fallback
- Token reuse: `read_config_gateway_token()` reads from existing `openclaw.json`
- Config management: all gateway config via `docker compose run --rm openclaw-cli config set/get`
- No pre-generated `openclaw.json` — config created by `onboard` and subsequent `config set` calls

## Setup Flow (setup.sh)
1. Create host dirs (`data/config/`, `data/workspace/`, `data/config/identity/`)
2. Generate or reuse gateway token
3. Write runtime values to `.env.openclaw` via `upsert_env`
4. `docker compose build`
5. `openclaw-cli onboard --no-install-daemon` (interactive)
6. `config set gateway.auth.mode token` + `config set gateway.auth.token <token>`
7. `config set gateway.controlUi.dangerouslyDisableDeviceAuth true` (upstream bug workaround)
8. `config set gateway.trustedProxies [Docker CIDRs]`
9. `ensure_control_ui_allowed_origins` (idempotent)
10. `docker compose up -d`

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
