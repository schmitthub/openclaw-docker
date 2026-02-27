package versions

type SemverParts struct {
	Major int     `json:"major"`
	Minor int     `json:"minor"`
	Patch int     `json:"patch"`
	Pre   *string `json:"pre"`
	Build *string `json:"build"`
}

type ReleaseMeta struct {
	FullVersion   string              `json:"fullVersion"`
	Version       SemverParts         `json:"version"`
	DebianDefault string              `json:"debianDefault"`
	AlpineDefault string              `json:"alpineDefault"`
	Variants      map[string][]string `json:"variants"`
}

type Manifest struct {
	Order   []string
	Entries map[string]ReleaseMeta
}

type ResolveOptions struct {
	Requested     []string
	DebianDefault string
	AlpineDefault string
	Variants      map[string][]string
	Arches        []string
	Debug         bool
}
