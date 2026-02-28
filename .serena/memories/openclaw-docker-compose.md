# Official OpenClaw Docker Compose & Setup

## docker-compose.yml
Source: https://raw.githubusercontent.com/openclaw/openclaw/refs/heads/main/docker-compose.yml

```yaml
services:
  openclaw-gateway:
    image: ${OPENCLAW_IMAGE:-openclaw:local}
    environment:
      HOME: /home/node
      TERM: xterm-256color
      OPENCLAW_GATEWAY_TOKEN: ${OPENCLAW_GATEWAY_TOKEN}
      CLAUDE_AI_SESSION_KEY: ${CLAUDE_AI_SESSION_KEY}
      CLAUDE_WEB_SESSION_KEY: ${CLAUDE_WEB_SESSION_KEY}
      CLAUDE_WEB_COOKIE: ${CLAUDE_WEB_COOKIE}
    volumes:
      - ${OPENCLAW_CONFIG_DIR}:/home/node/.openclaw
      - ${OPENCLAW_WORKSPACE_DIR}:/home/node/.openclaw/workspace
    ports:
      - "${OPENCLAW_GATEWAY_PORT:-18789}:18789"
      - "${OPENCLAW_BRIDGE_PORT:-18790}:18790"
    init: true
    restart: unless-stopped
    command:
      [
        "node",
        "dist/index.js",
        "gateway",
        "--bind",
        "${OPENCLAW_GATEWAY_BIND:-lan}",
        "--port",
        "18789",
      ]

  openclaw-cli:
    image: ${OPENCLAW_IMAGE:-openclaw:local}
    environment:
      HOME: /home/node
      TERM: xterm-256color
      OPENCLAW_GATEWAY_TOKEN: ${OPENCLAW_GATEWAY_TOKEN}
      BROWSER: echo
      CLAUDE_AI_SESSION_KEY: ${CLAUDE_AI_SESSION_KEY}
      CLAUDE_WEB_SESSION_KEY: ${CLAUDE_WEB_SESSION_KEY}
      CLAUDE_WEB_COOKIE: ${CLAUDE_WEB_COOKIE}
    volumes:
      - ${OPENCLAW_CONFIG_DIR}:/home/node/.openclaw
      - ${OPENCLAW_WORKSPACE_DIR}:/home/node/.openclaw/workspace
    stdin_open: true
    tty: true
    init: true
    entrypoint: ["node", "dist/index.js"]
```

## Key patterns from docker-setup.sh
Source: https://raw.githubusercontent.com/openclaw/openclaw/refs/heads/main/docker-setup.sh

1. **Token is an ENV VAR, not config set**: `OPENCLAW_GATEWAY_TOKEN` is passed as environment variable to both services. Setup NEVER does `config set gateway.auth.mode` or `config set gateway.auth.token`. OpenClaw reads the env var directly.

2. **Token flow**: Read from existing config → reuse. Else generate new. Written to .env. Passed to both services via env var. User told to enter it during onboard. No overwrite after onboard.

3. **Gateway health via exec**: `docker compose exec openclaw-gateway node dist/index.js health --token "$TOKEN"` — runs INSIDE the gateway container, not via CLI service.

4. **CLI service is for file I/O ops**: `run --rm openclaw-cli onboard`, `run --rm openclaw-cli config set/get`. These don't need gateway WebSocket.

5. **No auth config set after onboard**: Official setup only calls `ensure_control_ui_allowed_origins` after onboard. No `config set gateway.auth.*`.

6. **OPENCLAW_GATEWAY_BIND default is `lan`**: Exported and used in compose command.

7. **Both services share same image, volumes, and token env var**.
