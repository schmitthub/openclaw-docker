# Package: `internal/cmd`

Cobra command definitions and CLI option resolution.

## Files

| File | Purpose |
|------|---------|
| `root.go` | Root command, all flags, `mergedOptions()`, `applyEnvOverrides()` |
| `generate.go` | `generate` subcommand: resolve version + render artifacts |
| `resolve.go` | `resolve` subcommand: resolve version + write manifest |
| `render.go` | `render` subcommand: read manifest + render artifacts |
| `config.go` | `config init` subcommand: write config template |
| `version.go` | `version` subcommand: print build info |
| `prompt.go` | `confirmWrite()` — write safety prompts |

## Key Pattern: `mergedOptions(cmd)`

Central option resolution in `root.go`. Merges defaults -> config file -> env vars -> CLI flags.
All commands call this to get resolved `runtimeOptions`.

## Config Precedence

flags > env vars (`OPENCLAW_DOCKER_*`) > config file (`--config`) > defaults

## Adding a New Flag

1. Add field to `runtimeOptions` struct in `root.go`
2. Register flag in `NewRootCmd()` (use `PersistentFlags()` for cross-command flags)
3. Add default value in `mergedOptions()` defaults block
4. Add env override in `applyEnvOverrides()` if needed
5. Add `cmd.Flags().Changed()` check in `mergedOptions()` flag overlay section
6. Add to `render.Options` if it affects generation
