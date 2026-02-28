package config

func DefaultTemplate() string {
	return `# openclaw-docker configuration
#
# Precedence: flags > environment variables > config file > defaults
# Environment prefix: OPENCLAW_DOCKER_

# Requested version/tag to resolve (dist-tag or semver partial)
version: latest

# Output root for generated artifacts:
# - <output>/compose.yaml
# - <output>/.env.openclaw
# - <output>/setup.sh
# - <output>/compose/envoy/envoy.yaml
# - <output>/compose/openclaw/Dockerfile
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

# Comma-separated domains to whitelist in the Envoy egress proxy.
# openclaw.ai is always included. Set to "" to allow only openclaw.ai.
allowed_domains: "api.anthropic.com,api.openai.com,generativelanguage.googleapis.com,openrouter.ai,api.x.ai"

# External origin for server deployments (e.g. https://myclaw.example.com).
# Added to gateway.controlUi.allowedOrigins alongside https://localhost.
# Leave empty for local-only use.
external_origin: ""
`
}
