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
│   │   ├── server-cert.pem     # Self-signed TLS cert (SANs: localhost, envoy, gateway, --external-origin)
│   │   └── server-key.pem      # TLS key for ingress
│   └── openclaw/
│       ├── Dockerfile           # node:22-bookworm + iptables + iproute2 + gosu + libsecret-tools + pnpm + bun + brew
│       └── entrypoint.sh        # Root-owned iptables setup, drops to node user
├── compose.yaml                 # 2 services: envoy, openclaw-gateway
├── .env.openclaw                # Runtime env vars (token, ports, bind settings)
├── manifest.json                # Resolved version metadata
├── openclaw                     # CLI wrapper (docker run, remote client via wss://envoy:443)
└── setup.sh                     # Build, onboard, configure, pair CLI, start
```

## Compose Services

| Service | Purpose | Network | Restart |
|---------|---------|---------|---------|
| `envoy` | TLS termination, ingress reverse proxy, egress domain whitelist, DNS forwarder | internal (172.28.0.2) + egress | unless-stopped |
| `openclaw-gateway` | OpenClaw gateway (AI agent runtime) | internal only | unless-stopped |

- `openclaw-gateway` uses the Dockerfile ENTRYPOINT (`entrypoint.sh` → iptables → gosu) with an explicit `command: ["openclaw", "gateway", "--bind", "lan", "--port", "18789"]`.
- The CLI is **not** a compose service. It runs as a standalone `docker run --rm` container on the `openclaw-egress` network, connecting to the gateway through Envoy's TLS ingress (`wss://envoy:443`) as a remote client. Config and device identity persist via bind mount at `data/cli-config/`. Trusts the self-signed cert via `NODE_EXTRA_CA_CERTS`. The `./openclaw` wrapper script handles this.

## Setup Flow (setup.sh)

The generated `setup.sh` mirrors the official OpenClaw docker-setup.sh with Envoy additions. Supports `--skip-onboarding` flag to reuse existing gateway config.

1. Create host dirs: `data/config/`, `data/workspace/`, `data/config/identity/`
2. `docker compose build`
3. `docker compose run --rm openclaw-gateway openclaw onboard --no-install-daemon` (interactive, skipped with `--skip-onboarding`)
4. `gw_config config set gateway.mode local` (safety net — required for gateway to start)
5. Generate or reuse gateway token, set `gateway.auth.mode token` + `gateway.auth.token`
6. `gw_config config set gateway.trustedProxies [Docker CIDRs]`
7. `ensure_control_ui_allowed_origins` (sets `gateway.controlUi.allowedOrigins`)
8. `gw_config config set discovery.mdns.mode off`
9. Configure CLI remote access via `./openclaw` wrapper: `gateway.mode remote` + `gateway.remote.url wss://envoy:443` + `gateway.remote.transport direct` + `gateway.remote.token`
10. `docker compose up -d`, wait for gateway
11. Pair CLI device: `./openclaw devices list` (triggers pairing) + `devices approve --latest`

Gateway config helper in setup.sh:
- `gw_config`: `docker compose run --rm --no-deps openclaw-gateway openclaw "$@"` — passes `openclaw` as CMD so entrypoint.sh runs first (iptables + gosu), then executes `openclaw` as `node` user. `--no-deps` skips starting Envoy.

