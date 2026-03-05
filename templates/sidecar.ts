import { ENVOY_EGRESS_PORT, ENVOY_UID } from "../config/defaults";

/**
 * Renders the Tailscale sidecar entrypoint script.
 *
 * The sidecar owns the network namespace (gateway + envoy share it via
 * network_mode: container:). All containers share localhost, so iptables
 * uses REDIRECT (not DNAT) to route TCP to envoy's listener.
 *
 * Security model:
 * - TCP: REDIRECT to envoy on localhost (SNI whitelist)
 * - UDP: only root (uid 0 = tailscaled/containerboot) can send UDP
 * - The node user (openclaw gateway) cannot send UDP (exfiltration blocked)
 * - Envoy (uid ${ENVOY_UID}) is excluded from redirect to prevent loops
 */
export function renderSidecarEntrypoint(): string {
  return `#!/bin/sh
# Tailscale sidecar entrypoint — owns the network namespace.
# Gateway + Envoy containers share this netns via network_mode: container:<sidecar>.
#
# Responsibilities:
# 1. iptables NAT (TCP REDIRECT to envoy) + UDP owner-match
# 2. Hand off to containerboot (official Tailscale entrypoint)
set -eu

# Exclude envoy and root from redirect to prevent loops.
# Envoy (uid ${ENVOY_UID}) must reach upstream directly.
# Root (uid 0) runs containerboot/tailscaled — needs direct network access.
iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner ${ENVOY_UID} -j RETURN
iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner 0 -j RETURN

# Per-destination REDIRECT rules for SSH/TCP egress (port-mapped through Envoy).
# OPENCLAW_TCP_MAPPINGS format: "dst|dstPort|envoyPort;dst|dstPort|envoyPort;..."
# Each entry gets a dedicated iptables rule routing matching traffic to a specific Envoy listener port.
if [ -n "\${OPENCLAW_TCP_MAPPINGS:-}" ]; then
  OLD_IFS="$IFS"
  IFS=';'
  for entry in $OPENCLAW_TCP_MAPPINGS; do
    IFS='|'
    # shellcheck disable=SC2086
    set -- $entry
    DST="$1"; DST_PORT="\${2:-}"; ENVOY_PORT="\${3:-}"
    IFS="$OLD_IFS"
    if [ -z "$DST" ] || [ -z "$DST_PORT" ] || [ -z "$ENVOY_PORT" ]; then
      echo "WARN: malformed TCP mapping entry: $entry" >&2
      IFS=';'
      continue
    fi
    # Check if DST is an IPv4 address
    if echo "$DST" | grep -qE '^[0-9]{1,3}(\\.[0-9]{1,3}){3}$'; then
      RESOLVED_IP="$DST"
    elif echo "$DST" | grep -q ':'; then
      # IPv6 literal — iptables can't route IPv6 on this network
      echo "WARN: IPv6 destination $DST — iptables routing is IPv4-only" >&2
      IFS=';'
      continue
    else
      # Domain — resolve to IPv4 for iptables matching
      RESOLVE_ERR="$(getent ahostsv4 "$DST" 2>&1 1>/dev/null)" || true
      RESOLVED_IP="$(getent ahostsv4 "$DST" 2>/dev/null | head -1 | awk '{print $1}')" || true
      if [ -z "$RESOLVED_IP" ] || ! echo "$RESOLVED_IP" | grep -qE '^[0-9]{1,3}(\\.[0-9]{1,3}){3}$'; then
        echo "WARN: cannot resolve '$DST' for TCP mapping\${RESOLVE_ERR:+ ($RESOLVE_ERR)} — skipping" >&2
        IFS=';'
        continue
      fi
    fi
    if ! iptables -t nat -A OUTPUT -p tcp -d "$RESOLVED_IP" --dport "$DST_PORT" \\
         -j REDIRECT --to-ports "$ENVOY_PORT" 2>&1; then
      echo "ERROR: iptables REDIRECT failed for $DST:$DST_PORT -> :$ENVOY_PORT (resolved=$RESOLVED_IP)" >&2
      exit 1
    fi
    IFS=';'
  done
  IFS="$OLD_IFS"
fi

# Catch-all: redirect all other outbound TCP to envoy's transparent proxy listener.
iptables -t nat -A OUTPUT -p tcp ! -d 127.0.0.0/8 -j REDIRECT --to-ports ${ENVOY_EGRESS_PORT}

# UDP: Docker DNS for everyone, root (containerboot/tailscaled) for WireGuard, drop all others.
iptables -A OUTPUT -p udp -d 127.0.0.11 -j ACCEPT
iptables -A OUTPUT -p udp -m owner --uid-owner 0 -j ACCEPT
iptables -A OUTPUT -p udp -j DROP

# Hand off to official Tailscale entrypoint (containerboot handles auth, state, serve config).
exec /usr/local/bin/containerboot
`;
}
