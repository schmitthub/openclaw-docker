# Tailscale Docker Container Documentation

Official reference: https://tailscale.com/docs/features/containers/docker

## Image Availability & Versioning

**Registries:**
- Docker Hub: `tailscale/tailscale`
- GitHub Packages: `ghcr.io/tailscale/tailscale`

**Version Tags:**
- `stable`, `latest` — Current stable releases
- Specific versions — `v1.58.2`, `v1.58`, etc.
- `unstable` — Development builds (e.g., `unstable-v1.59.37`)

## Environment Variables

### Authentication

| Variable | Purpose | Notes |
|----------|---------|-------|
| `TS_AUTHKEY` | One-time auth key for node authentication | Equivalent to CLI `tailscale login` |
| `TS_CLIENT_ID` | OAuth client ID | For OAuth-based auth |
| `TS_CLIENT_SECRET` | OAuth client secret | For OAuth-based auth |
| `TS_ID_TOKEN` | Identity provider token | For workload identity federation |
| `TS_AUDIENCE` | Audience claim for ID tokens | Used in cloud environments for auto-generated ID tokens |
| `TS_AUTH_ONCE` | Boolean flag | If set, attempt login only once (don't reconnect if missing) |

**Auth Key Variants:**
- Append `?ephemeral=true` to auth key to register temporary (ephemeral) nodes
- OAuth secrets support tag specification: `TS_EXTRA_ARGS=--advertise-tags=tag:ci`

### Networking

| Variable | Purpose | Notes |
|----------|---------|-------|
| `TS_ACCEPT_DNS` | Accept DNS from admin console | Enables admin-controlled DNS settings |
| `TS_HOSTNAME` | Custom node hostname | Overrides default hostname in Tailnet |
| `TS_ROUTES` | Advertise subnet routes | Comma-separated CIDR subnets |
| `TS_SOCKS5_SERVER` | Enable SOCKS5 proxy | Format: `address:port` (e.g., `0.0.0.0:1055`) |
| `TS_OUTBOUND_HTTP_PROXY_LISTEN` | HTTP proxy listener | Configure HTTP proxy for container |
| `TS_DEST_IP` | Proxy destination IP | Incoming Tailscale traffic proxies to this IP |
| `TS_USERSPACE` | Enable userspace networking | **Enabled by default** (no `/dev/net/tun` or `CAP_NET_ADMIN` required) |

### Storage & API

| Variable | Purpose | Notes |
|----------|---------|-------|
| `TS_STATE_DIR` | Persistent state directory | Must persist across restarts for Tailscale state |
| `TS_SOCKET` | Unix socket path for LocalAPI | Default: `/var/run/tailscale/tailscaled.sock` |
| `TS_KUBE_SECRET` | Kubernetes secret name | Default: `tailscale`; set to `""` to disable |

### Monitoring (v1.78+)

| Variable | Purpose | Notes |
|----------|---------|-------|
| `TS_ENABLE_HEALTH_CHECK` | Enable `/healthz` endpoint | Returns `200 OK` if node has at least one tailnet IP, `503` otherwise |
| `TS_ENABLE_METRICS` | Enable `/metrics` endpoint | Prometheus-compatible metrics |
| `TS_LOCAL_ADDR_PORT` | Health/metrics listen address | Default: `[::]:9002` |

**Health Check Details:**
- `/healthz` endpoint is unauthenticated
- Returns `200 OK` when node has ≥1 tailnet IP
- Returns `503` Service Unavailable when not connected
- Both `/healthz` and `/metrics` are unauthenticated

### Advanced Configuration

| Variable | Purpose | Notes |
|----------|---------|-------|
| `TS_SERVE_CONFIG` | JSON config file path for Serve/Funnel | Enables `tailscale serve` functionality |
| `TS_EXTRA_ARGS` | Additional `tailscale up` flags | Appended to login command (e.g., `--advertise-tags=tag:container`) |
| `TS_TAILSCALED_EXTRA_ARGS` | Additional `tailscaled` daemon flags | Daemon-level configuration |

## Docker Compose Sidecar Pattern

### Basic Sidecar Setup

```yaml
version: '3.8'

services:
  # Tailscale sidecar container owns network namespace
  tailscale:
    image: tailscale/tailscale:latest
    environment:
      - TS_AUTHKEY=<your-auth-key>
      - TS_STATE_DIR=/var/lib/tailscale
      - TS_USERSPACE=false
    volumes:
      - ./state:/var/lib/tailscale
    devices:
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - net_admin
    restart: unless-stopped

  # App shares tailscale's network namespace
  app:
    image: myapp:latest
    network_mode: service:tailscale
    depends_on:
      - tailscale
```

### Key Pattern Elements

- **Sidecar**: Tailscale container on own network, owns namespace
- **Dependent app**: `network_mode: service:tailscale` shares Tailscale's networking
- **State volume**: Persists across restarts
- **Device mapping**: `/dev/net/tun` for kernel networking (when `TS_USERSPACE=false`)
- **Capabilities**: `net_admin` required for kernel mode

## Volumes & State Persistence

**State Directory Configuration:**

| Setting | Purpose | Persistence |
|---------|---------|-------------|
| `TS_STATE_DIR` | Local filesystem state | Must mount volume to persist across restarts |
| `TS_KUBE_SECRET` | Kubernetes secret (default) | Automatic in Kubernetes; set `""` to use `TS_STATE_DIR` instead |

**Volume Mount Example:**
```yaml
volumes:
  - ./state:/var/lib/tailscale
  - /var/run/tailscale:/var/run/tailscale  # Optional: socket for other containers
```

**Important:** Without persistent state, node re-authentication is required on each restart.

## Capabilities & Device Requirements

### Userspace Mode (Default)
- No special capabilities needed
- No device mappings required
- No `/dev/net/tun` access
- Works in restricted environments (no `CAP_NET_ADMIN`)
- `TS_USERSPACE=true` (enabled by default)

### Kernel Mode (`TS_USERSPACE=false`)
- Requires: `cap_add: [net_admin]`
- Requires: `devices: [/dev/net/tun:/dev/net/tun]`
- Uses kernel TUN for networking (faster, more efficient)
- May require elevated privileges in some container environments

### Capability/Device Matrix

| Mode | TS_USERSPACE | cap_add | /dev/net/tun | Restrictions |
|------|--------------|---------|--------------|--------------|
| Userspace (default) | true | None | Not needed | None |
| Kernel | false | net_admin | Required | Host kernel TUN device required |

## Serve/Funnel Configuration

**Via Environment Variable:**

```bash
TS_SERVE_CONFIG=/path/to/serve-config.json
```

**JSON Configuration File Format:**

The file specifies Tailscale Serve/Funnel rules (e.g., port forwarding, HTTPS endpoints).

**Example Pattern:**
```json
{
  "TCP": {
    "443": {
      "HTTPS": true
    }
  },
  "Web": {
    "/": {
      "Text": "Hello"
    }
  }
}
```

**Usage Notes:**
- Serve config enables exposing services via Tailscale (private network)
- Funnel extends to public internet
- Configuration file must be a valid JSON
- Changes may require container restart

## Health Check Patterns

### Docker Compose Healthcheck

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:9002/healthz"]
  interval: 30s
  timeout: 5s
  retries: 3
  start_period: 5s
```

### Health Endpoint Details

- **Endpoint**: `http://localhost:9002/healthz` (or configured `TS_LOCAL_ADDR_PORT`)
- **Status Codes**:
  - `200 OK` — Node has at least one tailnet IP (healthy)
  - `503 Service Unavailable` — Node not connected (unhealthy)
- **Authentication**: Unauthenticated (no credentials required)
- **Availability**: v1.78+ only

### Metrics Endpoint

- **Endpoint**: `http://localhost:9002/metrics`
- **Format**: Prometheus-compatible
- **Authentication**: Unauthenticated
- **Availability**: v1.78+ only

## Userspace vs Kernel Networking

### Userspace Networking (`TS_USERSPACE=true`, default)

**Pros:**
- No elevated privileges required
- No `/dev/net/tun` device needed
- Works in restricted/sandboxed environments
- Enabled by default (secure default)

**Cons:**
- Slightly higher CPU overhead
- May have minor performance impact on high-throughput scenarios

### Kernel Networking (`TS_USERSPACE=false`)

**Pros:**
- Better performance (kernel-level networking)
- Lower CPU overhead
- More efficient for sustained high traffic

**Cons:**
- Requires `CAP_NET_ADMIN` capability
- Requires `/dev/net/tun` device access
- Not available in restricted environments
- Not available in some cloud container platforms

## Official Image Tag Conventions

| Tag Pattern | Use Case | Stability |
|-------------|----------|-----------|
| `latest` | Current stable release | Recommended for most users |
| `stable` | Stable release channel | Same as `latest` |
| `v1.58.2` | Specific version | Pinned for reproducibility |
| `v1.58` | Minor version (patch updates) | Allows patch updates, minor is fixed |
| `unstable` | Latest development build | Bleeding edge, may be unstable |
| `unstable-v1.59.37` | Specific unstable build | Development/testing only |

**Recommendation:** Use `stable` or `latest` for production; use specific version tags (e.g., `v1.58.2`) if reproducibility is critical.

## Important Notes & Gotchas

### Authentication

- `TS_AUTHKEY` is a one-time key; once used, it's consumed and cannot be reused
- Use `TS_AUTH_ONCE` to prevent reconnection attempts if key expires
- Ephemeral nodes (appended `?ephemeral=true`) auto-cleanup when offline for 7 days
- OAuth/OIDC tokens may have shorter lifespans than auth keys

### State Management

- **State directory MUST persist** across container restarts
- Without persistence, node re-authenticates on every restart (consumes auth keys)
- In Kubernetes, state defaults to secrets (configurable via `TS_KUBE_SECRET`)
- On Docker, use volume mounts: `volumes: [./state:/var/lib/tailscale]`

### Networking

- Userspace mode is the default and recommended for most containers
- Kernel mode (`TS_USERSPACE=false`) is only needed for advanced use cases
- When using `network_mode: service:tailscale`, the sidecar owns the namespace; dependent containers inherit all routing
- Sidecar must start before dependent containers

### DNS & Health

- Health endpoint requires v1.78+; older versions don't expose `/healthz`
- Health endpoint is unauthenticated (no credentials needed)
- If node is not connected, health returns `503` until connected

### Kubernetes

- Default secret name is `tailscale`; override with `TS_KUBE_SECRET=""`
- Auth key can be stored in the secret's `authkey` field if `TS_AUTHKEY` env var is unset
- For local disk state instead of secrets, set `TS_KUBE_SECRET=""` and `TS_STATE_DIR=/var/lib/tailscale` with a persistent volume

### Local API Socket

- Default socket: `/var/run/tailscale/tailscaled.sock`
- Can be overridden with `TS_SOCKET`
- Required if other containers need to access Tailscale API
- Often mounted as a shared volume in sidecar patterns: `volumes: [/var/run/tailscale:/var/run/tailscale]`

## Sidecar Model Compatibility with OpenClaw

**Pattern Alignment:**
- The official Tailscale Docker pattern matches OpenClaw's sidecar architecture
- Dependent containers use `network_mode: service:tailscale` (same as OpenClaw's `network_mode: container:tailscale-<profile>`)
- State persistence via `TS_STATE_DIR` and volume mount aligns with `${dataDir}/tailscale`
- Health endpoint available via `/healthz` (useful for monitoring)
- Userspace networking is default (no special capabilities by default)
- When kernel mode is used, `cap_add: [net_admin]` and `devices: [/dev/net/tun]` are required

**Differences from OpenClaw:**
- Official docs use `network_mode: service:` (Docker Compose); OpenClaw uses `network_mode: container:` (generic Docker API)
- OpenClaw adds capabilities and device mapping for Kernel TUN (required for iptables DNAT injection)
- OpenClaw uses `TS_SOCKET` explicitly for inter-container communication
- OpenClaw adds custom `sidecar-entrypoint.sh` for iptables/routing setup (not in official image)
- OpenClaw uses `TS_EXTRA_ARGS` pattern for `--ssh --operator=node` configuration
