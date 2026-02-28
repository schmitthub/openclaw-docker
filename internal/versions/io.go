package versions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteManifest(path string, meta ReleaseMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func ReadManifest(path string) (ReleaseMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ReleaseMeta{}, fmt.Errorf("read manifest: %w", err)
	}

	var meta ReleaseMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ReleaseMeta{}, fmt.Errorf("parse manifest JSON: %w", err)
	}

	return meta, nil
}
