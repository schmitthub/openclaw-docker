package harness

import (
	"os"
	"testing"

	"github.com/schmitthub/openclaw-docker/internal/cmd"
	"github.com/schmitthub/openclaw-docker/internal/testenv"
)

// Harness provides an isolated filesystem environment for integration tests.
// It delegates to testenv for directory creation and env var setup, then
// provides CLI execution via Run().
type Harness struct {
	T *testing.T
}

// RunResult holds the outcome of a CLI command execution.
type RunResult struct {
	ExitCode int
	Err      error
}

// SetupResult holds the resolved paths from NewIsolatedFS.
type SetupResult struct {
	BaseDir  string
	CacheDir string
}

// NewIsolatedFS creates an isolated test environment.
func (h *Harness) NewIsolatedFS() *SetupResult {
	h.T.Helper()

	env := testenv.New(h.T)

	cacheDir := env.Dirs.Cache

	for _, dir := range []string{cacheDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			h.T.Fatalf("harness: creating dir %s: %v", dir, err)
		}
	}

	return &SetupResult{
		BaseDir:  env.Dirs.Base,
		CacheDir: env.Dirs.Cache,
	}
}

// Chdir changes the working directory and registers a cleanup to restore it
// to BaseDir when the test ends.
func (r *SetupResult) Chdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("harness: chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(r.BaseDir) })
}

// Run executes a CLI command through the full cmd.NewRootCmd Cobra pipeline.
func (h *Harness) Run(args ...string) *RunResult {
	h.T.Helper()

	rootCmd := cmd.NewRootCmd("test", "test")

	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	return &RunResult{ExitCode: exitCode, Err: err}
}
