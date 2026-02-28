# AGENTS.md

## Repository Overview

This repository provides a Go CLI that generates **OpenClaw Docker deployment artifacts** — a Dockerfile, Compose configuration, Envoy proxy config, environment file, and setup script.

Primary goals:
- Generate a lean, reproducible Dockerfile based on the official OpenClaw Docker pattern (`node:22-bookworm`).
- Help users launch OpenClaw through Docker with secure-by-default settings using Envoy proxy for egress control.
- Generate deployment-ready runtime artifacts (see Output Structure below).
- Mirror the official OpenClaw docker-setup.sh flow (onboarding, CLI-based config management) while layering Envoy-based network isolation on top.

## What Agents Should Assume

- The core artifact in this repo is Docker-related build configuration.
- Generated files in the output directory (`./openclaw-deploy` by default) are produced by the CLI.
- The CLI accepts a single `--openclaw-version` flag (dist-tag like `latest` or semver partial like `2026.2`).
- Version metadata comes from npm package `openclaw` via the Go CLI (`main.go`).
- Changes should prioritize compatibility, determinism, and minimal image complexity.
- Prefer small, focused edits rather than broad refactors.

## Output Structure

```
<output>/
├── compose/
│   ├── envoy/
│   │   ├── envoy.yaml          # Ingress + egress proxy config
│   │   ├── server-cert.pem     # Self-signed TLS cert for ingress
│   │   └── server-key.pem      # TLS key for ingress
│   └── openclaw/
│       ├── Dockerfile           # node:22-bookworm + iptables + gosu + pnpm + bun
│       └── entrypoint.sh        # Root-owned iptables setup, drops to node user
├── compose.yaml                 # 3 services: envoy, openclaw-gateway, openclaw-cli
├── .env.openclaw                # Runtime env vars (token, ports, proxy config)
├── manifest.json                # Resolved version metadata
└── setup.sh                     # Build, onboard, configure, start
```

## Compose Services

| Service | Purpose | Network | Restart |
|---------|---------|---------|---------|
| `envoy` | TLS termination, ingress reverse proxy, egress domain whitelist | internal + egress | unless-stopped |
| `openclaw-gateway` | OpenClaw gateway (AI agent runtime) | internal only | unless-stopped |
| `openclaw-cli` | Config management, onboarding, channel setup (run-and-exit) | internal only | none |

- `openclaw-gateway` uses the Dockerfile ENTRYPOINT (`entrypoint.sh` → iptables → gosu) with an explicit `command: ["openclaw", "gateway", "--bind", "lan", "--port", "18789"]`.
- `openclaw-cli` overrides the entrypoint to `["openclaw"]` so `docker compose run --rm openclaw-cli <subcommand>` works directly.
- Both gateway and CLI share the same image, volumes (`data/config`, `data/workspace`), and `env_file`.

## Setup Flow (setup.sh)

The generated `setup.sh` mirrors the official OpenClaw docker-setup.sh with Envoy additions:

1. Create host dirs: `data/config/`, `data/workspace/`, `data/config/identity/`
2. Generate or reuse gateway token
3. Write token + ports to `.env.openclaw`
4. `docker compose build`
5. `docker compose run --rm openclaw-cli onboard --no-install-daemon` (interactive)
6. `config set gateway.auth.mode token` + `config set gateway.auth.token <token>`
7. `config set gateway.controlUi.dangerouslyDisableDeviceAuth true` (see Known Issues)
8. `config set gateway.trustedProxies [Docker CIDRs]`
9. `ensure_control_ui_allowed_origins` (sets `gateway.controlUi.allowedOrigins`)
10. `docker compose up -d`

Gateway configuration is managed entirely via `openclaw-cli config set/get` — there is no pre-generated `openclaw.json` template.

## Known Issues

### Device auth incompatible with reverse proxy

