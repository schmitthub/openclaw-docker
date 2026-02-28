package config

func DefaultTemplate() string {
	return `# openclaw-docker configuration
#
# Precedence: flags > environment variables > config file > defaults
# Environment prefix: OPENCLAW_DOCKER_

# Requested versions/tags to resolve (dist-tags or semver selectors)
versions:
  - latest

# Where resolved version metadata is cached
versions_file: ${XDG_CACHE_HOME}/openclaw-docker/versions.json

# Template helper scripts source path used in generated Dockerfiles
templates_dir: ./build/templates

# Output root for generated artifacts:
# - <output>/<version>/<variant>/Dockerfile
# - <output>/compose.yaml
# - <output>/.env.openclaw
output: ./openclawdockerfiles

# Cleanup is defensive-only (prints warning, no delete operations)
cleanup: false

# Enable debug logging
debug: false

# Base distro defaults for generated variants
debian_default: trixie
alpine_default: alpine3.23

# Variant matrix (keys are variant names; values are arch lists when needed)
variants:
  trixie: []
  bookworm: []
  alpine3.23: []
  alpine3.22: []

# Target architectures
arches:
  - amd64
  - arm64v8

# Additional apt packages for Debian-based generated Dockerfiles
docker_apt_packages: ""

# Default OpenClaw runtime settings baked into generated Dockerfiles/.env.openclaw
openclaw_config_dir: /home/openclaw/.openclaw
openclaw_workspace_dir: /home/openclaw/.openclaw/workspace
openclaw_gateway_port: "18789"
openclaw_bridge_port: "18790"
openclaw_gateway_bind: lan

# Defaults used by generated compose/.env.openclaw
openclaw_image: openclaw:local
openclaw_gateway_token: ""
openclaw_extra_mounts: ""
openclaw_home_volume: ""
`
}
