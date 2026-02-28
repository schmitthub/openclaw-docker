package config

func DefaultTemplate() string {
	return `# openclaw-docker configuration
#
# Precedence: flags > environment variables > config file > defaults
# Environment prefix: OPENCLAW_DOCKER_

# Requested version/tag to resolve (dist-tag or semver partial)
version: latest

# Where resolved version metadata is cached
versions_file: ${XDG_CACHE_HOME}/openclaw-docker/versions.json

# Output root for generated artifacts:
# - <output>/Dockerfile
# - <output>/compose.yaml
# - <output>/.env.openclaw
# - <output>/setup.sh
output: ./openclaw-deploy

# Cleanup is defensive-only (prints warning, no delete operations)
cleanup: false

# Enable debug logging
debug: false

# Additional apt packages for the generated Dockerfile
docker_apt_packages: ""

# Default OpenClaw runtime settings baked into generated Dockerfile/.env.openclaw
openclaw_config_dir: /home/node/.openclaw
openclaw_workspace_dir: /home/node/.openclaw/workspace
openclaw_gateway_port: "18789"
openclaw_bridge_port: "18790"
openclaw_gateway_bind: lan

# Defaults used by generated compose/.env.openclaw
openclaw_image: openclaw:local
openclaw_gateway_token: ""
openclaw_extra_mounts: ""
openclaw_home_volume: ""

# Comma-separated domains to whitelist in squid egress proxy.
# Known providers: api.anthropic.com, api.openai.com,
#   generativelanguage.googleapis.com, openrouter.ai, api.x.ai
# openclaw.ai is always included.
squid_allowed_domains: ""
`
}
