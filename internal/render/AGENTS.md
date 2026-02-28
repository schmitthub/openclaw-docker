# Package: `internal/render`

Generates all deployment artifacts. Files: `render.go` and `ca.go`.

## Functions

| Function | Output |
|----------|--------|
| `Generate(opts Options)` | Orchestrates all writes |
| `dockerfileFor(opts)` | Dockerfile content (`fmt.Sprintf` template) |
| `entrypointContent()` | entrypoint.sh: default route + iptables rules + gosu drop to node |
| `composeFileContent(opts)` | compose.yaml (string-joined lines, 3 services) |
| `openClawEnvFileContent(opts)` | .env.openclaw (`fmt.Sprintf` template) |
| `setupScriptContent(opts)` | setup.sh (`fmt.Sprintf` template) |
| `writeComposeArtifacts(opts)` | Writes compose.yaml + .env.openclaw |
| `writeSetupScript(opts)` | Writes setup.sh with 0755 perms |
| `writeEntrypoint(opts)` | Writes entrypoint.sh with 0755 perms |
| `envoyConfigContent(opts)` | envoy.yaml with ingress + egress + DNS listeners |
| `cliWrapperContent()` | openclaw CLI wrapper script (docker compose passthrough) |
| `writeCLIWrapper(opts)` | Writes openclaw wrapper with 0755 perms |
| `generateTLSCert(opts)` | Self-signed TLS cert for Envoy ingress (in `ca.go`) |

## Options Struct

`render.Options` carries all configuration from CLI into generation.
`ConfirmWrite func(path string) error` — write safety callback (nil in tests).
`AllowedDomains string` — comma-separated domains for Envoy egress whitelist.

## Design Decisions

- All content is built via `fmt.Sprintf` with Go string templates (not `text/template`)
- Compose uses `build:` directive for gateway/CLI, stock `envoyproxy/envoy` image for Envoy
- 3 compose services: `envoy`, `openclaw-gateway`, `openclaw-cli`
- Gateway has explicit `command: ["openclaw", "gateway", "--bind", "lan", "--port", "<port>"]` for LAN binding
- Gateway has `cap_add: [NET_ADMIN]` for root-owned iptables + routing setup in entrypoint
- Gateway and CLI services use `dns: [172.28.0.2]` (Envoy's static IP for DNS forwarding)
- Gateway has `init: true`, `restart: unless-stopped`, `HOME`/`TERM` env vars
- CLI service overrides `entrypoint: ["openclaw"]` with `stdin_open`, `tty`, `init`, `BROWSER: echo`, `depends_on: [envoy]`
- Entrypoint runs as root: adds default route via Envoy, restores DOCKER_OUTPUT chain, sets iptables (NAT DNAT + FILTER DROP), then `gosu node`
- Default route via Envoy required because `internal: true` has no gateway — kernel rejects external IPs before iptables can DNAT
- DOCKER_OUTPUT chain jump restored after flushing nat OUTPUT — Docker DNS DNAT depends on it
- iptables NAT DNAT transparently redirects all outbound TCP to Envoy — no app proxy awareness needed
- Envoy is the unified ingress/egress/DNS proxy — publishes port 443, gateway has no published ports
- Envoy has static IP `172.28.0.2` on internal network (IPAM subnet `172.28.0.0/24`)
- Envoy ingress forwards client IP via `use_remote_address: true` and `xff_num_trusted_hops: 0`
- Envoy egress listener on port 10000 uses TLS Inspector + SNI-based domain filtering
- Envoy DNS listener on port 53 UDP forwards to Cloudflare malware-blocking resolvers (1.1.1.2 / 1.0.0.2)
- Egress whitelist: infrastructure (`clawhub.com`, `registry.npmjs.org`) + AI providers always hardcoded, `--allowed-domains` additive with dedup
- No SSL bump / MITM — TLS is end-to-end, domain filtering via TLS SNI from ClientHello
- No `HTTP_PROXY`/`HTTPS_PROXY` env vars — iptables DNAT is the transparent proxy mechanism
- The security boundary is: Docker `internal: true` network + root-owned iptables DNAT + Envoy SNI whitelist + malware-blocking DNS
- Dockerfile installs `iptables`, `iproute2`, `gosu`, `pnpm` (corepack), `bun` (install script to /usr/local), and OpenClaw (npm)
- setup.sh mirrors official docker-setup.sh: onboarding, CLI-based config management, no pre-generated openclaw.json
- setup.sh must be Bash 3.2 compatible (macOS)
- Defaults for config/workspace dirs use `/home/node/.openclaw`
- TLS cert preserved across re-runs (idempotent)

## Generated Output

```
<output>/
├── compose/
│   ├── envoy/
│   │   ├── envoy.yaml          # 0644, ingress+egress+DNS proxy config
│   │   ├── server-cert.pem     # 0644, self-signed TLS cert
│   │   └── server-key.pem      # 0600, TLS key
│   └── openclaw/
│       ├── Dockerfile           # 0644, node:22-bookworm + iptables + iproute2 + gosu + pnpm + bun
│       └── entrypoint.sh        # 0755, default route + iptables setup + drop to node
├── compose.yaml                 # 0644, envoy + gateway + cli
├── .env.openclaw                # 0644, runtime env vars (token, ports, bind)
├── setup.sh                     # 0755, build, onboard, configure, start
├── openclaw                     # 0755, CLI wrapper (docker compose passthrough)
└── manifest.json                # 0644, resolved version metadata
```
