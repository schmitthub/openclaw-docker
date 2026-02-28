---
globs: ["**/*.yaml", "**/*.yml", "internal/config/**/*.go", "internal/cmd/root.go"]
---

# Configuration Rules

## Config File Format (YAML)
- Loaded via `--config` / `-f` flag (no auto-discovery)
- Parsed by `internal/config/config.go` (`config.Load(path)`)
- Template written by `openclaw-docker config init --file <path>`

## Config Precedence
flags > env vars (`OPENCLAW_DOCKER_*`) > config file > defaults

## Key Config Fields
| YAML Key | CLI Flag | Env Var | Default |
|----------|----------|---------|---------|
| `version` | `--openclaw-version` | `OPENCLAW_DOCKER_VERSION` | `latest` |
| `output` | `--output` / `-o` | `OPENCLAW_DOCKER_OUTPUT` | `./openclaw-deploy` |
| `versions_file` | `--versions-file` | `OPENCLAW_DOCKER_VERSIONS_FILE` | `$CACHE/openclaw-docker/versions.json` |
| `docker_apt_packages` | `--docker-apt-packages` | `OPENCLAW_DOCKER_APT_PACKAGES` | `""` |
| `openclaw_config_dir` | `--openclaw-config-dir` | `OPENCLAW_DOCKER_OPENCLAW_CONFIG_DIR` | `/home/node/.openclaw` |
| `openclaw_gateway_port` | `--openclaw-gateway-port` | `OPENCLAW_DOCKER_OPENCLAW_GATEWAY_PORT` | `18789` |
| `cleanup` | `--cleanup` | `OPENCLAW_DOCKER_CLEANUP` | `false` |
| `debug` | `--debug` | `OPENCLAW_DOCKER_DEBUG` | `false` |

## Cache Dir Resolution
`defaultVersionsFilePath()` in `internal/cmd/root.go`:
1. `OPENCLAW_DOCKER_CACHE_DIR` env var (used by testenv)
2. `XDG_CACHE_HOME` env var
3. `~/.cache`
Manifest path: `<cache>/openclaw-docker/versions.json`

## mergedOptions Flow
`mergedOptions(cmd)` in `root.go` builds the full option set:
1. Start with hardcoded defaults
2. Overlay config file values (if `--config` provided)
3. Overlay env var overrides (`applyEnvOverrides`)
4. Overlay CLI flag values (only if `cmd.Flags().Changed(...)`)
5. Trim whitespace on all string fields
