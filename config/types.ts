// Egress policy types

export interface PathRule {
  path: string; // trailing "*" → Envoy prefix match; otherwise exact match. e.g. "/messages/*", "/health"
  action: "deny"; // deny matching paths; omit paths to allow them
}

export interface EgressRule {
  dst: string; // domain "x.com" | IPv4 "140.82.121.4" | IPv6 "2001:db8::1" | CIDR "10.0.0.0/24"
  proto: "tls" | "ssh" | "tcp"; // tls: SNI-based passthrough or MITM inspection; ssh/tcp: per-rule Envoy port mapping
  port?: number; // required for ssh/tcp, optional for tls (default 443)
  action: "allow" | "deny";
  inspect?: boolean; // MITM TLS termination for path-level rules
  pathRules?: PathRule[]; // when inspect=true
}

// VPS provider type (derived from PROVIDERS constant in defaults.ts)
import { PROVIDERS } from "./defaults";
export type VpsProvider = (typeof PROVIDERS)[number];

// Tailscale mode per gateway
export type TailscaleMode = "serve" | "funnel" | "off";

// Gateway configuration
export interface GatewayConfig {
  profile: string; // unique name for this gateway instance
  version: string; // openclaw version (npm dist-tag or semver)
  packages: string[]; // apt packages to bake into image
  port: number; // host port (maps to 18789 inside container)
  bridgePort?: number; // bridge port (defaults 18790)
  tailscale: TailscaleMode;
  installBrowser?: boolean; // bake Playwright + Chromium (~300MB)
  configSet: Record<string, string>; // openclaw config set key=value pairs
  env?: Record<string, string>; // additional env vars for container
}

// Per-rule port mapping for SSH/TCP egress (passed to gateway containers)
export interface TcpPortMapping {
  /** Destination domain or IP */
  dst: string;
  /** Destination port (e.g. 22 for SSH, 5432 for PostgreSQL) */
  dstPort: number;
  /** Protocol from the egress rule */
  proto: "ssh" | "tcp";
  /** Dedicated Envoy listener port for this mapping */
  envoyPort: number;
}

// Full stack configuration
export interface StackConfig {
  // VPS
  provider: VpsProvider;
  serverType: string; // e.g. "cx22" (Hetzner), "s-1vcpu-1gb" (DO), "VM.Standard.A1.Flex" (OCI)
  region: string; // e.g. "fsn1" (Hetzner), "nyc1" (DO), availability domain (OCI)
  sshKeyId: string; // provider-specific SSH key ID or fingerprint

  // Tailscale
  tailscaleAuthKey: string; // secret: one-time auth key

  // Egress
  egressPolicy: EgressRule[];

  // Gateways (1+)
  gateways: [GatewayConfig, ...GatewayConfig[]];
}
