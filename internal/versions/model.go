package versions

type SemverParts struct {
	Major int     `json:"major"`
	Minor int     `json:"minor"`
	Patch int     `json:"patch"`
	Pre   *string `json:"pre"`
	Build *string `json:"build"`
}

type ReleaseMeta struct {
	FullVersion string      `json:"fullVersion"`
	Version     SemverParts `json:"version"`
}

type ResolveOptions struct {
	Requested string
	Debug     bool
}
