# Tailscale Sidecar + SNI Wildcard + Remove UDP Infrastructure

## End Goal
Replace the per-destination UDP listener model with a Tailscale sidecar container that owns the network namespace. Use `*.tailscale.com` wildcard instead of enumerated DERP hostnames. Block UDP exfiltration via iptables `--uid-owner` matching.

## Plan File
`/Users/andrew/.claude/plans/lovely-squishing-lampson.md`

## Branch
`fix/tailscale-sni-whitelist`

## Status: ALL IMPLEMENTATION COMPLETE — NEEDS COMMIT + PR

## Completed Steps

### 1. [DONE] config/domains.ts — Wildcard + remove UDP
- Replaced DERP enumeration (28 domains) with `*.tailscale.com` wildcard
- Removed `TAILSCALE_UDP_DOMAINS` export
- `HARDCODED_EGRESS_RULES` no longer includes UDP domains

### 2. [DONE] Remove UDP types and defaults
- Removed `UdpPortMapping` interface from `config/types.ts`
- Removed `ENVOY_UDP_PORT_BASE` from `config/defaults.ts`
- Added `TAILSCALE_IMAGE = "tailscale/tailscale:v1.82.5"` and `TAILSCALE_SOCKET_DIR` to defaults
- Updated `templates/index.ts` exports

### 3. [DONE] Remove UDP from templates/envoy.ts
- Removed `renderUdpListener()`, `renderUdpCluster()`, UDP mappings array, `case "udp"`, UDP sections from YAML

### 4. [DONE] Create templates/sidecar.ts
- New `renderSidecarEntrypoint()` — iptables NAT + FILTER + UDP owner-match + tailscaled

### 5. [DONE] Simplify templates/entrypoint.ts
- No iptables, no routing, no tailscaled
- Socket wait + web tools + Tailscale Serve + gosu

### 6. [DONE] Update components (envoy.ts, gateway.ts, index.ts)
- `envoy.ts`: removed `udpPortMappings` output
- `gateway.ts`: added sidecar container, shared netns, removed `CAP_NET_ADMIN` from gateway
- `index.ts`: removed `udpPortMappings` pass-through

### 7. [DONE] Update all tests (260 tests pass)
- config.test.ts, templates.test.ts, envoy.test.ts, envoy-component.test.ts all updated

### 8. [DONE] Update docs (AGENTS.md, MEMORY.md, .claude/rules/docker-and-shell.md)
- Threat model updated for sidecar architecture
- Key invariants updated (UDP owner-match, sidecar model)
- Egress domain whitelist updated (wildcard)

### 9. [DONE] Verification — tsc + vitest pass
- `npx tsc --noEmit` — clean
- `npx vitest run` — 260/260 pass

### 10. [DONE] Update reference stack for sidecar model
- `reference/envoy.yaml`: replaced DERP enumeration with wildcard, removed all UDP listeners/clusters
- `reference/sidecar-entrypoint.sh`: NEW — iptables + UDP owner-match + tailscaled
- `reference/entrypoint.sh`: simplified — socket wait + web tools + gosu
- `reference/docker-compose.yml`: added `tailscale-sidecar` service, gateway uses `network_mode: container:tailscale-sidecar`
- `reference/Dockerfile`: removed iptables/iproute2, simplified CMD
- `reference/setup.sh`: updated Tailscale status query to target sidecar
- `reference/AGENTS.md`: updated docs

## Remaining TODOs

### 11. [ ] Commit all changes
- 21 modified files + 2 new files (templates/sidecar.ts, reference/sidecar-entrypoint.sh)
- Branch: `fix/tailscale-sni-whitelist`

### 12. [ ] Create PR
- Target: main
- Title suggestion: "feat: Tailscale sidecar model + SNI wildcard + remove UDP infrastructure"

### 13. [ ] (Optional) Local Docker Compose smoke test
- User asked to "update the reference stack to test locally with openclaw in docker compose" — the files are updated but not tested yet
- `cd reference && ./setup.sh` to validate

## Key Architecture Decisions
- Sidecar uses `tailscale/tailscale:v1.82.5` official image (has iptables, iproute2, tailscaled)
- Gateway image no longer needs iptables/iproute2
- `*.tailscale.com` wildcard is safe because UDP exfiltration is blocked by owner-match
- Shared socket volume (`tailscale-socket`) for sidecar↔gateway communication
- Sidecar healthcheck: `tailscale status --json`

## IMPERATIVE
Always check with the user before proceeding with the next todo item. If all work is done, ask the user if they want to delete this memory.
