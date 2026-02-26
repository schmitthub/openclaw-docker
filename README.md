# openclaw-docker

Docker image packaging for OpenClaw across multiple Linux flavors and versions.

## Overview

- Resolves OpenClaw versions from the official npm source (`openclaw` package).
- Supports npm dist-tags such as `latest` and `beta`, plus explicit version patterns.
- Generates Dockerfiles per version/variant matrix under `dockerfiles/<version>/<variant>/`.
- Installs OpenClaw in images via the official installer command:
	`curl -fsSL https://openclaw.ai/install.sh | bash`

## Variants

Current variants are defined in `versions.sh` and include:

- `trixie`
- `bookworm`
- `alpine3.23`
- `alpine3.22`

## Version and tag resolution

- `versions.sh` queries npm package metadata from `openclaw`.
- Dist-tags (for example `latest`, `beta`) are resolved first.
- If an input is not a dist-tag, semver matching is used.
- Resolved versions are written to `versions.json`.

## Common commands

```bash
# resolve default versions and regenerate Dockerfiles
make update

# resolve explicit dist-tags
make update VERSIONS='latest beta'

# resolve an explicit version pattern (example)
make update VERSIONS='2026.2'

# regenerate Dockerfiles from existing versions.json
make apply-templates

# list versions and variants
make list-versions
make list-variants VERSION=<version>

# build one image
make build VERSION=<version> VARIANT=alpine3.23

# build all variants for one version
make build-version VERSION=<version>

# build all versions/variants currently generated
make build-all
```

## Repository structure

- `versions.sh`: resolves versions/tags and generates `versions.json`
- `apply-templates.sh`: renders Dockerfiles from template + versions metadata
- `build/templates/Dockerfile.template`: source Dockerfile template
- `build/templates/docker-entrypoint.sh`: runtime entrypoint behavior
- `build/templates/docker-init-firewall.sh`: optional firewall setup helper
- `dockerfiles/`: generated output, not edited directly

