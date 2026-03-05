---
globs: ["**/Dockerfile*", "**/*.sh", "templates/dockerfile.ts", "templates/entrypoint.ts", "templates/sidecar.ts", "templates/envoy.ts", "templates/serve.ts"]
---

# Docker & Shell Artifact Rules

## Generated Artifacts (rendered by templates/)

Templates are pure TypeScript functions that return strings. They are called at Pulumi plan time
and uploaded to remote hosts via `command.remote.Command` (base64-encoded).

| Template | Output | Purpose |
|----------|--------|---------|
| `templates/dockerfile.ts` | `Dockerfile` | `node:22-bookworm` + openssh-server + gosu + libsecret-tools + pnpm + bun + brew + uv + tailscale + openclaw |
| `templates/entrypoint.ts` | `entrypoint.sh` | Gateway: sshd + gosu node (no iptables) |
| `templates/sidecar.ts` | `sidecar-entrypoint.sh` | Sidecar: iptables REDIRECT + UDP owner-match + exec containerboot |
| `templates/serve.ts` | `serve-config.json` | Tailscale Serve config (HTTPS + SSH forwarding) |
| `templates/envoy.ts` | `envoy.yaml` | Egress-only: transparent TLS proxy (SNI whitelist) |

Artifacts are written to the remote host at `/opt/openclaw-deploy/build/<profile>/` (Dockerfile, entrypoint.sh, sidecar-entrypoint.sh, serve-config.json)
and `/opt/openclaw-deploy/envoy/envoy.yaml` (Envoy config).

## Dockerfile Conventions
- Base: `node:22-bookworm` (matches official OpenClaw Docker pattern)
- Always installs `openssh-server`, `gosu`, and `libsecret-tools`
- Does NOT install `iptables` or `iproute2` (sidecar handles networking)
- Configures sshd: loopback-only, port 2222, root login with empty password, no PAM
- `ssh-keygen -A` generates host keys at build time
- `pnpm` installed via `npm install -g pnpm`; `PNPM_HOME=/home/node/.local/share/pnpm` for global bin dir
- `bun` installed via `curl -fsSL https://bun.sh/install | bash` then **copied** to `/usr/local/bin/bun` (symlinks through `/root/` fail — mode 0700)
- `uv` (Python package manager) installed via official install script as node user
- Tailscale (`tailscale` CLI only) installed via official install script (~20MB, always installed)
- Homebrew (Linuxbrew) installed via official install script as `node` user; available via PATH
- OpenClaw installed via `npm install -g openclaw@<version>`
- `SHARP_IGNORE_GLOBAL_LIBVIPS=1` set during npm install
- `NODE_OPTIONS=--max-old-space-size=2048` during npm install (prevents OOM on low-memory hosts)
- CLI symlink: `ln -sf "$(npm root -g)/openclaw/dist/entry.js" /usr/local/bin/openclaw`
- Optional `OPENCLAW_INSTALL_BROWSER` ARG: bakes Playwright + Chromium + Xvfb (~300MB)
- `COPY entrypoint.sh /usr/local/bin/entrypoint.sh` — root-owned, 0755
- `ENTRYPOINT ["entrypoint.sh"]` runs as root to start sshd, then drops to `node`
- `CMD ["openclaw", "gateway", "--port", "18789"]` (overridden by container command)
- `OPENCLAW_DOCKER_APT_PACKAGES` ARG for optional additional packages via `packages` config
- `OPENCLAW_PREFER_PNPM=1` env var (Bun may fail on ARM/Synology)
- `OPENCLAW_BRIDGE_PORT=18790` and `OPENCLAW_GATEWAY_BIND=loopback` env vars
- No dev tools (zsh, git-delta, hadolint, fzf)
- No ttyd, no filebrowser (replaced by SSH via Tailscale Serve)

## Sidecar Security Model (sidecar-entrypoint.sh)
The Tailscale sidecar container owns the shared network namespace and enforces egress isolation.
All containers (envoy, gateway) share this netns via `network_mode: container:tailscale-<profile>`.

