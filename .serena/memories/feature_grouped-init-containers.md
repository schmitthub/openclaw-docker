# Feature: Grouped Init Containers, Individual Env Vars, Post-Start Commands

## Branch
`fix/setupcommandstarttime` — PR #33 open at https://github.com/schmitthub/openclaw-deploy/pull/33

## What Was Done

### 1. Grouped Pre-Start Commands ✅
- `setupCommands` (string[]) → `preStartCommands` (Record<string, string[]>)
- Each group key = one `docker run --rm` init container (one cold start per group)
- Commands within a group run sequentially in a single shell script (base64 encoded, piped to container)
- Env var scanning: system reads command text for `$VAR` references, only includes those vars in Pulumi triggers per group
- Groups depend on each other sequentially (openclaw can't handle concurrent config writes)
- `components/gateway-init.ts` fully rewritten

### 2. Individual Secret Env Vars ✅
- Killed the `gatewaySecretEnv-<profile>` JSON blob nightmare
- Each env var is its own Pulumi config entry: `gatewayEnv-<profile>-<KEY>`
- Discovered via `pulumi.runtime.allConfig()` prefix scanning in `index.ts`
- Each read via `cfg.requireSecret()` — individual `Output<string>` values
- Per-group trigger scanning works because key names are known at plan time
- `ocm env set/list/delete` CLI commands added to `scripts/manage.sh`

### 3. Post-Start Commands ✅ (BUT BROKEN — SEE ISSUES)
- `postStartCommands` (Record<string, string[]>) — same grouped format as pre-start
- New `GatewayPostInit` component in `components/gateway-post-init.ts`
- Uses `docker exec` into running gateway (container env inherited, no SSH env needed)
- Health wait before execution: polls `/healthz` with `wget` + `timeout 60`
- Wired up in `index.ts` after Gateway, depends on gateway

### 4. Named Buildx Builder ✅
- `builder: { name: "openclaw-builder" }` on all `docker_build.Image` resources
- `ensure-builder` command (idempotent) runs before builds — creates builder if missing
- Deterministic container name: `buildx_buildkit_openclaw-builder0`
- Confirmed: provider reuses the stopped container instead of creating new ones

### 5. Base Image Upgrades ✅ (from earlier PR #32, merged)
- node:22 → node:24, CoreDNS 1.12.1 → 1.14.2, filebrowser unpinned → v2.61.2
- `update-base-digests.sh` now derives images from `defaults.ts` (no hardcoded list)

### 6. Documentation ✅
- README.md: new sections for Pre-Start Command Groups, Post-Start Commands, Secret Environment Variables, ocm env CLI
- AGENTS.md: updated component hierarchy, project structure, deployment model
- .claude/rules/pulumi-config.md and docker-and-shell.md updated
- Pulumi.dev.yaml.example updated with new format

## Known Issues / Bugs to Fix

### POST-START IS NOT WORKING
- `GatewayPostInit` component creates in state but child resources (health-wait, group-default) are NOT in state
- `pulumi stack export` shows only 1 post-init resource (the component itself), no children
- `--target` on the component doesn't recurse into children
- Full `pulumi up` needed to test — currently running, waiting for result
- The `logging: "none"` on the group command was removed due to semgrep but needs investigation — may have been suppressing execution or exit codes
- `openclaw system heartbeat disable` fails manually with WS gateway close error but "succeeded" in Pulumi (suspicious 0.28s creation time)
- **Root cause unclear** — could be: target not recursing, create string not resolving, or logging:none swallowing errors

### openclaw node:24 Performance Regression
- `openclaw config set` takes ~18 seconds per call (was fast on node:22)
- 20 commands in one container = 369 seconds (6+ minutes) for the default group
- This is an openclaw issue, not ours — nothing we can do
- Total init time is ~15-20 minutes for 5 groups on first deploy

### Echo Lines Removed
- The `echo "[init:group N/M] command"` progress lines caused gRPC UTF-8 marshaling errors
- Removed from both gateway-init.ts and gateway-post-init.ts
- Commands run without progress echo — Pulumi resource names provide visibility
- Root cause was quote escaping in echo conflicting with base64 encoding

## Uncommitted Changes
- `gateway-post-init.ts` has `logging: "none"` removed (was added for semgrep, but may be causing execution issues)
- Need to handle semgrep rule — either re-add logging:none after fixing the real bug, or exclude post-start from the rule

## Files Modified (key files)
- `components/gateway-init.ts` — grouped pre-start
- `components/gateway-post-init.ts` — NEW, post-start via docker exec
- `components/gateway.ts` — envVars replaces secretEnv
- `components/gateway-image.ts` — named builder + ensure-builder
- `components/index.ts` — exports GatewayPostInit
- `config/types.ts` — SetupCommand removed, GatewayConfig uses Record<string, string[]>
- `index.ts` — env var scanning, grouped commands, post-init wiring
- `scripts/manage.sh` — ocm env set/list/delete
- `Pulumi.hetzner.yaml` — new format with groups + individual env vars
- `Pulumi.dev.yaml.example` — updated example

## Current Deploy State
- Pre-start groups: all 5 in state and working (openrouter, tailscale, discord, telegram, default)
- Gateway container: running, healthy
- Post-start: BROKEN — needs debugging
- A full `pulumi up` is currently running to test post-start without `--target`

## Lessons Learned
- `command.remote.Command` `create` changes cause resource replacement even if `triggers` haven't changed — avoid unnecessary code changes that alter the create string
- `dependsOn` is ordering only, not replacement cascade — but each code change to the component rebuilds ALL create strings
- `docker exec` inherits container env — no need to pass env vars from SSH session
- `logging: "none"` may suppress execution or exit codes — needs investigation
- `pulumi up --target` on a ComponentResource may not recurse into children
- YAML `\"` sequences can cause gRPC UTF-8 errors when embedded in echo lines and base64 encoded
- `pulumi config set` with wrong stack name or invalid YAML header (`openclaw config:` vs `config:`) silently wipes the config file

## IMPERATIVE
**ALWAYS check with the user before proceeding with the next task.** Do not make changes without confirmation. If all work is done, ask the user if they want to delete this memory.
