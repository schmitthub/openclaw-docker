package versions

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	semver "github.com/Masterminds/semver/v3"
)

var partialSemverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*)(?:-([^\+]+))?(?:\+(.*))?)?)?$`)

type targetSemver struct {
	major int
	minor *int
	patch *int
	pre   *string
}

func parseTargetSemver(raw string) (targetSemver, error) {
	m := partialSemverPattern.FindStringSubmatch(raw)
	if m == nil {
		return targetSemver{}, fmt.Errorf("invalid semver target %q", raw)
	}

	major, _ := strconv.Atoi(m[1])
	var minorPtr *int
	if m[2] != "" {
		minor, _ := strconv.Atoi(m[2])
		minorPtr = &minor
	}

	var patchPtr *int
	if m[3] != "" {
		patch, _ := strconv.Atoi(m[3])
		patchPtr = &patch
	}

	var prePtr *string
	if m[4] != "" {
		pre := m[4]
		prePtr = &pre
	}

	return targetSemver{major: major, minor: minorPtr, patch: patchPtr, pre: prePtr}, nil
}

func parseVersion(raw string) (*semver.Version, error) {
	parsed, err := semver.NewVersion(raw)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func matchSemver(target string, candidates []string) (string, bool) {
	for _, value := range candidates {
		if value == target {
			return target, true
		}
	}

	targetSemver, err := parseTargetSemver(target)
	if err != nil {
		return "", false
	}

	matches := make([]*semver.Version, 0)
	for _, raw := range candidates {
		parsed, err := parseVersion(raw)
		if err != nil {
			continue
		}

		if int(parsed.Major()) != targetSemver.major {
			continue
		}
		if targetSemver.minor != nil && int(parsed.Minor()) != *targetSemver.minor {
			continue
		}
		if targetSemver.patch != nil && int(parsed.Patch()) != *targetSemver.patch {
			continue
		}

		if parsed.Prerelease() != "" {
			continue
		}

		matches = append(matches, parsed)
	}

	if len(matches) == 0 {
		return "", false
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].LessThan(matches[j])
	})

	return matches[len(matches)-1].Original(), true
}

func toSemverParts(raw string) (SemverParts, error) {
	v, err := parseVersion(raw)
	if err != nil {
		return SemverParts{}, err
	}

	var pre *string
	if value := v.Prerelease(); value != "" {
		pre = &value
	}

	var build *string
	if value := v.Metadata(); value != "" {
		build = &value
	}

	return SemverParts{
		Major: int(v.Major()),
		Minor: int(v.Minor()),
		Patch: int(v.Patch()),
		Pre:   pre,
		Build: build,
	}, nil
}
