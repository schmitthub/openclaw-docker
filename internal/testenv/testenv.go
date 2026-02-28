// Package testenv provides unified, progressively-configured test environments
// for isolated filesystem tests. It creates temp directories for all four XDG
// categories (config, data, state, cache), sets the corresponding CLAWKER_*_DIR
// env vars, and optionally wires up a real config.Config and/or ProjectManager.
//
// Usage:
//
//	// Just isolated dirs (storage tests):
//	env := testenv.New(t)
//	env.Dirs.Data // absolute path
//
//	// With real config (config, socketbridge tests):
//	env := testenv.New(t, testenv.WithConfig())
//	env.Config() // config.Config backed by temp dirs
//
//	// With real project manager (project tests):
//	env := testenv.New(t, testenv.WithProjectManager(nil))
//	env.ProjectManager() // project.ProjectManager
//	env.Config()         // also available — PM implies Config
package testenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schmitthub/openclaw-docker/internal/config"
)

// IsolatedDirs holds the four XDG-style directory paths created for the test.
type IsolatedDirs struct {
	Base  string // temp root (parent of all dirs)
	Cache string // CLAWKER_CACHE_DIR
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
//  1. Creates temp directories for config, data, state, and cache
//  2. Sets CLAWKER_CONFIG_DIR, CLAWKER_DATA_DIR, CLAWKER_STATE_DIR,
//     CLAWKER_CACHE_DIR env vars (restored on test cleanup)
//  3. Applies any options (WithConfig, WithProjectManager)
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