CLI config is done via the `./openclaw` wrapper script directly (standalone `docker run --rm` with `--entrypoint openclaw` to bypass entrypoint.sh since the CLI container has no `CAP_NET_ADMIN` and isn't on the internal network).

Gateway configuration is managed via direct file access (`gw_config`) — there is no pre-generated `openclaw.json` template. The CLI connects as a remote client with its own config directory at `data/cli-config/`.

## Known Issues

### Device auth behind reverse proxy

The Control UI WebSocket connection bypasses `gateway.auth.mode` and always requires device pairing, even behind a correctly configured trusted proxy. This is an upstream bug:
- [#25293](https://github.com/openclaw/openclaw/issues/25293) — Control UI ignores trusted-proxy auth mode
- [#4941](https://github.com/openclaw/openclaw/issues/4941) — Dashboard "pairing required" in Docker

**Current approach:** `setup.sh` pairs the CLI device during setup (`devices approve --latest`). The CLI connects through Envoy's TLS ingress as a remote client with its own device identity. Token auth + TLS termination at Envoy is the security boundary for the Control UI.

## Contribution Guidelines for Agents

When modifying this repository:
- Pin versions where stability matters; document why when pinning is non-obvious.
- Avoid introducing unnecessary runtime dependencies.
- The generated Dockerfile installs `iptables`, `iproute2`, `gosu`, `libsecret-tools`, `pnpm` (via npm), `bun` (copied to `/usr/local/bin/`), and Homebrew (Linuxbrew) beyond base.
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
to attacker-controlled domains using arbitrary tools and transports (`curl`, `wget`, `ncat`, `ssh`,
raw sockets, subprocesses — anything available in the container). Application-level proxy settings
like `HTTP_PROXY` env vars are insufficient because a prompt-injected agent can use **any tool**
that ignores proxy settings, connect on **any port**, or use **any protocol**.

**Defense-in-depth model (four layers):**

1. **Docker `internal: true` network** — the gateway container has no default route to the internet.
   There is no IP to reach. This is the hard network boundary. The entrypoint adds a default route
   via Envoy (`ip route add default via $ENVOY_IP`) so the kernel can make routing decisions for
   iptables DNAT to fire — without this, connections to external IPs fail with "Network is unreachable"
   before iptables can rewrite them.

2. **Root-owned iptables DNAT + FILTER rules** — set by `entrypoint.sh` running as root before
   dropping to the `node` user. The entrypoint derives `INTERNAL_SUBNET` from Envoy's IP (strip
   last octet, append `.0/24`). The NAT table skips DNAT for loopback (`-o lo`) and the internal
   subnet, then transparently redirects **all other outbound TCP** to Envoy's proxy listener via
   DNAT. This allows gateway health checks (loopback) and container-to-container traffic (internal
   subnet) to work without hitting Envoy, while all external traffic is captured. The FILTER table
   provides defense-in-depth with `OUTPUT DROP` default policy, only allowing loopback, Docker DNS,
   established/related, and the internal subnet. The `node` user **cannot modify these rules** —
   `CAP_NET_ADMIN` is only available to root, and the entrypoint drops to `node` via `gosu` after
   configuring iptables. The entrypoint also restores Docker's `DOCKER_OUTPUT` chain jump after
   flushing nat OUTPUT (Docker's embedded DNS uses this chain to DNAT port 53 to a high port).

3. **Envoy SNI-based domain whitelist** — the egress listener uses TLS Inspector to read the SNI
   from the TLS ClientHello without terminating TLS (no MITM). Only connections with whitelisted
   SNI values are forwarded via dynamic DNS resolution. Non-TLS traffic (SSH, plain HTTP, raw TCP)
   has no SNI and is categorically denied. Non-whitelisted SNI is denied. SNI spoofing is useless
   because Envoy resolves the domain independently — a forged SNI pointing to a different IP still
   connects to the real domain, not the attacker's server.

4. **Malware-blocking DNS** — Envoy runs a DNS listener (:53 UDP) that forwards all DNS queries to
   Cloudflare's malware-blocking resolvers (1.1.1.2 / 1.0.0.2). These resolvers refuse to resolve
   known malware, phishing, and command-and-control domains. Docker's embedded DNS cannot forward
   external queries on `internal: true` networks, so all containers use `dns: [172.28.0.2]`
   (Envoy's static IP) for DNS resolution.

No `HTTP_PROXY`/`HTTPS_PROXY` env vars are used. The transparent iptables DNAT captures all
outbound TCP regardless of what tool, port, or protocol is used.

**Key invariants (do not weaken):**
- Gateway container must use `cap_add: [NET_ADMIN]` in compose (needed by root during init only).
- Entrypoint must run as root, set iptables (NAT DNAT + FILTER DROP), then `exec gosu node "$@"` — never skip the drop.
- Entrypoint must restore `DOCKER_OUTPUT` chain jump after flushing nat OUTPUT (Docker DNS depends on it).
- Entrypoint must add default route via Envoy (`ip route add default via $ENVOY_IP`) before iptables rules.
- Entrypoint must derive `INTERNAL_SUBNET` from Envoy's IP and skip DNAT for loopback + internal subnet.
- Entrypoint NAT table must DNAT all non-local, non-subnet outbound TCP to Envoy's transparent proxy listener.
- Entrypoint FILTER table must ACCEPT internal subnet traffic (container-to-container, service discovery).
- `openclaw-internal` network must be `internal: true` with IPAM subnet `172.28.0.0/24`.
- Envoy must have static IP `172.28.0.2` on the internal network.
- Gateway service must use `dns: [172.28.0.2]` (Envoy DNS listener).
- Envoy is the only container on both internal and egress networks.
- All hardcoded domains (infrastructure + AI providers) are always included in the Envoy domain whitelist.
- Envoy DNS listener must forward to Cloudflare malware-blocking resolvers (1.1.1.2 / 1.0.0.2).

## Egress Domain Whitelist

All domains below are hardcoded and always included. They cannot be removed.

**Infrastructure:**
- `clawhub.com`
- `registry.npmjs.org`

**AI providers:**
- `api.anthropic.com`, `api.openai.com`, `generativelanguage.googleapis.com`, `openrouter.ai`, `api.x.ai`

**Homebrew (Linuxbrew):**
- `github.com`, `*.githubusercontent.com`, `ghcr.io`, `formulae.brew.sh`

`--allowed-domains` is **additive** to all hardcoded domains. Duplicates are deduplicated automatically.
Domain filtering uses TLS SNI inspection — non-TLS protocols are categorically denied.

## Current Deployment Model

- Generated `compose.yaml` includes 2 services: `envoy`, `openclaw-gateway`.
- Envoy has static IP `172.28.0.2` on `openclaw-internal` (IPAM subnet `172.28.0.0/24`).
- Envoy ingress listener (:443) terminates TLS with X-Forwarded-For forwarding (`use_remote_address: true`) and reverse-proxies to gateway with WebSocket support.
- Envoy egress listener (:10000) acts as transparent TLS proxy with SNI-based domain whitelist. All outbound TCP from the gateway is DNAT'd here by iptables. Non-TLS and non-whitelisted traffic is denied.
- Envoy DNS listener (:53 UDP) forwards DNS queries to Cloudflare malware-blocking resolvers (1.1.1.2 / 1.0.0.2). Docker's embedded DNS cannot forward external queries on `internal: true` networks.
- `openclaw-gateway` runs on an internal-only network (`internal: true`) — no direct internet access.
- Gateway service uses `dns: [172.28.0.2]` so Docker DNS forwards external queries to Envoy.
- Gateway starts as root to add default route via Envoy, set iptables (NAT DNAT + FILTER DROP), then drops to `node` user via `gosu`.
- Gateway has explicit `command` with `--bind lan --port 18789` to ensure LAN binding (required for Envoy to reach it over Docker network).
- The CLI is not a compose service — it runs as a standalone `docker run --rm` container on the `openclaw-egress` network, connecting to the gateway through Envoy's TLS ingress (`wss://envoy:443`) as a remote client. Config and device identity persist via bind mount at `data/cli-config/`. Trusts the self-signed cert via `NODE_EXTRA_CA_CERTS`, authenticates via device pairing. Uses `--entrypoint openclaw` to bypass entrypoint.sh (no `CAP_NET_ADMIN`, not on internal network).
- No `HTTP_PROXY`/`HTTPS_PROXY` env vars — iptables DNAT provides transparent egress routing.
- Gateway trusts Docker network CIDRs (`172.16.0.0/12`, `10.0.0.0/8`, `192.168.0.0/16`) via `trustedProxies` for correct client IP detection behind Envoy.
- `setup.sh` handles image build, interactive onboarding (skippable via `--skip-onboarding`), gateway config (via `gw_config`), CLI remote config (via `./openclaw` wrapper), compose orchestration, and device pairing.
- Self-signed TLS cert SANs include `DNS:localhost`, `DNS:openclaw-gateway`, `DNS:envoy`, `IP:172.28.0.2`, plus `--external-origin` hostname if set.

## Future Steps

- Add CI validation that checks generated compose/env pairs for parse correctness.
- Keep deployment docs/examples aligned with flag changes (`--openclaw-version`, prompt semantics).

## Out of Scope (Unless Explicitly Requested)

- Adding unrelated tooling or frameworks.
- Building registry publishing/release automation (for example pushing images to Docker Hub or GHCR).
- Changing release/versioning policy beyond the requested task.

## Editing Style

- Keep docs concise and operational.
- Keep commits scoped to one concern.
- Prefer clarity over cleverness in shell and Docker instructions.