1. Excludes Envoy (uid 101) from REDIRECT via `iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner ${ENVOY_UID:-101} -j RETURN`
2. Excludes root (uid 0) from REDIRECT via `iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner 0 -j RETURN`
3. Processes `OPENCLAW_TCP_MAPPINGS` env var (if set) — per-destination TCP REDIRECT rules for SSH/TCP egress. Each semicolon-delimited entry (`dst|dstPort|envoyPort`) resolves the domain to IP via `getent ahostsv4` and adds a destination-specific iptables REDIRECT rule routing matching traffic to a dedicated Envoy listener port. IP destinations skip resolution. Malformed entries and unresolvable domains emit warnings and are skipped.
4. Catch-all REDIRECT: all remaining outbound TCP (except loopback) to Envoy's transparent TLS proxy listener (localhost:10000)
5. **UDP rules**: ACCEPT Docker DNS (127.0.0.11), ACCEPT root-owned UDP (containerboot/tailscaled), DROP all other UDP. This prevents the `node` user (openclaw) from UDP exfiltration.
6. `exec /usr/local/bin/containerboot "$@"` — hands off to Tailscale's official entrypoint, which handles auth, state, serve config, and keeps the container alive.

Key differences from previous architecture:
- `REDIRECT` replaces `DNAT` (shared netns — everything is localhost)
- Owner-match exclusions replace subnet-based skip rules
- No FILTER table (NAT REDIRECT + UDP DROP is sufficient)
- No Envoy IP resolution, no subnet derivation, no default route
- No manual `tailscaled` — `exec containerboot` handles everything
- No `wait $TAILSCALED_PID` — containerboot is PID 1

## Gateway Entrypoint (entrypoint.sh)
The simplified gateway entrypoint handles only application-level setup:
1. Fixes config dir permissions + git safe.directory for linuxbrew
2. Starts sshd (for Tailscale Serve TCP forwarding — SSH access)
3. Drops to `node` user via `exec gosu node "$@"`

No iptables, no routing, no tailscaled, no socket wait, no web tools — all networking is handled by the sidecar, SSH replaces ttyd/filebrowser.

## Tailscale Serve Config (serve-config.json)
Rendered by `renderServeConfig(gatewayPort, sshdPort)`:
- `TCP.443.HTTPS: true` — HTTPS termination by Tailscale
- `TCP.22.TCPForward: 127.0.0.1:<sshdPort>` — SSH forwarding to gateway sshd
- `Web.${TS_CERT_DOMAIN}:443.Handlers./` — proxy to `http://127.0.0.1:<gatewayPort>`
- `AllowFunnel.${TS_CERT_DOMAIN}:443: false`

`${TS_CERT_DOMAIN}` is a containerboot variable that substitutes the node's Tailscale FQDN at runtime.
Deployed via `TS_SERVE_CONFIG` env var on the sidecar — no dynamic `tailscale serve` CLI calls.

## Envoy Configuration (egress-only)
- **No ingress listener** — Tailscale handles all ingress (Serve)
- **Egress listener (:10000)**: Transparent TLS proxy with SNI-based domain whitelist. All outbound TCP from gateway is REDIRECTed here by iptables.
- **No DNS listener** — DNS is handled by Docker's embedded DNS + Cloudflare via sidecar `dns:` config
- TLS Inspector listener filter reads SNI from ClientHello without terminating TLS (no MITM)
- Domain ACL via `filter_chain_match.server_names` matching TLS SNI
- Non-TLS, non-mapped traffic (plain HTTP, unmapped raw TCP) is denied — no SNI to inspect and no port mapping
- `sni_dynamic_forward_proxy` resolves whitelisted domains via DNS and forwards to port 443
- DNS lookup family: `V4_PREFERRED` — prefers IPv4 but falls back to IPv6 if no A record exists (Docker networks are IPv4-only by default; `AUTO` causes failures on IPv6-first resolvers)
- `deny_cluster` (STATIC, no endpoints) immediately resets non-whitelisted connections
- **SSH/TCP listeners (:10001+)**: Per-rule dedicated `tcp_proxy` listeners for SSH/TCP egress rules. Each rule gets a sequential port starting from `ENVOY_TCP_PORT_BASE` (10001). Uses `STRICT_DNS` clusters for domain destinations and `STATIC` clusters for IP destinations.
- No UDP listeners — UDP is handled by sidecar iptables owner-match (root only)
- Hardcoded domains defined in `config/domains.ts` (`*.tailscale.com` wildcard + `*.api.letsencrypt.org`), user rules additive via `egressPolicy` config
- Warnings emitted for CIDR SSH/TCP destinations (not supported) and missing port on SSH/TCP rules

