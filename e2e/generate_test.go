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

// seedManifest writes a test manifest to a temp file and sets the
// OPENCLAW_DOCKER_VERSIONS_FILE env var so generate reads it instead of
// resolving from npm. Returns the manifest path.
func seedManifest(t *testing.T, baseDir string) string {
	t.Helper()
	path := filepath.Join(baseDir, "manifest.json")
	if err := os.WriteFile(path, []byte(testManifest), 0o644); err != nil {
		t.Fatalf("seed manifest write: %v", err)
	}
	t.Setenv("OPENCLAW_DOCKER_VERSIONS_FILE", path)
	return path
}

func TestGenerateProducesAllFiles(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	for _, name := range []string{
		"compose/openclaw/Dockerfile",
		"compose/openclaw/openclaw.json",
		"compose/squid/Dockerfile.squid",
		"compose/squid/squid.conf",
		"compose/squid/ca-cert.pem",
		"compose/squid/ca-key.pem",
		"compose/nginx/nginx.conf",
		"compose/nginx/nginx-cert.pem",
		"compose/nginx/nginx-key.pem",
		"compose.yaml",
		".env.openclaw",
		"setup.sh",
		"manifest.json",
	} {
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

func TestGenerateOutputStructure(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	// Only expected subdirectory is compose/.
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "compose" {
			t.Errorf("unexpected subdirectory in output: %s", entry.Name())
		}
	}

	// Verify compose service subdirectories exist.
	for _, sub := range []string{"compose/openclaw", "compose/squid", "compose/nginx"} {
		info, err := os.Stat(filepath.Join(outputDir, sub))
		if err != nil {
			t.Errorf("expected %s to exist: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}
}

func TestGenerateDockerfileContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))
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

func TestGenerateDockerfileAptPackages(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
		"--docker-apt-packages", "git-lfs ripgrep",
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}

	if !strings.Contains(string(content), "git-lfs ripgrep") {
		t.Error("Dockerfile missing custom apt packages")
	}
}

func TestGenerateComposeContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose.yaml"))
	if err != nil {
		t.Fatalf("read compose.yaml: %v", err)
	}
	body := string(content)

	for _, svc := range []string{"nginx:", "squid:", "openclaw-gateway:"} {
		if !strings.Contains(body, svc) {
			t.Errorf("compose.yaml missing service %q", svc)
		}
	}

	for _, net := range []string{"openclaw-internal:", "openclaw-egress:"} {
		if !strings.Contains(body, net) {
			t.Errorf("compose.yaml missing network %q", net)
		}
	}

	if !strings.Contains(body, "dockerfile: Dockerfile.squid") {
		t.Error("compose.yaml missing Dockerfile.squid build reference for squid")
	}

	if !strings.Contains(body, "dockerfile: Dockerfile") {
		t.Error("compose.yaml should reference local Dockerfile via build directive")
	}
	if strings.Contains(body, "image: ${OPENCLAW_IMAGE}") {
		t.Error("compose.yaml should not use image tag — should build from Dockerfile")
	}

	for _, s := range []string{
		"nginx:alpine",
		"compose/nginx/nginx.conf:/etc/nginx/conf.d/default.conf",
		"compose/nginx/nginx-cert.pem:/etc/nginx/certs/server.pem",
		"compose/nginx/nginx-key.pem:/etc/nginx/certs/server-key.pem",
		"443:443",
		"compose/squid/squid.conf:/etc/squid/squid.conf",
		"compose/squid/ca-cert.pem:/etc/squid/ca-cert.pem",
		"compose/squid/ca-key.pem:/etc/squid/ca-key.pem",
		"NODE_EXTRA_CA_CERTS",
		"compose/squid/ca-cert.pem:/etc/ssl/certs/openclaw-ca.pem",
		"context: ./compose/squid",
		"context: ./compose/openclaw",
		"squid-log:",
		"squid-cache:",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("compose.yaml missing expected content: %q", s)
		}
	}
}

func TestGenerateEnvContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
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

func TestGenerateSetupScriptExecutable(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
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

func TestGenerateCustomOptions(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
		"--openclaw-gateway-port", "9999",
		"--openclaw-gateway-bind", "loopback",
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	dockerfile, _ := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))

	if !strings.Contains(string(dockerfile), "OPENCLAW_GATEWAY_PORT=9999") {
		t.Error("Dockerfile missing custom gateway port")
	}
	if !strings.Contains(string(dockerfile), "OPENCLAW_GATEWAY_BIND=loopback") {
		t.Error("Dockerfile missing custom gateway bind")
	}
}

