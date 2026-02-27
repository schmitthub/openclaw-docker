package versions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteManifest(path string, manifest Manifest) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("{\n")

	for index, version := range manifest.Order {
		meta, ok := manifest.Entries[version]
		if !ok {
			continue
		}

		key, err := json.Marshal(version)
		if err != nil {
			return fmt.Errorf("encode manifest key: %w", err)
		}

		value, err := json.MarshalIndent(meta, "  ", "  ")
		if err != nil {
			return fmt.Errorf("encode manifest value: %w", err)
		}

		buf.WriteString("  ")
		buf.Write(key)
		buf.WriteString(": ")
		buf.Write(value)
		if index < len(manifest.Order)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("}\n")

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func ReadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	entries := make(map[string]ReleaseMeta)
	if err := json.Unmarshal(raw, &entries); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest JSON: %w", err)
	}

	order := make([]string, 0, len(entries))
	for key := range entries {
		order = append(order, key)
	}
	sortVersionsDesc(order)

	return Manifest{Order: order, Entries: entries}, nil
}
