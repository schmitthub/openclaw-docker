# openclaw-docker

![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![Cobra 1.8.1](https://img.shields.io/badge/Cobra-1.8.1-38BDAE)
![Semver 3.3.0](https://img.shields.io/badge/Masterminds%2Fsemver-3.3.0-5A67D8)
![YAML v3.0.1](https://img.shields.io/badge/yaml.v3-3.0.1-CB171E)
![GoReleaser v2](https://img.shields.io/badge/GoReleaser-v2-00ADD8)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-supported-6E56CF)](https://docs.openclaw.ai/install/docker)
![macOS](https://img.shields.io/badge/macOS-supported-000000?logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-supported-FCC624?logo=linux&logoColor=black)

CLI for building custom OpenClaw Dockerfiles across multiple Linux variants and versions.

## Overview

- Standalone Go CLI (Cobra) with entrypoint in `main.go`.
- Resolves OpenClaw versions from npm package `openclaw` (dist-tags first, semver fallback).
- Lets you customize generated outputs via flags and explicit config file input.
- Generates Dockerfiles per version/variant matrix at `<output>/<version>/<variant>/Dockerfile`.
- Focuses on empowering users to run OpenClaw via Docker with secure-by-default images.
- Supports config file input via explicit `--config|-f` only (no discovery), with flags taking precedence.
- Installs OpenClaw in images via:
  `curl -fsSL https://openclaw.ai/install.sh | bash`
- Includes CLI build metadata and a `version` command.
- Checks for newer GitHub releases and shows upgrade hints after command execution.
- Uses defensive write prompts by default for manifest and Dockerfile writes.

## Scope

- In scope: generating and maintaining Dockerfiles that make OpenClaw easy to launch securely.
- Out of scope: registry publishing workflows (for example Docker Hub/GHCR push automation and release pipelines).

## Variants

Current default variants are defined in the CLI defaults and include:

- `trixie`
- `bookworm`
- `alpine3.23`
- `alpine3.22`

## Version and tag resolution

- The CLI queries npm package metadata from `openclaw`.
- Dist-tags (for example `latest`, `beta`) are resolved first.
- If an input is not a dist-tag, semver matching is used.
- Resolved versions are written to `$XDG_CACHE_HOME/openclaw-docker/versions.json`.
- If `XDG_CACHE_HOME` is not set, the fallback path is `~/.cache/openclaw-docker/versions.json`.

## Common commands

```bash
# generate Dockerfiles (default versions: latest)
go run .

# explicit tags/versions
go run . --version latest --version beta

# explicit output directory
go run . --version latest --output ./dockerfiles

# config file (explicit path only)
go run . --config ./config.yaml

# config + flag precedence override
go run . --config ./config.yaml --output ./dockerfiles

# approve each write interactively
go run . --version latest --output ./dockerfiles

# optional explicit commands
go run . version
go run . resolve --version latest
go run . render --versions-file "$XDG_CACHE_HOME/openclaw-docker/versions.json" --output ./dockerfiles

# disable update checks for a command
go run . --no-update-check version

# skip per-write prompts (non-interactive/CI)
go run . --dangerous-inline --version latest

# explicit subcommands in non-interactive mode
go run . --dangerous-inline resolve --version latest
go run . --dangerous-inline render --versions-file "$XDG_CACHE_HOME/openclaw-docker/versions.json" --output ./dockerfiles
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

- By default the CLI checks `schmitthub/openclaw-docker` releases with a cached interval and prints a concise upgrade hint when a newer version exists.
- Disable checks per invocation with `--no-update-check`.

### Output behavior

- `--output|-o` controls Dockerfile output root.
- If omitted, output defaults to `./openclawdockerfiles`.
- Generation is additive and overwrite-only; existing generated files can be replaced, but directories are not deleted.
- `--cleanup` prints a defensive warning with the target directory and still does not perform deletes.
- By default, each manifest and Dockerfile write prompts for confirmation.
- Use `--dangerous-inline` to bypass all write prompts (recommended for CI/non-interactive runs).

### Config behavior

- Config file path must be passed explicitly using `--config` or `-f`.
- No automatic config discovery is performed.
- Flags always override values from config.

## Repository structure

- `main.go`: CLI entrypoint
- `internal/cmd`: Cobra root/commands
- `internal/config`: YAML config loading
- `internal/versions`: npm resolution + manifest IO
- `internal/render`: additive Dockerfile generation (overwrite-only)
- `internal/update`: release update checks and local cache state
- `build/templates/docker-entrypoint.sh`: runtime entrypoint behavior
- `build/templates/docker-init-firewall.sh`: optional firewall setup helper
- `dockerfiles/`: one generated output location (when used as `--output`)

