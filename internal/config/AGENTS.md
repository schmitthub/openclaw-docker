# Package: `internal/config`

YAML config file loading and default template generation.

## Files

| File | Purpose |
|------|---------|
| `config.go` | `FileConfig` struct, `Load(path)`, `FromString(yaml)` |
| `template.go` | `DefaultTemplate()` — annotated YAML config template |

## FileConfig

Struct with YAML tags mapping to config file keys. All fields are optional — empty/nil means "use default".
Bool fields use `*bool` to distinguish "not set" from "false".

## Usage

- `config.Load(path)` — read YAML file into `FileConfig`
- `config.FromString(s)` — parse YAML string (used in tests via `testenv.WithConfig()`)
- `config.DefaultTemplate()` — returns annotated YAML for `config init` command

## Config Precedence

flags > env vars (`OPENCLAW_DOCKER_*`) > config file (`--config`) > defaults

Config files are only loaded when `--config` / `-f` is explicitly passed. No auto-discovery.
