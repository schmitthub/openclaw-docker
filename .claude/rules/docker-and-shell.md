---
globs: ["**/Dockerfile*", "**/compose*.yaml", "**/.env*", "**/*.sh", "internal/render/**/*.go"]
---

# Docker & Shell Artifact Rules

## Generated Artifacts
The CLI generates files to `<output>/` (default `./openclaw-deploy`), organized by service:

**Root files:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `compose.yaml` | 0644 | Nginx + squid + gateway services on internal network |
| `.env.openclaw` | 0644 | Runtime env vars including proxy config |
| `setup.sh` | 0755 | Token gen, openclaw.json seeding, compose up orchestration |
| `manifest.json` | 0644 | Resolved version metadata |

**`compose/openclaw/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `Dockerfile` | 0644 | Lean `node:22-bookworm` image with OpenClaw installed via curl |
| `openclaw.json` | 0644 | Pre-seeded gateway config (token placeholder replaced by setup.sh) |

**`compose/squid/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `Dockerfile.squid` | 0644 | Custom squid image with `squid-openssl` for SSL bump support |
| `squid.conf` | 0644 | Squid proxy config with SSL bump + domain whitelist ACLs |
| `ca-cert.pem` | 0644 | Self-signed CA cert for squid SSL bump (cross-referenced by gateway) |
| `ca-key.pem` | 0600 | CA private key (mounted into squid only) |

**`compose/nginx/`:**
| File | Permissions | Purpose |
|------|-------------|---------|
| `nginx.conf` | 0644 | HTTPS reverse proxy with WebSocket support + commented-out mTLS |
| `nginx-cert.pem` | 0644 | TLS server cert signed by CA (for nginx HTTPS on port 443) |
| `nginx-key.pem` | 0600 | TLS server key for nginx |

## Dockerfile Conventions
- Base: `node:22-bookworm` (matches official OpenClaw Docker pattern)
- User: `node` (from base image, not custom)
- No `ENTRYPOINT` — only `CMD`
- No firewall scripts, no dev tools (zsh, git-delta, hadolint, fzf)
- Install via: `curl -fsSL "https://openclaw.ai/install.sh" | bash`
- `OPENCLAW_DOCKER_APT_PACKAGES` ARG for optional packages

## Compose Conventions
- nginx (`nginx:alpine`) is the sole ingress — publishes port 443, proxies to gateway
- Gateway and squid build from `compose/<service>/` subdirectories; nginx uses stock image
- Build contexts: `./compose/openclaw` (gateway), `./compose/squid` (squid)
- Gateway has no published ports — only accessible via nginx on internal network
- Squid proxy on both `openclaw-internal` and `openclaw-egress` networks
- Gateway on `openclaw-internal` (internal: true) only — all egress routes through squid
- Squid SSL-bumps TLS with CA cert; gateway trusts it via `NODE_EXTRA_CA_CERTS`
- Gateway cross-references CA cert from squid dir: `./compose/squid/ca-cert.pem`
- nginx TLS cert is signed by the same CA; users can swap for production certs
- Named volumes: `squid-log`, `squid-cache` for squid persistence
- Env vars sourced from `.env.openclaw` via `env_file`

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
- `composeFileContent()` — compose YAML as string-joined lines
- `openClawEnvFileContent(opts)` — env file via `fmt.Sprintf`
- `setupScriptContent(opts)` — setup.sh via `fmt.Sprintf`
- `squidDockerfileContent()` — Dockerfile.squid content
- `squidConfContent(opts)` — squid.conf with SSL bump + domain ACLs
- `openClawJSONContent(opts)` — openclaw.json with gateway config
- `generateCA(opts)` — CA cert+key generation (in `ca.go`, preserves existing across re-runs)
- `generateNginxCert(opts)` — TLS server cert signed by CA (in `ca.go`)
- `nginxConfContent(opts)` — nginx.conf with HTTPS reverse proxy + mTLS comments

## Validation
```bash
# Generate artifacts
go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline

# Validate compose
docker compose --env-file ./openclaw-deploy/.env.openclaw -f ./openclaw-deploy/compose.yaml config

# Check setup.sh is executable
test -x ./openclaw-deploy/setup.sh
```
