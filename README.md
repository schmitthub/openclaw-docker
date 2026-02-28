# openclaw-docker

![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![Cobra 1.8.1](https://img.shields.io/badge/Cobra-1.8.1-38BDAE)
![Semver 3.3.0](https://img.shields.io/badge/Masterminds%2Fsemver-3.3.0-5A67D8)
![YAML v3.0.1](https://img.shields.io/badge/yaml.v3-3.0.1-CB171E)
![GoReleaser v2](https://img.shields.io/badge/GoReleaser-v2-00ADD8)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-supported-6E56CF)](https://docs.openclaw.ai/install/docker)
![macOS](https://img.shields.io/badge/macOS-supported-000000?logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-supported-FCC624?logo=linux&logoColor=black)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/schmitthub/openclaw-docker)

CLI for generating OpenClaw Docker deployment artifacts.

## Overview

- Standalone Go CLI (Cobra) with entrypoint in `main.go`.
- Resolves a single OpenClaw version from npm package `openclaw` (dist-tag or semver partial).
- Generates a lean Dockerfile based on `node:22-bookworm` at `<output>/Dockerfile`.
- Generates `compose.yaml`, `.env.openclaw`, and `setup.sh` at the output root.
- Generated compose uses a `squid` proxy and an internal-only app network for egress control.
- Installs OpenClaw in images via `curl -fsSL https://openclaw.ai/install.sh | bash`.
- Includes CLI build metadata and a `version` command.
- Checks for newer GitHub releases and shows upgrade hints after command execution.
- Uses defensive write prompts by default for overwrite operations.

## Output Structure

```
<output>/
├── Dockerfile         # Lean node:22-bookworm based Dockerfile
├── compose.yaml       # Docker Compose with squid proxy
├── .env.openclaw      # Environment variables for compose
└── setup.sh           # Helper script for build/pull, token gen, compose up
```

## Scope

- In scope: generating deployment artifacts that make OpenClaw easy to launch securely.
- Out of scope: registry publishing workflows (Docker Hub/GHCR push automation).

## Version and tag resolution

- The CLI queries npm package metadata from `openclaw`.
- Accepts a single `--openclaw-version` value (dist-tag like `latest` or semver partial like `2026.2`).
- Dist-tags are resolved first; if not a dist-tag, semver matching is used.
- Resolved version metadata is written to `$XDG_CACHE_HOME/openclaw-docker/versions.json`.
- If `XDG_CACHE_HOME` is not set, the fallback path is `~/.cache/openclaw-docker/versions.json`.

## Common commands

```bash
# generate all artifacts (default version: latest)
go run .

# print CLI version
go run . --version

# explicit OpenClaw version
go run . --openclaw-version latest

# explicit output directory
go run . --openclaw-version latest --output ./openclaw-deploy

# config file (explicit path only)
go run . --config ./config.yaml

# skip per-write prompts (non-interactive/CI)
go run . --dangerous-inline --openclaw-version latest

# bake setup defaults into generated Dockerfile
go run . --dangerous-inline \
  --openclaw-version latest \
  --docker-apt-packages "git-lfs ripgrep" \
  --openclaw-config-dir /home/node/.openclaw \
  --openclaw-workspace-dir /home/node/.openclaw/workspace \
  --openclaw-gateway-port 18789 \
  --openclaw-bridge-port 18790 \
  --openclaw-gateway-bind lan \
  --openclaw-image openclaw:local

# explicit subcommands
go run . resolve --openclaw-version latest
go run . render --versions-file "$XDG_CACHE_HOME/openclaw-docker/versions.json" --output ./openclaw-deploy
```

### Local PATH setup (direnv)

If you use `direnv`, bootstrap local CLI binary path with:

```bash
cp .envrc.example .envrc
direnv allow
```

This prepends `./bin` to `PATH` for this repository.

### Install script (Linux)

```bash
# local install/update (default: ~/.local/bin)
curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash

# global install/update (/usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash -s -- --global

# install a specific version
curl -fsSL https://raw.githubusercontent.com/schmitthub/openclaw-docker/main/scripts/install.sh | bash -s -- --version v0.1.0
```

### Update checks

- The CLI always checks `schmitthub/openclaw-docker` GitHub releases with a cached interval and prints a concise upgrade hint when a newer CLI version exists.

### Output behavior

- `--output|-o` controls output root.
- If omitted, output defaults to `./openclaw-deploy`.
- Generation is additive and overwrite-only; existing generated files can be replaced, but directories are not deleted.
- Output root includes `Dockerfile`, `compose.yaml`, `.env.openclaw`, and `setup.sh`.
- Generated `.env.openclaw` includes proxy defaults (`OPENCLAW_HTTP_PROXY`, `OPENCLAW_HTTPS_PROXY`, `OPENCLAW_NO_PROXY`).
- Generated `compose.yaml` attaches `openclaw-gateway` and `openclaw-cli` to an internal-only network and routes egress through `squid`.
- `setup.sh` handles image build/pull, gateway token generation, and compose orchestration.
- `--cleanup` prints a defensive warning with the target directory and still does not perform deletes.
- By default, only overwrites prompt for confirmation; first-time file creates are written directly.
- Use `--dangerous-inline` to bypass all write prompts (recommended for CI/non-interactive runs).

### Compose usage

- `compose.yaml` expects values from `.env.openclaw`.
- Use the generated `setup.sh` or run Compose manually:

```bash
docker compose --env-file ./.env.openclaw -f ./compose.yaml up -d
docker compose --env-file ./.env.openclaw -f ./compose.yaml down
```

### Config behavior

- Config file path must be passed explicitly using `--config` or `-f`.
- No automatic config discovery is performed.
- Precedence is: `flags > environment variables > config file > defaults`.
- Environment variable overrides use the `OPENCLAW_DOCKER_` prefix (examples):
  - `OPENCLAW_DOCKER_OUTPUT`, `OPENCLAW_DOCKER_VERSIONS_FILE`, `OPENCLAW_DOCKER_VERSION`
  - `OPENCLAW_DOCKER_DEBUG`, `OPENCLAW_DOCKER_CLEANUP`, `OPENCLAW_DOCKER_DANGEROUS_INLINE`
  - `OPENCLAW_DOCKER_OPENCLAW_CONFIG_DIR`, `OPENCLAW_DOCKER_OPENCLAW_WORKSPACE_DIR`
  - `OPENCLAW_DOCKER_OPENCLAW_GATEWAY_PORT`, `OPENCLAW_DOCKER_OPENCLAW_BRIDGE_PORT`, `OPENCLAW_DOCKER_OPENCLAW_GATEWAY_BIND`
  - `OPENCLAW_DOCKER_OPENCLAW_IMAGE`, `OPENCLAW_DOCKER_OPENCLAW_GATEWAY_TOKEN`, `OPENCLAW_DOCKER_OPENCLAW_EXTRA_MOUNTS`, `OPENCLAW_DOCKER_OPENCLAW_HOME_VOLUME`

## Repository structure

- `main.go`: CLI entrypoint
- `internal/cmd`: Cobra root/commands
- `internal/config`: YAML config loading
- `internal/versions`: npm resolution + manifest IO
- `internal/render`: Dockerfile, compose, env, and setup script generation
- `internal/update`: release update checks and local cache state

## Future steps

- Add a lightweight e2e check that validates generated `compose.yaml` with `.env.openclaw`.
- Add docs/examples for operating multiple generated outputs in parallel (custom `--output` per deployment).
