package versions

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const npmPackageName = "openclaw"

func Resolve(ctx context.Context, opts ResolveOptions) (ReleaseMeta, error) {
	requested := strings.TrimSpace(opts.Requested)
	if requested == "" {
		requested = "latest"
	}

	allVersions, err := npmVersions(ctx, npmPackageName)
	if err != nil {
		return ReleaseMeta{}, err
	}

	distTags, err := npmDistTags(ctx, npmPackageName)
	if err != nil {
		return ReleaseMeta{}, err
	}

	resolved := ""
	if distTag, ok := distTags[requested]; ok && distTag != "" {
		resolved = distTag
	} else {
		matched, ok := matchSemver(requested, allVersions)
		if !ok {
			return ReleaseMeta{}, fmt.Errorf("cannot find version matching %q", requested)
		}
		resolved = matched
	}

	parts, err := toSemverParts(resolved)
	if err != nil {
		return ReleaseMeta{}, fmt.Errorf("parse resolved version %q: %w", resolved, err)
	}

	return ReleaseMeta{
		FullVersion: resolved,
		Version:     parts,
	}, nil
}

func npmVersions(ctx context.Context, pkg string) ([]string, error) {
	out, err := runNpmView(ctx, pkg, "versions")
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(strings.TrimSpace(out), "[") {
		var versions []string
		if err := json.Unmarshal([]byte(out), &versions); err != nil {
			return nil, fmt.Errorf("decode npm versions array: %w", err)
		}
		return versions, nil
	}

	var one string
	if err := json.Unmarshal([]byte(out), &one); err != nil {
		return nil, fmt.Errorf("decode npm versions string: %w", err)
	}
	if one == "" {
		return nil, fmt.Errorf("npm returned empty versions")
	}
	return []string{one}, nil
}

func npmDistTags(ctx context.Context, pkg string) (map[string]string, error) {
	out, err := runNpmView(ctx, pkg, "dist-tags")
	if err != nil {
		return nil, err
	}

	var tags map[string]string
	if err := json.Unmarshal([]byte(out), &tags); err != nil {
		return nil, fmt.Errorf("decode npm dist-tags: %w", err)
	}

	return tags, nil
}

func runNpmView(ctx context.Context, pkg, field string) (string, error) {
	cmd := exec.CommandContext(ctx, "npm", "view", pkg, field, "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("npm view %s %s --json: %w (%s)", pkg, field, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
