# AGENTS.md

## Repository Overview

This repository provides a Go CLI that generates **OpenClaw Docker deployment artifacts** — a Dockerfile, Compose configuration, environment file, and setup script.

Primary goals:
- Generate a lean, reproducible Dockerfile based on the official OpenClaw Docker pattern (`node:22-bookworm`).
- Help users launch OpenClaw through Docker with secure-by-default settings using Envoy proxy for egress control.
- Generate deployment-ready runtime artifacts (`Dockerfile`, `compose.yaml`, `.env.openclaw`, `setup.sh`).
- Make it straightforward to maintain and update generation logic over time.

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
│       ├── Dockerfile           # node:22-bookworm with iptables + gosu
│       └── entrypoint.sh        # Root-owned iptables setup, drops to node user
├── compose.yaml                 # Docker Compose with envoy + gateway + cli
├── .env.openclaw                # Environment variables for compose
├── manifest.json                # Resolved version metadata
└── setup.sh                     # Token gen, onboarding, compose up
```

## Contribution Guidelines for Agents

When modifying this repository:
- Pin versions where stability matters; document why when pinning is non-obvious.
- Avoid introducing unnecessary runtime dependencies.
- The generated Dockerfile installs only `iptables` and `gosu` beyond base — no dev tools.
- Never weaken the egress isolation model (see Threat Model & Egress Security below).

## Validation Expectations

Before considering work complete, agents should:
- Run `go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline` after CLI/template changes.
- Verify `openclaw-deploy/compose/openclaw/Dockerfile` exists.
- Validate generated compose:
	- `docker compose -f ./openclaw-deploy/compose.yaml config`
- Verify `openclaw-deploy/setup.sh` exists and is executable.
- Ensure commands are non-interactive and CI-friendly.

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
- `openclaw.ai` is always included in the Envoy domain whitelist.

## Current Deployment Model

- Generated `compose.yaml` includes `envoy` as unified ingress/egress proxy.
- Envoy ingress listener (:443) terminates TLS and reverse-proxies to gateway with WebSocket support.
- Envoy egress listener (:10000) acts as HTTP CONNECT forward proxy with domain whitelist ACL.
- `openclaw-gateway` runs on an internal-only network (`internal: true`) — no direct internet access.
- Gateway container starts as root to set iptables, then drops to `node` user via `gosu`.
- `setup.sh` handles image build/pull, gateway token generation, and compose orchestration.

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
