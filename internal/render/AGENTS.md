# Package: `internal/render`

Generates all deployment artifacts. Files: `render.go` and `ca.go`.

## Functions

| Function | Output |
|----------|--------|
| `Generate(opts Options)` | Orchestrates all writes |
| `dockerfileFor(opts)` | Dockerfile content (`fmt.Sprintf` template) |
| `composeFileContent()` | compose.yaml (string-joined lines) |
| `openClawEnvFileContent(opts)` | .env.openclaw (`fmt.Sprintf` template) |
| `setupScriptContent(opts)` | setup.sh (`fmt.Sprintf` template) |
| `writeComposeArtifacts(opts)` | Writes compose.yaml + .env.openclaw |
| `writeSetupScript(opts)` | Writes setup.sh with 0755 perms |
| `squidDockerfileContent()` | Dockerfile.squid content |
| `squidConfContent(opts)` | squid.conf with SSL bump + domain ACLs |
| `openClawJSONContent(opts)` | openclaw.json with gateway config |
| `generateCA(opts)` | CA cert+key generation (in `ca.go`) |

## Options Struct

`render.Options` carries all configuration from CLI into generation.
`ConfirmWrite func(path string) error` — write safety callback (nil in tests).
`SquidAllowedDomains string` — comma-separated domains for squid whitelist.

## Design Decisions

- All content is built via `fmt.Sprintf` with Go string templates (not `text/template`)
- Compose uses `build:` directive, never `image:` tag
- Dockerfile uses `node` user from `node:22-bookworm` base
- No ENTRYPOINT — CMD only
- setup.sh must be Bash 3.2 compatible (macOS)
- Defaults for config/workspace dirs use `/home/node/.openclaw`
- CA cert+key preserved across re-runs (idempotent)
- Squid uses `squid-openssl` package for SSL bump support
- `openclaw.ai` always included in squid domain whitelist

## Generated Output

```
<output>/
├── Dockerfile         # 0644, lean node:22-bookworm
├── Dockerfile.squid   # 0644, squid-openssl + ssl_db init
├── compose.yaml       # 0644, squid proxy + gateway
├── .env.openclaw      # 0644, runtime env vars + proxy config
├── setup.sh           # 0755, token gen + openclaw.json seeding + compose up
├── squid.conf         # 0644, SSL bump + egress whitelist ACLs
├── openclaw.json      # 0644, pre-seeded gateway config
├── ca-cert.pem        # 0644, self-signed CA cert (mounted into both containers)
└── ca-key.pem         # 0600, CA private key (mounted into squid only)
```
