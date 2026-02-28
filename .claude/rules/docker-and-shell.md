---
globs: ["**/Dockerfile*", "**/compose*.yaml", "**/.env*", "**/*.sh", "internal/render/**/*.go"]
---

# Docker & Shell Artifact Rules

## Generated Artifacts
The CLI generates four files to `<output>/` (default `./openclaw-deploy`):

| File | Permissions | Purpose |
|------|-------------|---------|
| `Dockerfile` | 0644 | Lean `node:22-bookworm` image with OpenClaw installed via curl |
| `compose.yaml` | 0644 | Squid proxy + gateway + cli services on internal network |
| `setup.sh` | 0755 | Token gen, `docker compose build`, compose up orchestration |
| `.env.openclaw` | 0644 | Runtime env vars including proxy config |

## Dockerfile Conventions
- Base: `node:22-bookworm` (matches official OpenClaw Docker pattern)
- User: `node` (from base image, not custom)
- No `ENTRYPOINT` — only `CMD`
- No firewall scripts, no dev tools (zsh, git-delta, hadolint, fzf)
- Install via: `curl -fsSL "https://openclaw.ai/install.sh" | bash`
- `OPENCLAW_DOCKER_APT_PACKAGES` ARG for optional packages

## Compose Conventions
- Services build from local Dockerfile (`build: context: . / dockerfile: Dockerfile`)
- No `image:` tag references — always local build
- Squid proxy on `openclaw-egress` network for outbound traffic
- Gateway and CLI on `openclaw-internal` (internal: true) network
- Env vars sourced from `.env.openclaw` via `env_file`

## Shell Script Conventions
- Shebang: `#!/usr/bin/env bash`
- `set -euo pipefail`
- Bash 3.2 compatible (macOS default)
- Token gen: `openssl rand -hex 32` with python3 fallback
- Uses `docker compose -f "$COMPOSE_FILE" build` (not standalone docker build)

## Generation Code
All generation lives in `internal/render/render.go`:
- `Generate(opts)` — orchestrates all four files
- `dockerfileFor(opts)` — Dockerfile content via `fmt.Sprintf`
- `composeFileContent()` — compose YAML as string-joined lines
- `openClawEnvFileContent(opts)` — env file via `fmt.Sprintf`
- `setupScriptContent(opts)` — setup.sh via `fmt.Sprintf`

## Validation
```bash
# Generate artifacts
go run . --openclaw-version latest --output ./openclaw-deploy --dangerous-inline

# Validate compose
docker compose --env-file ./openclaw-deploy/.env.openclaw -f ./openclaw-deploy/compose.yaml config

# Check setup.sh is executable
test -x ./openclaw-deploy/setup.sh
```
