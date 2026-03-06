import {
  BYPASS_SOCKS_PORT,
  DEFAULT_BYPASS_TIMEOUT_SECS,
  SSHD_PORT,
} from "../config/defaults";

export function renderFirewallBypass(): string {
  return `#!/bin/bash
set -euo pipefail

# Root-only SOCKS proxy for temporary firewall bypass.
# Starts an SSH SOCKS proxy (root-owned, bypasses iptables RETURN rule).
# Usage: firewall-bypass [timeout_secs|stop|list]

SOCKS_PORT=${BYPASS_SOCKS_PORT}
PIDFILE="/run/firewall-bypass.pid"

if [ "$(id -u)" -ne 0 ]; then
  echo "ERROR: firewall-bypass must be run as root" >&2
  exit 1
fi

stop_proxy() {
  if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
      kill "$PID" 2>/dev/null || true
      echo "Stopped firewall bypass proxy (PID $PID)"
    else
      echo "Proxy not running (stale PID $PID)"
    fi
    rm -f "$PIDFILE"
  else
    echo "No active firewall bypass proxy"
  fi
}

list_proxy() {
  if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "Firewall bypass proxy ACTIVE on localhost:$SOCKS_PORT (PID $PID)"
      return 0
    else
      rm -f "$PIDFILE"
    fi
  fi
  echo "No active firewall bypass proxy"
  return 1
}

start_proxy() {
  TIMEOUT="\${1:-${DEFAULT_BYPASS_TIMEOUT_SECS}}"

  # Idempotent: if already running, show status and exit
  if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "Firewall bypass proxy already running on localhost:$SOCKS_PORT (PID $PID)"
      echo "Use 'firewall-bypass stop' to kill it, or wait for auto-timeout"
      return 0
    fi
    rm -f "$PIDFILE"
  fi

  if ! ssh -D "127.0.0.1:$SOCKS_PORT" -f -N \\
    -o StrictHostKeyChecking=no \\
    -o UserKnownHostsFile=/dev/null \\
    root@127.0.0.1 -p ${SSHD_PORT}; then
    echo "ERROR: Failed to start SOCKS proxy — is sshd running on port ${SSHD_PORT}?" >&2
    exit 1
  fi

  # Find the ssh process we just started
  PID=$(pgrep -f "ssh -D 127.0.0.1:$SOCKS_PORT" | head -1 || true)
  if [ -z "$PID" ]; then
    echo "ERROR: Failed to find SOCKS proxy process" >&2
    exit 1
  fi
  echo "$PID" > "$PIDFILE"

  # Auto-kill after timeout (background, disowned)
  (sleep "$TIMEOUT"; kill "$PID" 2>/dev/null; rm -f "$PIDFILE") &
  disown

  echo "Firewall bypass proxy started on localhost:$SOCKS_PORT (PID $PID, timeout \${TIMEOUT}s)"
  echo "Usage: curl --socks5 localhost:$SOCKS_PORT https://example.com"
}

case "\${1:-}" in
  stop)
    stop_proxy
    ;;
  list)
    list_proxy
    ;;
  *)
    start_proxy "\${1:-}"
    ;;
esac
`;
}
