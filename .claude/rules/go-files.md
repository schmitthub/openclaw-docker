---
globs: ["**/*.go"]
---

# Go Development Rules

## Build & Verify
- `go build .` — compile the CLI
- `go vet ./...` — static analysis
- `golangci-lint run --config .golangci.yml` — linting
- `make build` — build with ldflags (version/date injection)

## Module
- Module path: `github.com/schmitthub/openclaw-docker`
- Go 1.25, dependencies: cobra, semver/v3, yaml.v3

## Package Layout
| Package | Purpose |
|---------|---------|
| `internal/cmd` | Cobra commands (root, generate, config, version) |
| `internal/render` | Dockerfile/compose/env/setup.sh/envoy config generation |
| `internal/versions` | npm version resolution, manifest I/O, semver matching |
| `internal/config` | YAML config loading and default template |
| `internal/build` | Build metadata (version/date via ldflags) |
| `internal/update` | GitHub release update checks |
| `internal/testenv` | Isolated filesystem test environments |
| `e2e` | End-to-end generation tests |
| `e2e/harness` | Test harness wrapping testenv + Cobra execution |

## Conventions
- Config precedence: flags > env vars (`OPENCLAW_DOCKER_*`) > config file > defaults
- `mergedOptions(cmd)` in `internal/cmd/root.go` resolves the full option set
- Write safety: `confirmWrite()` prompts unless `--dangerous-inline` is set
- Generated Dockerfile uses `ENTRYPOINT ["entrypoint.sh"]` (root → iptables → gosu node) with `CMD` for default arguments
- Generated Dockerfile uses `node:22-bookworm` base
