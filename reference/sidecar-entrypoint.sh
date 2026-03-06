#!/bin/sh
# Tailscale sidecar entrypoint — owns the network namespace.
set -eu

# Get envoy's UID (101 in the official image)
# Exclude envoy from redirect to prevent loop
iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner ${ENVOY_UID:-101} -j RETURN
iptables -t nat -A OUTPUT -p tcp -m owner --uid-owner 0 -j RETURN

# Redirect all other outbound TCP through envoy
iptables -t nat -A OUTPUT -p tcp ! -d 127.0.0.0/8 -j REDIRECT --to-ports 10000

# Redirect DNS (UDP 53) from node user (uid 1000) to CoreDNS allowlist proxy
iptables -t nat -A OUTPUT -p udp --dport 53 -m owner --uid-owner 1000 -j REDIRECT --to-port 5300

# UDP: tailscaled only + DNS for everyone else
iptables -A OUTPUT -p udp -d 127.0.0.11 -j ACCEPT
iptables -A OUTPUT -p udp --dport 5300 -j ACCEPT
iptables -A OUTPUT -p udp -m owner --uid-owner root -j ACCEPT
iptables -A OUTPUT -p udp -j DROP

# Hand off to the real tailscale entrypoint
exec /usr/local/bin/containerboot "$@"