The Control UI WebSocket connection bypasses `gateway.auth.mode` and always requires device pairing, even behind a correctly configured trusted proxy. This is an upstream bug:
- [#25293](https://github.com/openclaw/openclaw/issues/25293) — Control UI ignores trusted-proxy auth mode
- [#4941](https://github.com/openclaw/openclaw/issues/4941) — Dashboard "pairing required" in Docker

**Workaround:** `setup.sh` sets `gateway.controlUi.dangerouslyDisableDeviceAuth: true`. Token auth + TLS termination at Envoy is the actual security boundary. This should be reverted when the upstream bug is fixed.

## Contribution Guidelines for Agents

When modifying this repository:
- Pin versions where stability matters; document why when pinning is non-obvious.
- Avoid introducing unnecessary runtime dependencies.
- The generated Dockerfile installs `iptables`, `gosu`, `pnpm` (via corepack), and `bun` (via install script) beyond base.
- Never weaken the egress isolation model (see Threat Model & Egress Security below).

## Validation Expectations

Before considering work complete, agents should:
- Run `go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline` after CLI/template changes.
- Verify `openclaw-deploy/compose/openclaw/Dockerfile` exists.
- Validate generated compose:
	- `docker compose -f ./openclaw-deploy/compose.yaml config`
- Verify `openclaw-deploy/setup.sh` exists and is executable.
- Ensure commands are non-interactive and CI-friendly.
- Run `go test ./...` to verify all tests pass.

## Safety Model

- Generation is additive and overwrite-only; directory deletion is disabled.
- `--cleanup` prints a defensive warning with the target path and does not delete files.
- By default, only overwrite writes prompt for confirmation; CI and automation should use `--dangerous-inline`.

## Threat Model & Egress Security

The primary threat is a compromised or malicious AI agent instructing OpenClaw to exfiltrate data
to attacker-controlled domains using arbitrary tools and transports (`curl`, `wget`, raw sockets,
subprocesses — anything available in the container). Application-level proxy settings like
`HTTP_PROXY` env vars or Node.js preloads are insufficient because a malicious agent can bypass
them by using any tool that ignores proxy settings, or by making raw TCP connections.

**Defense-in-depth model (three layers):**

1. **Docker `internal: true` network** — the gateway container has no default route to the internet.
   There is no IP to reach. This is the hard network boundary.

2. **Root-owned iptables rules** — set by `entrypoint.sh` running as root before dropping to the
   `node` user. Default policy: `OUTPUT DROP`. Only allows: loopback, Docker DNS (127.0.0.11:53),
   established/related connections, and traffic to the Envoy container's IP. The `node` user
   **cannot modify these rules** — `CAP_NET_ADMIN` is only available to root, and the entrypoint
   drops to `node` via `gosu` after configuring iptables.

3. **Envoy domain whitelist** — the egress listener only tunnels HTTP CONNECT requests to
   whitelisted domains. Everything else gets 403. No SSL bump / MITM — TLS is end-to-end.

`HTTP_PROXY`/`HTTPS_PROXY` env vars are set as a convenience so proxy-aware tools like `curl`
route correctly, but they are **not** the security boundary. The iptables rules + `internal: true`
network are.

**Key invariants (do not weaken):**
- Gateway container must use `cap_add: [NET_ADMIN]` in compose (needed by root during init only).
- Entrypoint must run as root, set iptables, then `exec gosu node "$@"` — never skip the drop.
- `openclaw-internal` network must be `internal: true`.
- Envoy is the only container on both internal and egress networks.
- `clawhub.com` and `registry.npmjs.org` are always included in the Envoy domain whitelist.

## Egress Domain Whitelist

**Always included (hardcoded, cannot be removed):**
- `clawhub.com`
- `registry.npmjs.org`

**Default AI providers (via `--allowed-domains`, additive):**
- `api.anthropic.com`, `api.openai.com`, `generativelanguage.googleapis.com`, `openrouter.ai`, `api.x.ai`

`--allowed-domains` is additive to the hardcoded domains. Duplicates are deduplicated automatically.

## Current Deployment Model

- Generated `compose.yaml` includes 3 services: `envoy`, `openclaw-gateway`, `openclaw-cli`.
- Envoy ingress listener (:443) terminates TLS with X-Forwarded-For forwarding (`use_remote_address: true`) and reverse-proxies to gateway with WebSocket support.
- Envoy egress listener (:10000) acts as HTTP CONNECT forward proxy with domain whitelist ACL.
- `openclaw-gateway` runs on an internal-only network (`internal: true`) — no direct internet access.
- Gateway starts as root to set iptables, then drops to `node` user via `gosu`.
- Gateway has explicit `command` with `--bind lan --port 18789` to ensure LAN binding (required for Envoy to reach it over Docker network).
- `openclaw-cli` shares the same image/volumes but overrides `entrypoint: ["openclaw"]` for direct CLI access.
- Gateway trusts Docker network CIDRs (`172.16.0.0/12`, `10.0.0.0/8`, `192.168.0.0/16`) via `trustedProxies` for correct client IP detection behind Envoy.
- `setup.sh` handles image build, interactive onboarding, CLI-based config management, and compose orchestration.

## Future Steps

- Add CI validation that checks generated compose/env pairs for parse correctness.
- Keep deployment docs/examples aligned with flag changes (`--openclaw-version`, prompt semantics).
- Remove `dangerouslyDisableDeviceAuth` workaround when upstream bug is fixed.

## Out of Scope (Unless Explicitly Requested)

- Adding unrelated tooling or frameworks.
- Building registry publishing/release automation (for example pushing images to Docker Hub or GHCR).
- Changing release/versioning policy beyond the requested task.

## Editing Style

- Keep docs concise and operational.
- Keep commits scoped to one concern.
- Prefer clarity over cleverness in shell and Docker instructions.
