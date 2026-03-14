# Init Container Consolidation — Future Feature

## Problem
Each `setupCommand` in `GatewayInit` runs as a separate `docker run --rm` container via `command.remote.Command`. With 30+ setup commands, each cold-starting node:24, this takes 60+ minutes on first deploy. Each container pays the full Node.js cold start cost (~2min per command).

## Current Architecture (gateway-init.ts)
- Each `setupCommand` → separate `command.remote.Command` resource
- Each resource runs `docker run --rm --network none --user node` with the gateway image
- Commands are base64-encoded and piped to `sh -e`
- Env vars passed via SSH `AcceptEnv` (secrets) and command string (non-secrets)
- Env var scanning: checks if command references `$TAILSCALE_SERVE_HOST` or `$OPENCLAW_GATEWAY_TOKEN` to determine selective triggers
- Commands chained with `dependsOn` — sequential execution

## Why Per-Command Granularity Is Broken
1. **Cascade replacement**: Commands are chained sequentially. If command 0 triggers a replacement, Pulumi cascades replacements to all downstream commands. The selective re-run optimization is effectively useless.
2. **Environment blob diffing**: The `environment` property is a single object. Any env var change (even unrelated) triggers re-run of all commands that share the environment.
3. **N cold starts**: Each `docker run --rm` boots Node.js from scratch. No compile cache (ephemeral containers). With node:24 this is ~2min per command.

## Proposed Solution
Single `docker run --rm` container that executes all setup commands in one shell script.

### Approach
1. Concatenate all `setupCommands` into one shell script
2. Base64-encode the entire script
3. One `command.remote.Command` resource with one `docker run --rm`
4. Content hash of all commands as trigger — if any command changes, re-run the whole script
5. All env vars available to all commands (no per-command scanning needed since they all re-run anyway)
6. Single cold start instead of N

### Implementation Sketch
```typescript
// Combine all commands into one script
const allCmds = setupCmds.join("\n");
const encoded = Buffer.from(allCmds).toString("base64");
const contentHash = crypto.createHash("sha256").update(allCmds).digest("hex");

// Single docker run
const initCmd = buildInitCommand({
  profile: args.profile,
  imageName: args.imageName,
  encoded,
  dDir,
  // Always include all env vars since granularity is gone
  needsHostname: true,
  needsToken: true,
  tailscaleHostname: args.tailscaleHostname,
  gatewayToken: args.gatewayToken,
  secretEnv: args.secretEnv,
});

new command.remote.Command(`${name}-setup`, {
  connection: args.connection,
  create: initCmd.create,
  environment: initCmd.environment,
  triggers: [contentHash],
  // ...
});
```

### Benefits
- 1 cold start instead of 30+ (60+ min → ~3 min)
- Simpler Pulumi resource graph (1 resource instead of 30+)
- No cascade replacement problem
- Env var changes still trigger re-run (via content hash or environment diff)

### Tradeoffs
- All-or-nothing re-run (but this is already the de facto behavior)
- Single error fails the whole script (but `set -e` already does this per-command)
- Lose per-command Pulumi resource naming in logs (harder to see which command failed — mitigate with `echo` markers in the script)

### Migration
- Replace the for-loop in `GatewayInit` constructor with single resource
- Keep the `buildInitCommand` helper but adapt for combined script
- Update tests in `components.test.ts`
- The `openclaw.init-hash` label on the gateway container stays (content hash of combined script)
