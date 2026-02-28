# AGENTS.md

## Repository Overview

This repository provides a Go CLI that builds **custom OpenClaw Dockerfiles** for multiple Linux flavors and versions.

Primary goals:
- Keep generated Dockerfiles reproducible and easy to build.
- Support a customizable matrix of Linux variants and version tags.
- Help users launch OpenClaw through Docker with secure-by-default settings.
- Make it straightforward to maintain and update generation logic over time.

## What Agents Should Assume

- The core artifact in this repo is Docker-related build configuration.
- Dockerfiles in `dockerfiles/` are generated artifacts from the CLI.
- The CLI defaults output to `./openclaw-deploy` when `--output` is omitted.
- Version metadata comes from npm package `openclaw` via the Go CLI (`main.go`).
- Changes should prioritize compatibility, determinism, and minimal image complexity.
- Prefer small, focused edits rather than broad refactors.

## Contribution Guidelines for Agents

When modifying this repository:
- Keep distro/version-specific logic explicit and readable.
- Reuse shared patterns across Dockerfiles when practical, but do not over-engineer abstraction.
- Pin versions where stability matters; document why when pinning is non-obvious.
- Avoid introducing unnecessary runtime dependencies.
- Preserve existing naming/tagging conventions for image variants.
- Update templates/scripts first, then regenerate `dockerfiles/`.

## Validation Expectations

Before considering work complete, agents should:
- Run `go run . --version latest --version beta --output ./dockerfiles --versions-file "${XDG_CACHE_HOME:-$HOME/.cache}/openclaw-docker/versions.json"` after CLI/template changes.
- For non-interactive validation, include `--dangerous-inline` to bypass per-write safety prompts.
- Verify Dockerfile syntax and build steps for touched variants.
- Ensure commands are non-interactive and CI-friendly.
- Confirm OpenClaw installation/startup steps still work for the modified target image(s).

## Safety Model

- Generation is additive and overwrite-only; directory deletion is disabled.
- `--cleanup` prints a defensive warning with the target path and does not delete files.
- By default, writes prompt for confirmation; CI and automation should use `--dangerous-inline`.

## Out of Scope (Unless Explicitly Requested)

- Redesigning repository structure.
- Adding unrelated tooling or frameworks.
- Building registry publishing/release automation (for example pushing images to Docker Hub or GHCR).
- Changing release/versioning policy beyond the requested task.

## Editing Style

- Keep docs concise and operational.
- Keep commits scoped to one concern (single distro/version family when possible).
- Prefer clarity over cleverness in shell and Docker instructions.
