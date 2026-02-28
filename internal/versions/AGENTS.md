# Package: `internal/versions`

npm version resolution, semver matching, and manifest I/O.

## Files

| File | Purpose |
|------|---------|
| `model.go` | `ReleaseMeta`, `SemverParts`, `ResolveOptions` types |
| `resolve.go` | `Resolve()` — npm dist-tag or semver partial resolution |
| `semver.go` | `matchSemver()` — partial semver matching against candidates |
| `io.go` | `WriteManifest()` / `ReadManifest()` — JSON manifest I/O |

## Resolution Flow

1. `Resolve(ctx, ResolveOptions{Requested: "latest"})` called
2. Fetches `npm view openclaw versions --json` and `npm view openclaw dist-tags --json`
3. If `Requested` matches a dist-tag key, uses that tag's version
4. Otherwise, `matchSemver()` finds best match from all published versions
5. Returns `ReleaseMeta{FullVersion, Version}` with parsed semver parts

## Manifest Format (`versions.json`)

```json
{
  "fullVersion": "2026.2.26",
  "version": {
    "major": 2026, "minor": 2, "patch": 26,
    "pre": null, "build": null
  }
}
```

## Semver Matching

`matchSemver(target, candidates)` supports partial matching:
- `"2026"` — matches highest `2026.x.x`
- `"2026.2"` — matches highest `2026.2.x`
- `"2026.2.26"` — exact match
- Excludes prereleases unless target has prerelease suffix
