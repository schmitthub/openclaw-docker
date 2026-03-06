# Firewall Bypass + Agent Environment Prompt Feature

## Branch: `feat/firewall-bypass-exceptions`

## End Goal
Add firewall bypass (root-only SOCKS proxy) and agent environment prompt (ENVIRONMENT.md) to both the Pulumi IaC deployment and the `reference/` Docker Compose test harness.

## What Was Done

### Pulumi-side (pre-existing on branch before this session)
- `templates/bypass.ts` — `renderFirewallBypass()` pure template
- `templates/agent-prompt.ts` — `renderAgentPrompt()` pure template (now imports constants from defaults.ts)
- `templates/dockerfile.ts` — COPY firewall-bypass + chmod 700
- `templates/index.ts` — re-exports both new templates
- `components/gateway-image.ts` — writes firewall-bypass to build context
- `config/defaults.ts` — `BYPASS_SOCKS_PORT=9100`, `DEFAULT_BYPASS_TIMEOUT_SECS=30`
- `index.ts` — post-deploy `command.remote.Command` resources for ENVIRONMENT.md write + AGENTS.md injection
- `index.ts` — hardcoded setupCommands for `hooks enable bootstrap-extra-files` + config set paths
- `tests/templates.test.ts` — 23 new tests (15 bypass, 8 agent prompt) = 255 total passing
- `AGENTS.md` + `README.md` — documentation for both features

### Reference stack (done this session)
- [x] `reference/firewall-bypass` — static copy with constants inlined (SOCKS_PORT=9100, TIMEOUT=30, SSHD_PORT=2222)
- [x] `reference/Dockerfile` — COPY + chmod 700 after entrypoint
- [x] `reference/setup.sh` — ENVIRONMENT.md write (base64, sha256 hash-verified, root-owned 444) + AGENTS.md injection (prepend line 1, skip if file missing) + hooks enable bootstrap-extra-files
- [x] `reference/AGENTS.md` — updated Files table, Separation of Concerns table, Relationship section

### PR Review Fixes Applied (this session)
- [x] `renderAgentPrompt` now imports `BYPASS_SOCKS_PORT` and `DEFAULT_BYPASS_TIMEOUT_SECS` from defaults.ts (was hardcoded)
- [x] `reference/setup.sh` uses `shasum -a 256` instead of `sha256sum` (macOS compatibility)
- [x] AGENTS.md injection simplified: always `sed -i "1i\..."` (prepend line 1), no header detection
- [x] AGENTS.md injection: `[ ! -f "$FILE" ]` guard skips if file doesn't exist yet
- [x] `firewall-bypass` ssh -D wrapped in `if !` with actionable error message
- [x] `firewall-bypass` pgrep guarded with `|| true` for pipefail compatibility
- [x] `firewall-bypass` auto-kill uses `;` instead of `&&` so pidfile cleanup always runs
- [x] Gendered pronoun "his" → "their" in agent prompt
- [x] Stale reference/AGENTS.md descriptions fixed (DNAT→REDIRECT, socket wait→permissions fix, tailscaled→containerboot)
- [x] `index.ts` comment "immutable" → "read-only chmod 444"
- [x] `gateway-image.ts` build context comment updated to mention firewall-bypass
- [x] `reference/AGENTS.md` GatewayInit attribution → correct "command.remote.Command resources"
- [x] Tests: added SSH options test, actionable error test, constants-based agent prompt tests
- [x] DEFAULT_BYPASS_TIMEOUT_SECS bumped from 10 to 30

## Current State
- All 255 tests passing, types clean
- User is deploying to test
- No commits made yet on this session's changes

## TODO
- [ ] User confirms deploy works
- [ ] Commit all changes
- [ ] Update AGENTS.md/README.md if default timeout references say "10s" (check grep for "10s" or "10 seconds" in docs)
- [ ] Create PR or merge

## Key Files Modified
- `config/defaults.ts`, `templates/bypass.ts`, `templates/agent-prompt.ts`, `templates/dockerfile.ts`, `templates/index.ts`
- `components/gateway-image.ts`, `index.ts`, `tests/templates.test.ts`
- `AGENTS.md`, `README.md`
- `reference/firewall-bypass`, `reference/Dockerfile`, `reference/setup.sh`, `reference/AGENTS.md`

## Plan File
/Users/andrew/.claude/plans/shiny-wishing-snowflake.md

## IMPERATIVE
Always check with the user before proceeding with the next todo item. If all work is done, ask the user if they want to delete this memory.
