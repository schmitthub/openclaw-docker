# openclaw-docker

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-supported-6E56CF)](https://docs.openclaw.ai/install/docker)
![macOS](https://img.shields.io/badge/macOS-supported-000000?logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-supported-FCC624?logo=linux&logoColor=black)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/schmitthub/openclaw-docker)

CLI that generates a hardened Docker Compose stack for [OpenClaw](https://openclaw.ai) with network-level egress isolation via Envoy proxy.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Host                                                               │
│                                                                     │
│   Browser ─── https://localhost ──┐                                 │
│                                   │                                 │
│   ┌───────────────────────────────┼───────────────────────────────┐ │
│   │  openclaw-egress network      │                               │ │
│   │                               ▼ :443                          │ │
│   │                  ┌────────────────────────┐                   │ │
│   │    Internet ◄──► │        Envoy           │                   │ │
│   │   (allowed       │  • TLS termination     │                   │ │
│   │    domains       │  • X-Forwarded-For     │                   │ │
│   │    only)         │  • WebSocket upgrade   │                   │ │
│   │                  │  • Domain whitelist ACL │                   │ │
│   │                  └───────────┬────────────┘                   │ │
│   └──────────────────────────────┼────────────────────────────────┘ │
│   ┌──────────────────────────────┼────────────────────────────────┐ │
│   │  openclaw-internal network   │  (internal: true — NO default  │ │
│   │                              │   route to the internet)       │ │
│   │                              ▼ :10000 (egress)                │ │
│   │   ┌──────────────────────────────────────────┐                │ │
│   │   │         openclaw-gateway                  │                │ │
│   │   │  • OpenClaw + pnpm + bun                 │                │ │
│   │   │  • iptables OUTPUT DROP (root-owned)     │                │ │
│   │   │  • Only allows: loopback, DNS, Envoy     │                │ │
│   │   │  • Drops to node user via gosu           │                │ │
│   │   │  • HTTPS_PROXY=http://envoy:10000        │                │ │
│   │   └──────────────────────────────────────────┘                │ │
│   │                                                               │ │
│   │   ┌──────────────────────────────────────────┐                │ │
│   │   │         openclaw-cli                      │                │ │
│   │   │  • Same image, entrypoint: ["openclaw"]  │                │ │
│   │   │  • Config management (onboard, config)   │                │ │
│   │   │  • Channel setup (WhatsApp, Telegram)    │                │ │
│   │   │  • Run-and-exit (no restart policy)      │                │ │
│   │   └──────────────────────────────────────────┘                │ │
│   └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
│   data/config/    ← bind-mounted config (openclaw.json, identity/) │
│   data/workspace/ ← bind-mounted workspace                         │
└─────────────────────────────────────────────────────────────────────┘
```

**Three layers of egress defense:**

1. **Docker `internal: true` network** — gateway has no default route to the internet. There is no IP to reach.
2. **Root-owned iptables rules** — `OUTPUT DROP` default policy. Only loopback, Docker DNS, and Envoy are allowed. The `node` user cannot modify these rules.
3. **Envoy domain whitelist** — egress listener only tunnels HTTPS CONNECT to whitelisted domains. Everything else gets 403. No SSL bump — TLS is end-to-end.

## Quickstart

### Prerequisites

- [Go 1.25+](https://go.dev/dl/) (to build the CLI)
- [Docker](https://docs.docker.com/get-docker/) with `docker compose` v2

### 1. Build the CLI

```bash
git clone https://github.com/schmitthub/openclaw-docker.git
cd openclaw-docker
go build -o openclaw-docker .
```

Or install directly:

```bash
curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash
```

### 2. Generate deployment artifacts

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
│   │   ├── server-cert.pem     # Self-signed TLS cert
│   │   └── server-key.pem      # TLS private key
│   └── openclaw/
│       ├── Dockerfile           # node:22-bookworm + iptables + gosu + pnpm + bun
│       └── entrypoint.sh        # iptables setup, drops to node user
├── compose.yaml                 # 3 services: envoy, gateway, cli
├── .env.openclaw                # Runtime env vars (token, ports, proxy)
├── setup.sh                     # Interactive setup: build, onboard, configure, start
└── manifest.json                # Resolved version metadata
```

### 3. Run setup

```bash
cd openclaw-deploy
./setup.sh
```

`setup.sh` does the following in order:

1. Creates `data/config/`, `data/workspace/`, `data/config/identity/`
2. Generates a gateway token (or reuses existing)
3. Writes token + ports to `.env.openclaw`
4. Builds Docker images (`docker compose build`)
5. Runs interactive onboarding (`openclaw onboard --no-install-daemon`)
6. Sets gateway auth token via CLI (ensures config matches `.env.openclaw`)
7. Disables device auth ([upstream bug](https://github.com/openclaw/openclaw/issues/25293) — incompatible with reverse proxy)
8. Configures trusted proxies for Docker network CIDRs
9. Sets Control UI allowed origins (`https://localhost`)
10. Starts services (`docker compose up -d`)

### 4. Open the dashboard

The setup script prints the URL at the end:

```
https://localhost/?token=<your-token>
```

Accept the self-signed certificate warning in your browser.

## Common Operations

```bash
# View gateway logs
docker compose -f ./openclaw-deploy/compose.yaml logs -f openclaw-gateway

# View gateway config
docker compose -f ./openclaw-deploy/compose.yaml run --rm openclaw-cli config get gateway

# Add WhatsApp (QR code)
docker compose -f ./openclaw-deploy/compose.yaml run --rm openclaw-cli channels login

# Add Telegram bot
docker compose -f ./openclaw-deploy/compose.yaml run --rm openclaw-cli channels add --channel telegram --token <token>

# Add Discord bot
docker compose -f ./openclaw-deploy/compose.yaml run --rm openclaw-cli channels add --channel discord --token <token>

# Restart after editing envoy.yaml
docker compose -f ./openclaw-deploy/compose.yaml restart envoy

# Stop everything
docker compose -f ./openclaw-deploy/compose.yaml down
```

## Egress Domain Whitelist

The Envoy egress proxy only allows HTTPS CONNECT to whitelisted domains. The default list includes:

**Always included (hardcoded):**
- `clawhub.com`
- `registry.npmjs.org`

**Default AI providers (configurable via `--allowed-domains`):**
- `api.anthropic.com`
- `api.openai.com`
- `generativelanguage.googleapis.com`
- `openrouter.ai`
- `api.x.ai`

`--allowed-domains` is **additive** — the hardcoded domains are always present. To add custom domains:

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
| `--external-origin` | `""` | External origin for server deployments |
| `--docker-apt-packages` | `""` | Extra apt packages for Dockerfile |
| `--openclaw-gateway-port` | `18789` | Gateway port |
| `--openclaw-gateway-bind` | `lan` | Gateway bind address |
| `--config`, `-f` | none | YAML config file path |
| `--dangerous-inline` | `false` | Skip write confirmation prompts |

Config precedence: **flags > env vars (`OPENCLAW_DOCKER_*`) > config file > defaults**

## Known Issues

### Device auth behind reverse proxy

The OpenClaw Control UI WebSocket connection bypasses `gateway.auth.mode` and always requires device pairing, even when running behind a trusted proxy with correct headers. This is an [upstream bug](https://github.com/openclaw/openclaw/issues/25293) ([#4941](https://github.com/openclaw/openclaw/issues/4941)).

**Workaround:** `setup.sh` automatically sets `gateway.controlUi.dangerouslyDisableDeviceAuth: true`. Token auth + TLS termination at Envoy is the actual security boundary.

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
