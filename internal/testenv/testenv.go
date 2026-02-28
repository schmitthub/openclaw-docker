// Package testenv provides isolated test environments with temp directories
// and environment variable overrides. It creates a cache directory and sets
// OPENCLAW_DOCKER_CACHE_DIR (restored on test cleanup).
//
// Usage:
//
//	// Isolated dirs:
//	env := testenv.New(t)
//	env.Dirs.Base  // temp root
//	env.Dirs.Cache // cache dir (OPENCLAW_DOCKER_CACHE_DIR)
//
//	// With config:
//	env := testenv.New(t, testenv.WithConfig(yamlString))
//	env.Config // *config.FileConfig
package testenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schmitthub/openclaw-docker/internal/config"
)

// IsolatedDirs holds the directory paths created for the test.
type IsolatedDirs struct {
	Base   string // temp root (parent of all dirs)
	Cache  string //
	Deploy string //
}

// Env is a unified test environment with isolated directories and optional
// higher-level capabilities (config, project manager).
type Env struct {
	Dirs   IsolatedDirs
	Config *config.FileConfig
}

// Option configures an Env during construction.
type Option func(t *testing.T, e *Env)

// WithConfig creates a real config.Config backed by the isolated directories.
func WithConfig(yaml string) Option {
	return func(t *testing.T, e *Env) {
		t.Helper()
		cfg, err := config.FromString(yaml)
		if err != nil {
			t.Fatalf("testenv: creating config: %v", err)
		}
		e.Config = &cfg
	}
}

// New creates an isolated test environment. It:
//  1. Creates a temp directory with a cache subdirectory
//  2. Sets OPENCLAW_DOCKER_CACHE_DIR env var (restored on test cleanup)
//  3. Applies any options (e.g. WithConfig)
func New(t *testing.T, opts ...Option) *Env {
	t.Helper()

	// Resolve symlinks on the base temp dir so paths match os.Getwd()
	// after chdir (macOS: /var → /private/var).
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("testenv: resolving temp dir symlinks: %v", err)
	}

	dirs := IsolatedDirs{
		Base:  base,
		Cache: filepath.Join(base, "cache"),
	}

	for _, dir := range []string{dirs.Cache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("testenv: creating dir %s: %v", dir, err)
		}
	}

	t.Setenv("OPENCLAW_DOCKER_CACHE_DIR", dirs.Cache)

	env := &Env{Dirs: dirs}

	for _, opt := range opts {
		opt(t, env)
	}

	return env
}
