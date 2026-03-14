# Composite Sidecar Image — Future Feature

## Problem
Each gateway currently runs 3 separate containers sharing a network namespace:
- `tailscale-<profile>` (Tailscale sidecar — owns netns, iptables, containerboot)
- `envoy-<profile>` (egress proxy — SNI-based domain whitelist)
- `openclaw-gateway-<profile>` (gateway — runs CoreDNS + filebrowser + openclaw)

This adds complexity: `network_mode: container:` wiring, 3 Pulumi components per gateway, independent health checks, and restart cascade logic in `ocm`.

## Proposal
Combine Tailscale, Envoy, and CoreDNS into a single sidecar image using multi-stage builds. Tailscale is the final runtime stage (Alpine 3.22). Envoy and CoreDNS binaries are copied in from their official images.

```dockerfile
FROM envoyproxy/envoy:v1.33-latest AS envoy
FROM coredns/coredns:1.14.2 AS coredns
FROM tailscale/tailscale:v1.94.2

COPY --from=envoy /usr/local/bin/envoy /usr/local/bin/envoy
COPY --from=coredns /coredns /usr/local/bin/coredns
```

Entrypoint flow: iptables rules → start CoreDNS (root, port 5300) → start Envoy (uid 101) → exec containerboot.

## Key Considerations

### Distro compatibility
- Tailscale: Alpine 3.22 (musl libc)
- Envoy: Ubuntu 22.04 (glibc) — **may not run on Alpine musl without static linking**
- CoreDNS: distroless/static — **static binary, works anywhere**

**Critical risk:** Envoy is dynamically linked against glibc. Must verify it runs on Alpine, or:
- Use envoy's `-distroless` variant as copy source
- Statically compile envoy
- Switch final stage to a glibc-based image (e.g. `debian:bookworm-slim` + install tailscale)
- Use `ldd` to check envoy's dynamic dependencies

### Architecture
- UID separation still works: root (tailscale/containerboot), 101 (envoy), 1000 (node in gateway)
- CoreDNS moves from the gateway container to the sidecar (architectural change — update entrypoint.sh, remove CoreDNS from gateway Dockerfile)
- Process supervision: shell entrypoint manages coredns + envoy as background processes, containerboot as PID 1 via exec
- Health checks: need composite health (all 3 services) or separate health endpoints

### Benefits
- 2 containers per gateway instead of 3
- No `network_mode: container:` for envoy (it's in the same container)
- Simpler Pulumi component graph (remove EnvoyProxy component)
- Faster startup (one container boot instead of three)
- Simpler `ocm restart` logic

### Tradeoffs
- Custom image to build and maintain (no longer using official tailscale/envoy images directly)
- Lose independent container restarts (`ocm restart envoy` without touching tailscale)
- Harder to upgrade individual components (envoy or tailscale version bump requires rebuild)
- Must pin and track 3 upstream versions in one image
- More complex entrypoint (process management for 3 services)

### Init Container Consolidation (related)
Currently each `setupCommand` is a separate `docker run --rm` = separate cold start. With 30+ commands, this is extremely slow (~2min per command on node:24). Should consolidate into a single `docker run --rm` that executes all setup commands in one shell script. The per-command Pulumi resource granularity provides no benefit because:
1. Commands are chained with `dependsOn` — if one triggers, all downstream cascade
2. The `environment` object is a single blob — any env var change triggers all commands
3. N cold starts vs 1 cold start is a massive performance difference

### Buildkit Container Lifecycle (related)
`@pulumi/docker-build` leaves buildkit containers running after builds (pulumi/pulumi-docker-build#65). Current cleanup uses `docker stop` post-build but doesn't handle cancellations. Pulumi has no exit hooks. Named builder (`builder: { name: "openclaw-builder" }`) on the `docker_build.Image` resource would give deterministic container/volume names and prevent orphaned builders with random names. The provider reuses existing builders if the registration in `~/.docker/buildx/instances/` is intact.
