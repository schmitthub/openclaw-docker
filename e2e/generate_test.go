package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schmitthub/openclaw-docker/e2e/harness"
)

const testManifest = `{
  "fullVersion": "2026.2.26",
  "version": {
    "major": 2026,
    "minor": 2,
    "patch": 26,
    "pre": null,
    "build": null
  }
}
`

// seedManifest writes a test manifest into the harness cache dir at the path
// the CLI expects and returns the versions-file path.
func seedManifest(t *testing.T, cacheDir string) string {
	t.Helper()
	manifestDir := filepath.Join(cacheDir, "openclaw-docker")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("seed manifest dir: %v", err)
	}
	path := filepath.Join(manifestDir, "versions.json")
	if err := os.WriteFile(path, []byte(testManifest), 0o644); err != nil {
		t.Fatalf("seed manifest write: %v", err)
	}
	return path
}

func TestRenderProducesAllFiles(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	for _, name := range []string{"Dockerfile", "compose.yaml", ".env.openclaw", "setup.sh"} {
		path := filepath.Join(outputDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("expected %s to be non-empty", name)
		}
	}
}

func TestRenderFlatOutput(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	// Dockerfile must be at the output root, not nested.
	if _, err := os.Stat(filepath.Join(outputDir, "Dockerfile")); err != nil {
		t.Fatal("Dockerfile not at output root")
	}

	// No version or variant subdirectories should exist.
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("unexpected subdirectory in output: %s", entry.Name())
		}
	}
}

func TestRenderDockerfileContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	body := string(content)

	mustContain := []string{
		"FROM node:22-bookworm",
		"OPENCLAW_VERSION=2026.2.26",
		"openclaw.ai/install.sh",
		"USER node",
		"CMD [\"node\", \"openclaw.mjs\", \"gateway\", \"--allow-unconfigured\"]",
	}
	for _, s := range mustContain {
		if !strings.Contains(body, s) {
			t.Errorf("Dockerfile missing expected content: %q", s)
		}
	}

	mustNotContain := []string{
		"firewall",
		"iptables",
		"entrypoint",
		"ENTRYPOINT",
		"alpine",
		"zsh-in-docker",
		"hadolint",
		"git-delta",
	}
	for _, s := range mustNotContain {
		if strings.Contains(body, s) {
			t.Errorf("Dockerfile contains unexpected content: %q", s)
		}
	}
}

func TestRenderDockerfileAptPackages(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
		"--docker-apt-packages", "git-lfs ripgrep",
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}

	if !strings.Contains(string(content), "git-lfs ripgrep") {
		t.Error("Dockerfile missing custom apt packages")
	}
}

func TestRenderComposeContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose.yaml"))
	if err != nil {
		t.Fatalf("read compose.yaml: %v", err)
	}
	body := string(content)

	for _, svc := range []string{"squid:", "openclaw-gateway:", "openclaw-cli:"} {
		if !strings.Contains(body, svc) {
			t.Errorf("compose.yaml missing service %q", svc)
		}
	}

	for _, net := range []string{"openclaw-internal:", "openclaw-egress:"} {
		if !strings.Contains(body, net) {
			t.Errorf("compose.yaml missing network %q", net)
		}
	}

	if !strings.Contains(body, "ubuntu/squid:latest") {
		t.Error("compose.yaml missing squid image reference")
	}

	if !strings.Contains(body, "dockerfile: Dockerfile") {
		t.Error("compose.yaml should reference local Dockerfile via build directive")
	}
	if strings.Contains(body, "image: ${OPENCLAW_IMAGE}") {
		t.Error("compose.yaml should not use image tag — should build from Dockerfile")
	}
}

func TestRenderEnvContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, ".env.openclaw"))
	if err != nil {
		t.Fatalf("read .env.openclaw: %v", err)
	}
	body := string(content)

	expectedVars := []string{
		"OPENCLAW_CONFIG_DIR=",
		"OPENCLAW_WORKSPACE_DIR=",
		"OPENCLAW_GATEWAY_PORT=",
		"OPENCLAW_BRIDGE_PORT=",
		"OPENCLAW_GATEWAY_BIND=",
		"OPENCLAW_HTTP_PROXY=http://squid:3128",
		"OPENCLAW_HTTPS_PROXY=http://squid:3128",
		"OPENCLAW_NO_PROXY=",
	}
	for _, v := range expectedVars {
		if !strings.Contains(body, v) {
			t.Errorf(".env.openclaw missing %q", v)
		}
	}
}

func TestRenderSetupScriptExecutable(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	info, err := os.Stat(filepath.Join(outputDir, "setup.sh"))
	if err != nil {
		t.Fatalf("stat setup.sh: %v", err)
	}

	if info.Mode()&0o111 == 0 {
		t.Error("setup.sh is not executable")
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "setup.sh"))
	if err != nil {
		t.Fatalf("read setup.sh: %v", err)
	}
	body := string(content)

	if !strings.HasPrefix(body, "#!/usr/bin/env bash") {
		t.Error("setup.sh missing shebang")
	}

	for _, s := range []string{"docker compose", "build", "openssl rand", "OPENCLAW_GATEWAY_TOKEN"} {
		if !strings.Contains(body, s) {
			t.Errorf("setup.sh missing expected content: %q", s)
		}
	}
}

func TestRenderCustomOptions(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
		"--openclaw-gateway-port", "9999",
		"--openclaw-gateway-bind", "loopback",
	)
	if result.Err != nil {
		t.Fatalf("render failed: %v", result.Err)
	}

	dockerfile, _ := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))

	if !strings.Contains(string(dockerfile), "OPENCLAW_GATEWAY_PORT=9999") {
		t.Error("Dockerfile missing custom gateway port")
	}
	if !strings.Contains(string(dockerfile), "OPENCLAW_GATEWAY_BIND=loopback") {
		t.Error("Dockerfile missing custom gateway bind")
	}
}

func TestRenderIdempotent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	versionsFile := seedManifest(t, setup.CacheDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	args := []string{
		"render",
		"--dangerous-inline",
		"--versions-file", versionsFile,
		"--output", outputDir,
	}

	result := h.Run(args...)
	if result.Err != nil {
		t.Fatalf("first render failed: %v", result.Err)
	}

	first, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read first Dockerfile: %v", err)
	}

	// Run again — should overwrite with identical content.
	result = h.Run(args...)
	if result.Err != nil {
		t.Fatalf("second render failed: %v", result.Err)
	}

	second, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read second Dockerfile: %v", err)
	}

	if string(first) != string(second) {
		t.Error("render is not idempotent — Dockerfile content differs between runs")
	}
}

func TestGenerateFullPipeline(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not in PATH; skipping full generate test")
	}

	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--openclaw-version", "latest",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	// All four artifacts should exist.
	for _, name := range []string{"Dockerfile", "compose.yaml", ".env.openclaw", "setup.sh"} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("expected %s to exist after generate: %v", name, err)
		}
	}

	// Manifest should have been written to the cache dir.
	manifestPath := filepath.Join(setup.CacheDir, "openclaw-docker", "versions.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("expected manifest at %s: %v", manifestPath, err)
	}

	// Dockerfile should contain a real resolved version (not empty or "latest").
	content, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	if strings.Contains(string(content), "OPENCLAW_VERSION=latest") {
		t.Error("generate should resolve 'latest' to a concrete version, not embed the tag")
	}
	if !strings.Contains(string(content), "OPENCLAW_VERSION=") {
		t.Error("Dockerfile missing OPENCLAW_VERSION")
	}
}
