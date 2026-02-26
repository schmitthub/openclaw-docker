# openclaw-docker

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
- Resolved versions are written to `versions.json`.

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

# optional explicit commands
go run . resolve --version latest
go run . render --versions-file versions.json --output ./dockerfiles
```

### Output behavior

- `--output|-o` controls Dockerfile output root.
- If omitted, output defaults to current working directory.

### Config behavior

- Config file path must be passed explicitly using `--config` or `-f`.
- No automatic config discovery is performed.
- Flags always override values from config.

## Repository structure

- `main.go`: CLI entrypoint
- `internal/cmd`: Cobra root/commands
- `internal/config`: YAML config loading
- `internal/versions`: npm resolution + manifest IO
- `internal/render`: Dockerfile generation + cleanup
- `build/templates/docker-entrypoint.sh`: runtime entrypoint behavior
- `build/templates/docker-init-firewall.sh`: optional firewall setup helper
- `dockerfiles/`: one generated output location (when used as `--output`)