func TestGenerateIdempotent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	args := []string{
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	}

	result := h.Run(args...)
	if result.Err != nil {
		t.Fatalf("first generate failed: %v", result.Err)
	}

	first, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))
	if err != nil {
		t.Fatalf("read first Dockerfile: %v", err)
	}

	// Run again — should overwrite with identical content.
	result = h.Run(args...)
	if result.Err != nil {
		t.Fatalf("second generate failed: %v", result.Err)
	}

	second, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))
	if err != nil {
		t.Fatalf("read second Dockerfile: %v", err)
	}

	if string(first) != string(second) {
		t.Error("generate is not idempotent — Dockerfile content differs between runs")
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

	// All artifacts should exist including manifest.json in output dir.
	for _, name := range []string{
		"compose/openclaw/Dockerfile",
		"compose/openclaw/openclaw.json",
		"compose/squid/Dockerfile.squid",
		"compose/squid/squid.conf",
		"compose/squid/ca-cert.pem",
		"compose/squid/ca-key.pem",
		"compose/nginx/nginx.conf",
		"compose/nginx/nginx-cert.pem",
		"compose/nginx/nginx-key.pem",
		"compose.yaml",
		".env.openclaw",
		"setup.sh",
		"manifest.json",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("expected %s to exist after generate: %v", name, err)
		}
	}

	// Dockerfile should contain a real resolved version (not empty or "latest").
	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "Dockerfile"))
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

func TestGenerateSquidConfContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "squid", "squid.conf"))
	if err != nil {
		t.Fatalf("read squid.conf: %v", err)
	}
	body := string(content)

	for _, s := range []string{
		"http_port 3128",
		"ssl-bump",
		"sslcrtd_program",
		"deny all",
		"openclaw.ai",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("squid.conf missing expected content: %q", s)
		}
	}
}

func TestGenerateSquidConfAllowedDomains(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
		"--squid-allowed-domains", "api.anthropic.com,api.openai.com",
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "squid", "squid.conf"))
	if err != nil {
		t.Fatalf("read squid.conf: %v", err)
	}
	body := string(content)

	for _, domain := range []string{"api.anthropic.com", "api.openai.com", "openclaw.ai"} {
		if !strings.Contains(body, domain) {
			t.Errorf("squid.conf missing allowed domain: %q", domain)
		}
	}
}

func TestGenerateOpenClawJSONContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "openclaw", "openclaw.json"))
	if err != nil {
		t.Fatalf("read openclaw.json: %v", err)
	}
	body := string(content)

	for _, s := range []string{
		`"gateway"`,
		`"mode"`,
		`"local"`,
		`"bind"`,
		`"auth"`,
		`"token"`,
		"__GATEWAY_TOKEN__",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("openclaw.json missing expected content: %q", s)
		}
	}
}

func TestGenerateCAGeneration(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	certPath := filepath.Join(outputDir, "compose", "squid", "ca-cert.pem")
	keyPath := filepath.Join(outputDir, "compose", "squid", "ca-key.pem")

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read ca-cert.pem: %v", err)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read ca-key.pem: %v", err)
	}

	if !strings.Contains(string(certData), "BEGIN CERTIFICATE") {
		t.Error("ca-cert.pem missing PEM certificate header")
	}
	if !strings.Contains(string(keyData), "BEGIN EC PRIVATE KEY") {
		t.Error("ca-key.pem missing PEM EC private key header")
	}

	// Re-run should preserve the same cert (idempotency).
	result = h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("second generate failed: %v", result.Err)
	}

	certData2, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read ca-cert.pem after second generate: %v", err)
	}
	if string(certData) != string(certData2) {
		t.Error("CA cert changed between generates — should be preserved")
	}
}

func TestGenerateNginxConfContent(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "nginx", "nginx.conf"))
	if err != nil {
		t.Fatalf("read nginx.conf: %v", err)
	}
	body := string(content)

	for _, s := range []string{
		"upstream openclaw_gateway",
		"proxy_pass http://openclaw_gateway",
		"ssl_certificate",
		"ssl_certificate_key",
		"Upgrade",
		"proxy_http_version 1.1",
		"ssl_client_certificate",
		"ssl_verify_client",
		"proxy_read_timeout",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("nginx.conf missing expected content: %q", s)
		}
	}
}

func TestGenerateNginxCertGeneration(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	certPath := filepath.Join(outputDir, "compose", "nginx", "nginx-cert.pem")
	keyPath := filepath.Join(outputDir, "compose", "nginx", "nginx-key.pem")

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read nginx-cert.pem: %v", err)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read nginx-key.pem: %v", err)
	}

	if !strings.Contains(string(certData), "BEGIN CERTIFICATE") {
		t.Error("nginx-cert.pem missing PEM certificate header")
	}
	if !strings.Contains(string(keyData), "BEGIN EC PRIVATE KEY") {
		t.Error("nginx-key.pem missing PEM EC private key header")
	}

	// Re-run should preserve the same cert (idempotency).
	result = h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
	)
	if result.Err != nil {
		t.Fatalf("second generate failed: %v", result.Err)
	}

	certData2, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read nginx-cert.pem after second generate: %v", err)
	}
	if string(certData) != string(certData2) {
		t.Error("nginx cert changed between generates — should be preserved")
	}
}
