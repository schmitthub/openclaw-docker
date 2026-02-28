# Package: `internal/render`

Generates all deployment artifacts. Files: `render.go` and `ca.go`.

## Functions

| Function | Output |
|----------|--------|
| `Generate(opts Options)` | Orchestrates all writes |
| `dockerfileFor(opts)` | Dockerfile content (`fmt.Sprintf` template) |
| `entrypointContent()` | entrypoint.sh: iptables rules + gosu drop to node |
| `composeFileContent(opts)` | compose.yaml (string-joined lines) |
| `openClawEnvFileContent(opts)` | .env.openclaw (`fmt.Sprintf` template) |
| `setupScriptContent(opts)` | setup.sh (`fmt.Sprintf` template) |
| `writeComposeArtifacts(opts)` | Writes compose.yaml + .env.openclaw |
| `writeSetupScript(opts)` | Writes setup.sh with 0755 perms |
| `writeEntrypoint(opts)` | Writes entrypoint.sh with 0755 perms |
| `envoyConfigContent(opts)` | envoy.yaml with ingress + egress listeners |
| `generateTLSCert(opts)` | Self-signed TLS cert for Envoy ingress (in `ca.go`) |

## Options Struct

`render.Options` carries all configuration from CLI into generation.
`ConfirmWrite func(path string) error` — write safety callback (nil in tests).
`AllowedDomains string` — comma-separated domains for Envoy egress whitelist.

## Design Decisions

- All content is built via `fmt.Sprintf` with Go string templates (not `text/template`)
- Compose uses `build:` directive for gateway, stock `envoyproxy/envoy` image for Envoy
- Gateway has `cap_add: [NET_ADMIN]` for root-owned iptables setup in entrypoint
- Entrypoint runs as root, sets iptables (OUTPUT DROP + allow Envoy only), then `gosu node`
- Envoy is the unified ingress/egress proxy — publishes port 443, gateway has no published ports
- Envoy egress listener on port 10000 handles HTTP CONNECT with domain ACL
- No SSL bump / MITM — TLS is end-to-end, domain filtering via CONNECT authority
- `HTTP_PROXY`/`HTTPS_PROXY` env vars are convenience for proxy-aware tools, not the security boundary
- The security boundary is: Docker `internal: true` network + root-owned iptables rules
- setup.sh must be Bash 3.2 compatible (macOS)
- Defaults for config/workspace dirs use `/home/node/.openclaw`
- TLS cert preserved across re-runs (idempotent)
- `openclaw.ai` always included in Envoy domain whitelist

## Generated Output

```
<output>/
├── compose/
│   ├── envoy/
│   │   ├── envoy.yaml          # 0644, ingress+egress proxy config
│   │   ├── server-cert.pem     # 0644, self-signed TLS cert
│   │   └── server-key.pem      # 0600, TLS key
│   └── openclaw/
│       ├── Dockerfile           # 0644, node:22-bookworm + iptables + gosu
│       └── entrypoint.sh        # 0755, iptables setup + drop to node
├── compose.yaml                 # 0644, envoy + gateway + cli
├── .env.openclaw                # 0644, runtime env vars + proxy config
├── setup.sh                     # 0755, token gen, onboarding, compose up
└── manifest.json                # 0644, resolved version metadata
```
