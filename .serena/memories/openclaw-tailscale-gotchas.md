# OpenClaw Tailscale Sidecar Gotchas

## Date: 2026-03-04
## Context: Migrating from Tailscale-inside-gateway to Tailscale sidecar container

## Why Sidecar?
- Tailscale DERP hostnames changed from `derpN.tailscale.com` to `derpNx.tailscale.com` (lettered suffixes)
- Only derp1-12 retain old hostnames; derp13-28 are NXDOMAIN
- Envoy SNI whitelist can't keep up with hostname changes
- Sidecar with direct internet access avoids proxying DERP/STUN traffic entirely

## Gotchas

### 1. `tailscale` CLI must be accessible inside gateway container
OpenClaw shells out to `tailscale` in several places:
- `tailscale serve --bg --yes <port>` — `src/infra/tailscale.ts:391`
- `tailscale status --json` — `src/infra/tailscale.ts:117` (discover tailnet hostname)
- `tailscale whois --json <ip>` — `src/infra/tailscale.ts:488` (auth verification)

**Fix:** Mount the `tailscale` CLI binary into the gateway container, or install just the CLI (not `tailscaled`) pointing at the shared socket.

### 2. Shared `tailscaled` socket
The CLI talks to `tailscaled` via Unix socket (default: `/var/run/tailscale/tailscaled.sock`).
Share this socket via volume mount. Set `TS_SOCKET` env var if path differs.

### 3. `tailscale serve` requires bind=loopback
Gateway enforces at `src/gateway/server-runtime-config.ts:130-131`:
```
tailscale serve/funnel requires gateway bind=loopback (127.0.0.1)
```
**With shared network namespace** (`network_mode: "service:tailscale"`): loopback is shared, works naturally.
**With separate networks**: `tailscale serve` can't reach gateway's loopback — need shared netns or target gateway IP + add sidecar IP to `trustedProxies`.

### 4. Tailscale auth and `allowTailscale`
At `src/gateway/auth.ts:277-279`, Tailscale auth auto-enables when `tailscaleMode === "serve"` (unless mode is `password` or `trusted-proxy`).
Since we use `trusted-proxy` mode, this is bypassed — no issue.

Trusted-proxy reads `tailscale-user-login` headers (`src/gateway/auth.ts:151`). Tailscale serve injects these automatically when proxying — works as long as sidecar's `tailscale serve` targets the gateway listener.

### 5. Whois verification needs CLI + socket
Even with trusted-proxy, if `allowTailscale` were enabled, gateway calls `tailscale whois --json <clientIP>`.
Since we use `trusted-proxy` with custom `userHeader`, whois path is NOT used — safe.

### 6. Tailscale state persistence
Sidecar needs persistent `/var/lib/tailscale/` to keep node key across restarts.
Mount a volume or set `TS_STATE_DIR`.

### 7. Network namespace: shared vs separate
**Shared (recommended):** `network_mode: "service:tailscale-sidecar"` — gateway and sidecar share loopback. Simplest.
**Separate:** Need to add sidecar IP to `trustedProxies`, and `tailscale serve` must target gateway container IP.

### 8. Envoy still needed for egress
Sidecar only solves ingress (Tailscale serve/funnel). Envoy still needed for outbound (API calls, Discord WebSocket, etc.).
**Can remove:** All DERP hostname entries and UDP relay mappings from Envoy config.
**Keep:** API provider hostnames, Discord, npm, GitHub, Let's Encrypt, etc.

## Recommended Architecture
```
┌─────────────────────────────────────┐
│  Shared network namespace           │
│  ┌─────────────┐  ┌──────────────┐  │
│  │  tailscale   │  │   openclaw   │  │
│  │  sidecar     │  │   gateway    │  │
│  │  (tailscaled)│  │  (bind:loop) │  │
│  │  port 443 ←──┼──→ port 18789  │  │
│  └─────────────┘  └──────────────┘  │
│         ↕ (direct internet)         │
└─────────────────────────────────────┘
         ↕ (egress only)
┌─────────────────────────────────────┐
│  Envoy (SNI whitelist proxy)        │
│  - API providers, Discord, npm...   │
│  - NO more DERP entries needed      │
└─────────────────────────────────────┘
```

## Config Changes for Pulumi
```bash
# Keep existing:
config set gateway.bind loopback
config set gateway.auth.mode trusted-proxy
config set gateway.trustedProxies '["127.0.0.1"]'
config set gateway.auth.trustedProxy.userHeader tailscale-user-login

# Tailscale serve mode still "serve" (sidecar runs it)
# tailscale serve --bg --yes 18789  (run in sidecar, not gateway)
```
