package harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schmitthub/openclaw-docker/internal/cmd"
	"github.com/schmitthub/openclaw-docker/internal/testenv"
)

// Harness provides an isolated filesystem environment for integration tests.
// It creates temp directories, sets XDG env vars, registers a project, and
// optionally persists config.
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
	BaseDir    string
	ProjectDir string
	ConfigDir  string
	DataDir    string
	StateDir   string
	CacheDir   string
}

// FSOptions allows overriding the project directory name.
type FSOptions struct {
	ProjectDir string // subdirectory name under base (default: "testproject")
}

// NewIsolatedFS creates an isolated test environment.
//
// Delegates XDG directory setup to testenv.New, then adds a project directory
// and chdirs into it (restored on cleanup).
func (h *Harness) NewIsolatedFS(opts *FSOptions) *SetupResult {
	h.T.Helper()

	if opts == nil {
		opts = &FSOptions{}
	}
	if opts.ProjectDir == "" {
		opts.ProjectDir = "testproject"
	}

	env := testenv.New(h.T)

	configDir := filepath.Join(env.Dirs.Base, "config")
	dataDir := filepath.Join(env.Dirs.Base, "data")
	stateDir := filepath.Join(env.Dirs.Base, "state")
	for _, dir := range []string{configDir, dataDir, stateDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			h.T.Fatalf("harness: creating dir %s: %v", dir, err)
		}
	}

	projectDir := filepath.Join(env.Dirs.Base, opts.ProjectDir)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		h.T.Fatalf("harness: creating project dir %s: %v", projectDir, err)
	}

	// Chdir to project directory so config discovery works from CWD.
	prevDir, err := os.Getwd()
	if err != nil {
		h.T.Fatalf("harness: getting cwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		h.T.Fatalf("harness: chdir to project dir: %v", err)
	}
	h.T.Cleanup(func() {
		_ = os.Chdir(prevDir)
	})

	return &SetupResult{
		BaseDir:    env.Dirs.Base,
		ProjectDir: projectDir,
		ConfigDir:  configDir,
		DataDir:    dataDir,
		StateDir:   stateDir,
		CacheDir:   env.Dirs.Cache,
	}
}

// Chdir changes the working directory and registers a cleanup to restore it
// to ProjectDir when the test ends.
func (r *SetupResult) Chdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("harness: chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(r.ProjectDir) })
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
