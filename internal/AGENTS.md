# Internal Packages

All packages under `internal/` are private to the CLI binary.

| Package | Purpose |
|---------|---------|
| `cmd/` | Cobra commands and option merging |
| `render/` | Dockerfile, compose, env, setup.sh generation |
| `versions/` | npm version resolution, manifest I/O, semver matching |
| `config/` | YAML config file loading |
| `build/` | Build metadata (version/date via ldflags) |
| `update/` | GitHub release update checks |
| `testenv/` | Isolated test environments |

## Conventions

- Go module: `github.com/schmitthub/openclaw-docker`
- Go 1.25 with cobra, semver/v3, yaml.v3
- Config precedence: flags > env vars (`OPENCLAW_DOCKER_*`) > config file > defaults
- `mergedOptions(cmd)` in `cmd/root.go` resolves the full option set
- Write safety: `confirmWrite()` prompts unless `--dangerous-inline` is set
- Generated Dockerfile uses `node` user from `node:22-bookworm` base, no ENTRYPOINT

## Build & Test

```bash
go build .             # compile
go vet ./...           # static analysis
go test ./...          # all tests
make check             # test + vet + lint
```