## Docker Container Conventions
- Per gateway: 1 bridge network + 3 containers (sidecar + envoy + gateway), all sharing sidecar's netns
- Bridge network: `openclaw-net-<profile>` (NOT `internal: true` — sidecar needs internet)
- **Tailscale sidecar** (`tailscale-<profile>`): uses `tailscale/tailscale:v1.94.2` image, `capabilities.adds: [NET_ADMIN]`, `dns: [1.1.1.2, 1.0.0.2]`, runs on bridge network. Owns the shared network namespace. Env: `TS_AUTHKEY`, `TS_STATE_DIR`, `TS_USERSPACE=false`, `TS_SERVE_CONFIG`, `TS_ENABLE_HEALTH_CHECK=true`, `ENVOY_UID=101`, `OPENCLAW_TCP_MAPPINGS`. Devices: `/dev/net/tun`. Healthcheck: `wget -q --spider http://localhost:9002/healthz`.
- **Envoy container** (`envoy-<profile>`): `network_mode: container:tailscale-<profile>`. No `networksAdvanced`, no `dns` (inherited). Env: `ENVOY_UID=101`. Labels: `openclaw.config-hash` (triggers replacement on config change). Volumes: envoy.yaml, CA cert, MITM certs.
- **Gateway container** (`openclaw-<profile>`): `network_mode: container:tailscale-<profile>` (shared netns). No `CAP_NET_ADMIN`, no `dns` (inherited), no `networksAdvanced` (mutually exclusive with networkMode).
- Gateway has `init: true`, `restart: unless-stopped`
- No `HTTP_PROXY`/`HTTPS_PROXY` env vars — iptables REDIRECT in sidecar handles all routing transparently
- `OPENCLAW_TCP_MAPPINGS` env var (optional): semicolon-delimited `dst|dstPort|envoyPort` entries for SSH/TCP egress. Set on the **sidecar** container. Processed by sidecar-entrypoint.sh to create per-destination iptables REDIRECT rules.
- `OPENCLAW_GATEWAY_TOKEN` env var (secret): gateway auth token, always set on gateway container.
- `TS_AUTHKEY` env var (secret): Tailscale auth key, set on the **sidecar** container. Consumed by containerboot for authentication.
- Tailscale state volume: `${dataDir}/tailscale → /var/lib/tailscale` (mounted on sidecar, persists auth across restarts)
- Gateway config is written to the shared volume *before* the gateway container starts via an ephemeral CLI container (`docker run --rm --network none --user node --entrypoint /bin/sh -v ${dataDir}/config:/home/node/.openclaw <image> -c "openclaw config set ..."`). This avoids crash-loops from the gateway requiring config (especially `gateway.mode=local`) before it will start.
- Init container runs commands in order: (1) required config set (security-critical), (2) user `configSet` entries, (3) user `setupCommands` (OpenClaw subcommands, auto-prefixed with `openclaw `, e.g. `models set`, `onboard`).
- `gatewaySecretEnv-<profile>` (secret, optional): JSON `{"KEY":"value"}` map. Each entry becomes a `-e KEY='VALUE'` flag on the init container (for `setupCommands` that reference `$KEY`) and an env var on the main gateway container (for runtime use). Set via `pulumi config set --secret gatewaySecretEnv-<profile> '{"OPENROUTER_API_KEY":"sk-or-..."}'`. Init container uses `logging: "none"` and `additionalSecretOutputs` to suppress secrets in logs.
- Per-gateway Docker image: `openclaw-gateway-<profile>:<version>`

## Template Code Conventions
- Templates live in `templates/` and are **pure functions** returning strings
- No side effects, no I/O, no Pulumi dependencies
- `renderDockerfile(opts)` — parameterized by version, packages, installBrowser, ports
- `renderEntrypoint()` — static content (no parameters, simplified: sshd + gosu)
- `renderSidecarEntrypoint()` — static content (iptables REDIRECT + UDP owner-match + exec containerboot)
- `renderServeConfig(gatewayPort, sshdPort)` — Tailscale Serve JSON config
- `renderEnvoyConfig(userRules)` — returns `{ yaml, warnings }`, merges with hardcoded domains
- All constants imported from `config/defaults.ts`
