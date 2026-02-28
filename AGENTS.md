# AGENTS.md

## Repository Overview

This repository provides a Go CLI that generates **OpenClaw Docker deployment artifacts** — a Dockerfile, Compose configuration, environment file, and setup script.

Primary goals:
- Generate a lean, reproducible Dockerfile based on the official OpenClaw Docker pattern (`node:22-bookworm`).
- Help users launch OpenClaw through Docker with secure-by-default settings using squid proxy for egress control.
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
│   ├── nginx/
│   │   ├── nginx.conf         # HTTPS reverse proxy with WebSocket support
│   │   ├── nginx-cert.pem     # TLS server cert signed by CA
│   │   └── nginx-key.pem      # TLS server key
│   ├── openclaw/
│   │   ├── Dockerfile          # Lean node:22-bookworm based Dockerfile
│   │   └── openclaw.json       # Pre-seeded gateway config (token placeholder)
│   └── squid/
│       ├── Dockerfile.squid    # Custom squid image with squid-openssl
│       ├── squid.conf          # Squid proxy config with SSL bump + domain ACLs
│       ├── ca-cert.pem         # Self-signed CA cert for SSL bump
│       └── ca-key.pem          # CA private key
├── compose.yaml                # Docker Compose with nginx, squid, gateway
├── .env.openclaw               # Environment variables for compose
├── manifest.json               # Resolved version metadata
└── setup.sh                    # Token gen, config seeding, compose up
```

## Contribution Guidelines for Agents

When modifying this repository:
- Pin versions where stability matters; document why when pinning is non-obvious.
- Avoid introducing unnecessary runtime dependencies.
- The generated Dockerfile should stay lean (no dev tools, no firewall scripts).
- Egress control is handled by squid proxy in compose, not by iptables in the container.

## Validation Expectations

Before considering work complete, agents should:
- Run `go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline` after CLI/template changes.
- Verify `openclaw-deploy/compose/openclaw/Dockerfile` exists.
- Validate generated compose with env file:
	- `docker compose --env-file ./openclaw-deploy/.env.openclaw -f ./openclaw-deploy/compose.yaml config`
- Verify `openclaw-deploy/setup.sh` exists and is executable.
- Ensure commands are non-interactive and CI-friendly.

## Safety Model

- Generation is additive and overwrite-only; directory deletion is disabled.
- `--cleanup` prints a defensive warning with the target path and does not delete files.
- By default, only overwrite writes prompt for confirmation; CI and automation should use `--dangerous-inline`.

## Current Deployment Model

- Generated `compose.yaml` includes `squid` as explicit egress proxy.
- `openclaw-gateway` and `openclaw-cli` run on an internal-only app network.
- Proxy env vars (`OPENCLAW_HTTP_PROXY`, `OPENCLAW_HTTPS_PROXY`, `OPENCLAW_NO_PROXY`) are emitted in `.env.openclaw`.
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
