# AGENTS.md

## Repository Overview

This repository packages **OpenClaw** into Docker images for multiple Linux flavors and versions.

Primary goals:
- Keep Dockerfiles reproducible and easy to build.
- Support a matrix of Linux variants and version tags.
- Make it straightforward to compare, maintain, and update base images over time.

## What Agents Should Assume

- The core artifact in this repo is Docker-related build configuration.
- Dockerfiles in `dockerfiles/` are generated artifacts.
- Version metadata comes from npm package `openclaw` via `versions.sh`.
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
- Run `bash ./versions.sh latest beta && bash ./apply-templates.sh latest beta` after template/script changes.
- Verify Dockerfile syntax and build steps for touched variants.
- Ensure commands are non-interactive and CI-friendly.
- Confirm OpenClaw installation/startup steps still work for the modified target image(s).

## Out of Scope (Unless Explicitly Requested)

- Redesigning repository structure.
- Adding unrelated tooling or frameworks.
- Changing release/versioning policy beyond the requested task.

## Editing Style

- Keep docs concise and operational.
- Keep commits scoped to one concern (single distro/version family when possible).
- Prefer clarity over cleverness in shell and Docker instructions.
