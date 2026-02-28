# openclaw-docker

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-supported-6E56CF)](https://docs.openclaw.ai/install/docker)
![macOS](https://img.shields.io/badge/macOS-supported-000000?logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-supported-FCC624?logo=linux&logoColor=black)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/schmitthub/openclaw-docker)

CLI that generates a hardened Docker Compose stack for [OpenClaw](https://openclaw.ai) with network-level egress isolation via Envoy proxy.

> This is in early development - I will be adding features and fixing bugs to serve my own needs as I experiment with OpenClaw. And claude boy generated a lot of the docs so they might be hallucinated slightly. But contributions and feedback are welcome!

## Install

**Homebrew:**

```bash
brew install schmitthub/tap/openclaw-docker
```

**Install script (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash
```

Options: `--global` (install to `/usr/local/bin`), `--local` (default, `~/.local/bin`), `--install-dir <dir>`, `--version <tag>`.

**From source:**

```bash
git clone https://github.com/schmitthub/openclaw-docker.git
cd openclaw-docker
go build -o openclaw-docker .
```

## Table of Contents

- [openclaw-docker](#openclaw-docker)
  - [Install](#install)
  - [Table of Contents](#table-of-contents)
  - [Architecture](#architecture)
  - [Threat Model](#threat-model)
  - [Quickstart](#quickstart)
    - [Prerequisites](#prerequisites)
    - [1. Generate deployment artifacts](#1-generate-deployment-artifacts)
    - [2. Run setup](#2-run-setup)
    - [3. Open the dashboard](#3-open-the-dashboard)
  - [Server Deployment](#server-deployment)
    - [1. Generate with `--external-origin`](#1-generate-with---external-origin)
    - [2. Point your reverse proxy at port 443](#2-point-your-reverse-proxy-at-port-443)
    - [3. Lock down the origin with mTLS (Cloudflare Authenticated Origin Pulls)](#3-lock-down-the-origin-with-mtls-cloudflare-authenticated-origin-pulls)
  - [Common Operations](#common-operations)
  - [Egress Domain Whitelist](#egress-domain-whitelist)
  - [CLI Flags](#cli-flags)
  - [Known Issues](#known-issues)
    - [Device auth behind reverse proxy](#device-auth-behind-reverse-proxy)
  - [Development](#development)
  - [Repository Structure](#repository-structure)

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│  Host                                                                │
│                                                                      │
│   Browser ─── https://localhost ──┐                                  │
│                                   │                                  │
│   ./openclaw <cmd> ───────────────┤ (docker run, wss://envoy:443)   │
│     (remote CLI client)           │  NODE_EXTRA_CA_CERTS for TLS    │
│     openclaw-cli-config volume    │  device-paired for auth         │
│                                   │                                  │
│   ┌───────────────────────────────┼────────────────────────────────┐ │
│   │  openclaw-egress network      │                                │ │
│   │                               ▼ :443                           │ │
│   │                  ┌─────────────────────────┐                   │ │
│   │    Internet ◄──► │     Envoy (172.28.0.2)  │                   │ │
│   │   (whitelisted   │  Ingress:               │                   │ │
│   │    domains       │  • TLS termination      │                   │ │
│   │    only)         │  • X-Forwarded-For      │                   │ │
│   │                  │  • WebSocket upgrade    │                   │ │
│   │                  │                         │                   │ │
│   │    Cloudflare    │  Egress (:10000):        │                   │ │
│   │    1.1.1.2  ◄──  │  • TLS Inspector (SNI)  │                   │ │
│   │    1.0.0.2       │  • Domain whitelist      │                   │ │
│   │   (malware       │  • Non-TLS = DENIED     │                   │ │
│   │    blocking)     │                         │                   │ │
│   │                  │  DNS (:53 UDP):          │                   │ │
│   │                  │  • Forwards to Cloudflare│                   │ │
│   │                  │  • Malware domains blocked│                  │ │
│   │                  └────────────┬────────────┘                   │ │
│   └───────────────────────────────┼────────────────────────────────┘ │
│   ┌───────────────────────────────┼────────────────────────────────┐ │
│   │  openclaw-internal network    │  (internal: true — NO default  │ │
│   │  subnet: 172.28.0.0/24       │   route to the internet)       │ │
│   │                               ▼                                │ │
│   │   ┌───────────────────────────────────────────┐                │ │
│   │   │          openclaw-gateway                  │                │ │
│   │   │  • OpenClaw + pnpm + bun                  │                │ │
│   │   │  • dns: [172.28.0.2] (Envoy)              │                │ │
│   │   │                                           │                │ │
│   │   │  entrypoint.sh (root-owned, immutable):   │                │ │
│   │   │  ┌───────────────────────────────────┐    │                │ │
│   │   │  │ ip route: default via Envoy       │    │                │ │
│   │   │  │ NAT:  ALL outbound TCP ──DNAT──►  │    │                │ │
│   │   │  │       Envoy:10000 (transparent)   │    │                │ │
│   │   │  │ FILTER: OUTPUT DROP (defense in   │    │                │ │
│   │   │  │         depth, only Envoy allowed) │    │                │ │
│   │   │  └───────────────────────────────────┘    │                │ │
│   │   │  • Drops to node user via gosu            │                │ │
│   │   │  • No proxy env vars — apps unaware       │                │ │
│   │   └───────────────────────────────────────────┘                │ │
│   └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│   data/config/     ← gateway config (openclaw.json, identity/)      │
│   openclaw-cli-config (named volume) ← CLI remote config + identity │
│   data/workspace/  ← bind-mounted workspace                         │
└──────────────────────────────────────────────────────────────────────┘
```

## Threat Model

**Threat:** Prompt injection coerces the AI agent into exfiltrating data. The agent can run any tool available in the container — `curl`, `wget`, `ncat`, `ssh`, raw sockets, subprocesses. It can use any port, any protocol, and target any destination. Application-level proxy settings (`HTTP_PROXY`) are trivially bypassed.

**Defense-in-depth (four layers):**

| Layer | Mechanism | What it stops | Bypassable by `node` user? |
|-------|-----------|---------------|---------------------------|
| **1. Network isolation** | Docker `internal: true` network | No default route to internet — no IP to reach | No |
| **2. iptables DNAT** | NAT table redirects ALL outbound TCP to Envoy:10000 | Every TCP connection, regardless of tool/port/protocol, goes through Envoy | No (`CAP_NET_ADMIN` required, root only) |
| **3. Envoy SNI whitelist** | TLS Inspector reads SNI from ClientHello, forwards only whitelisted domains | Non-whitelisted HTTPS, all non-TLS (SSH, HTTP, raw TCP) | No (Envoy runs separately, resolves DNS independently) |
| **4. Malware-blocking DNS** | Cloudflare 1.1.1.2 / 1.0.0.2 (via Envoy DNS listener) | Known malware, phishing, and C2 domains blocked at DNS resolution | No (Envoy resolves DNS, containers cannot override) |

**Why SNI spoofing doesn't work:** If an attacker forges the SNI to `api.anthropic.com` while connecting to `evil.com`'s IP, Envoy resolves `api.anthropic.com` via DNS independently and connects to the **real** IP — not the attacker's server.

**DNS security:** All DNS resolution from internal containers is forwarded through Envoy to Cloudflare's malware-blocking resolvers (1.1.1.2 / 1.0.0.2). These resolvers refuse to resolve known malware, phishing, and command-and-control domains — adding a DNS-layer defense even for whitelisted TLS connections. Docker's embedded DNS cannot forward external queries on `internal: true` networks, so Envoy serves as the DNS forwarder on its static IP (172.28.0.2).

**What gets blocked:**
- `curl https://evil.com` — SNI `evil.com` not in whitelist → **BLOCKED**
- `ssh user@evil.com` — no TLS, no SNI → **BLOCKED**
- `ncat evil.com 4444` — no TLS, no SNI → **BLOCKED**
- `curl http://evil.com` — no TLS, no SNI → **BLOCKED**
- `python3 -c "import socket; s=socket.socket(); s.connect(('1.2.3.4', 443))"` — no SNI → **BLOCKED**
- `curl https://api.anthropic.com` — SNI matches whitelist → **ALLOWED**

## Quickstart

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with `docker compose` v2

### 1. Generate deployment artifacts

```bash
openclaw-docker generate \
  --openclaw-version latest \
  --output ./openclaw-deploy \
  --dangerous-inline
```

This creates:

```
openclaw-deploy/
├── compose/
│   ├── envoy/
│   │   ├── envoy.yaml          # Ingress + egress proxy config
│   │   ├── server-cert.pem     # Self-signed TLS cert (SANs: localhost, envoy, gateway)
│   │   └── server-key.pem      # TLS private key
│   └── openclaw/
│       ├── Dockerfile           # node:22-bookworm + iptables + iproute2 + gosu + pnpm + bun
│       └── entrypoint.sh        # iptables setup, drops to node user
├── compose.yaml                 # 2 services: envoy, openclaw-gateway
├── .env.openclaw                # Runtime env vars (token, ports, bind)
├── setup.sh                     # Build, onboard, configure, pair CLI, start
├── openclaw                     # CLI wrapper (docker run, remote client via wss://envoy:443)
└── manifest.json                # Resolved version metadata
```

### 2. Run setup

```bash
cd openclaw-deploy
./setup.sh
```

`setup.sh` does the following in order:

1. Creates `data/config/`, `data/workspace/`, `data/config/identity/`
2. Builds Docker images (`docker compose build`)
3. Runs interactive onboarding (`openclaw onboard --no-install-daemon`)
4. Sets `gateway.mode local` (safety net — required for gateway to start)
5. Generates a gateway token (or reuses from onboarding)
6. Configures gateway auth, trusted proxies, Control UI origins, mDNS off (via `gw_config`)
7. Configures CLI for remote access (`wss://envoy:443`) on named volume (via `cli_config`)
8. Starts services (`docker compose up -d`)
9. Waits for gateway, pairs CLI device (`devices approve --latest`)

Gateway config is managed via `gw_config` (a `docker compose run --no-deps` helper that mounts the gateway's config volume). The CLI runs as a `docker run --rm` container with its config persisted on the `openclaw-cli-config` named Docker volume, connecting to the gateway through Envoy's TLS ingress as a remote client.

### 3. Open the dashboard

The setup script prints the URL at the end:

```
https://localhost/?token=<your-token>
```

Accept the self-signed certificate warning in your browser.

## Server Deployment

To expose OpenClaw on a public server behind a domain (e.g. `https://myclaw.example.com`):

### 1. Generate with `--external-origin`

```bash
openclaw-docker generate \
  --openclaw-version latest \
  --output ./openclaw-deploy \
  --external-origin "https://myclaw.example.com" \
  --dangerous-inline
```

This adds your domain to:
- The Control UI's `allowedOrigins` list alongside `https://localhost`
- The self-signed TLS certificate SANs (so `NODE_EXTRA_CA_CERTS` works for the hostname)

### 2. Point your reverse proxy at port 443

Envoy terminates TLS on port 443. Your edge proxy (Cloudflare, nginx, Caddy) should forward traffic to this port.

### 3. Lock down the origin with mTLS (Cloudflare Authenticated Origin Pulls)

The generated `compose/envoy/envoy.yaml` includes commented-out mTLS configuration. To enable it:

1. Download the [Cloudflare origin pull CA certificate](https://developers.cloudflare.com/ssl/origin-configuration/authenticated-origin-pull/)
2. Save it as `compose/envoy/cloudflare-origin-pull-ca.pem`
3. Mount it in `compose.yaml` under the `envoy` service volumes:
   ```yaml
   - ./compose/envoy/cloudflare-origin-pull-ca.pem:/etc/envoy/certs/client-ca.pem:ro
   ```
4. Uncomment the mTLS lines in `compose/envoy/envoy.yaml`:
   ```yaml
   validation_context:
     trusted_ca:
       filename: /etc/envoy/certs/client-ca.pem
   require_client_certificate: true
   ```
5. Restart Envoy: `docker compose restart envoy`

With mTLS enabled, only requests presenting a valid Cloudflare client certificate are accepted. Direct connections to the origin IP are rejected.

## Common Operations

The `./openclaw` wrapper runs a `docker run --rm` container that connects to the gateway as a remote client via `wss://envoy:443`. Config and device identity persist on the `openclaw-cli-config` named Docker volume. Trusts the self-signed TLS cert via `NODE_EXTRA_CA_CERTS`.

```bash
# List paired devices
./openclaw-deploy/openclaw devices list

# Channel setup (WhatsApp QR, Telegram bot, Discord bot)
./openclaw-deploy/openclaw channels login
./openclaw-deploy/openclaw channels add --channel telegram --token <token>
./openclaw-deploy/openclaw channels add --channel discord --token <token>

# List Discord pairing requests
./openclaw-deploy/openclaw pairing list discord

# Approve a Discord pairing request
./openclaw-deploy/openclaw pairing approve discord <CODE>

# View gateway logs
docker compose -f ./openclaw-deploy/compose.yaml logs -f openclaw-gateway

# Restart gateway after config changes
docker compose -f ./openclaw-deploy/compose.yaml restart openclaw-gateway

# Restart after editing envoy.yaml
docker compose -f ./openclaw-deploy/compose.yaml restart envoy

# Stop everything
docker compose -f ./openclaw-deploy/compose.yaml down
```

**Gateway config changes** (e.g., auth, trusted proxies) require direct file access via compose:

```bash
cd openclaw-deploy
docker compose run --rm --no-deps --entrypoint openclaw openclaw-gateway config get gateway
docker compose run --rm --no-deps --entrypoint openclaw openclaw-gateway config set <key> <value>
docker compose restart openclaw-gateway  # apply changes
```

## Egress Domain Whitelist

Envoy only forwards TLS connections with whitelisted SNI. All other traffic (non-TLS, non-whitelisted) is denied.

**Always included (hardcoded, cannot be removed):**

| Category | Domains |
|----------|---------|
| Infrastructure | `clawhub.com`, `registry.npmjs.org` |
| AI providers | `api.anthropic.com`, `api.openai.com`, `generativelanguage.googleapis.com`, `openrouter.ai`, `api.x.ai` |

`--allowed-domains` is **additive** — all hardcoded domains are always present. To add custom domains:

```bash
openclaw-docker generate \
  --allowed-domains "api.anthropic.com,api.openai.com,custom.example.com" \
  --output ./openclaw-deploy \
  --dangerous-inline
```

To edit the whitelist after generation, modify `compose/envoy/envoy.yaml` directly and restart Envoy.

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--openclaw-version` | `latest` | OpenClaw version (dist-tag or semver partial) |
| `--output`, `-o` | `./openclaw-deploy` | Output directory |
| `--allowed-domains` | AI providers | Comma-separated egress whitelist (additive) |
| `--external-origin` | `""` | External origin for server deployments (e.g. `https://myclaw.example.com`) |
| `--docker-apt-packages` | `""` | Extra apt packages for Dockerfile |
| `--openclaw-gateway-port` | `18789` | Gateway port |
| `--openclaw-gateway-bind` | `lan` | Gateway bind address |
| `--config`, `-f` | none | YAML config file path |
| `--dangerous-inline` | `false` | Skip write confirmation prompts |

Config precedence: **flags > env vars (`OPENCLAW_DOCKER_*`) > config file > defaults**

## Known Issues

### Device auth behind reverse proxy

The OpenClaw Control UI WebSocket connection bypasses `gateway.auth.mode` and always requires device pairing, even when running behind a trusted proxy with correct headers. This is an [upstream bug](https://github.com/openclaw/openclaw/issues/25293) ([#4941](https://github.com/openclaw/openclaw/issues/4941)).

**Current approach:** `setup.sh` pairs the CLI device automatically during setup (`devices approve --latest`). The CLI connects through Envoy's TLS ingress (`wss://envoy:443`) as a remote client with its own device identity. Token auth + TLS termination at Envoy is the security boundary for the Control UI.

## Development

```bash
go build .                  # compile CLI
go test ./...               # run all tests
go vet ./...                # static analysis
make check                  # test + vet + lint

# generate and validate artifacts
go run . generate --openclaw-version latest --output ./openclaw-deploy --dangerous-inline
docker compose -f ./openclaw-deploy/compose.yaml config
```

## Repository Structure

```
main.go                     # CLI entrypoint
internal/
  cmd/                      # Cobra commands (root, generate, config, version)
  render/                   # Artifact generation (Dockerfile, compose, envoy, setup.sh)
  versions/                 # npm version resolution, manifest I/O
  config/                   # YAML config loading
  build/                    # Build metadata (version/date via ldflags)
  update/                   # GitHub release update checks
  testenv/                  # Isolated test environments
e2e/                        # End-to-end generation tests
  harness/                  # Test harness (isolated FS + Cobra execution)
```
