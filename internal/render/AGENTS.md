# Package: `internal/render`

Generates all four deployment artifacts. Single file: `render.go`.

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

## Options Struct

`render.Options` carries all configuration from CLI into generation.
`ConfirmWrite func(path string) error` — write safety callback (nil in tests).

## Design Decisions

- All content is built via `fmt.Sprintf` with Go string templates (not `text/template`)
- Compose uses `build:` directive, never `image:` tag
- Dockerfile uses `node` user from `node:22-bookworm` base
- No ENTRYPOINT — CMD only
- setup.sh must be Bash 3.2 compatible (macOS)
- Defaults for config/workspace dirs use `/home/node/.openclaw`

## Generated Output

```
<output>/
├── Dockerfile         # 0644, lean node:22-bookworm
├── compose.yaml       # 0644, squid proxy + gateway + cli
├── .env.openclaw      # 0644, runtime env vars + proxy config
└── setup.sh           # 0755, token gen + compose build + compose up
```
