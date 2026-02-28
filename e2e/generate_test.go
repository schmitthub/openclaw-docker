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
		"compose/openclaw/entrypoint.sh",
		"compose/openclaw/openclaw.json",
		"compose/envoy/envoy.yaml",
		"compose/envoy/server-cert.pem",
		"compose/envoy/server-key.pem",
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
	allowedDirs := map[string]bool{"compose": true, "data": true}
	for _, entry := range entries {
		if entry.IsDir() && !allowedDirs[entry.Name()] {
			t.Errorf("unexpected subdirectory in output: %s", entry.Name())
		}
	}

	// Verify compose service subdirectories exist.
	for _, sub := range []string{"compose/openclaw", "compose/envoy"} {
		info, err := os.Stat(filepath.Join(outputDir, sub))
		if err != nil {
			t.Errorf("expected %s to exist: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}

	// Verify old squid/nginx directories do NOT exist.
	for _, sub := range []string{"compose/squid", "compose/nginx"} {
		if _, err := os.Stat(filepath.Join(outputDir, sub)); err == nil {
			t.Errorf("unexpected directory %s should not exist", sub)
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
		"npm install -g",
		`openclaw@${OPENCLAW_VERSION}`,
		"SHARP_IGNORE_GLOBAL_LIBVIPS=1",
		"/usr/local/bin/openclaw",
		"iptables",
		"gosu",
		"COPY entrypoint.sh /usr/local/bin/entrypoint.sh",
		`ENTRYPOINT ["entrypoint.sh"]`,
		`CMD ["openclaw", "gateway", "--allow-unconfigured"]`,
		"OPENCLAW_INSTALL_BROWSER",
		"playwright-core/cli.js",
		"xvfb",
	}
	for _, s := range mustContain {
		if !strings.Contains(body, s) {
			t.Errorf("Dockerfile missing expected content: %q", s)
		}
	}

	mustNotContain := []string{
		"firewall.sh",
		"alpine",
		"zsh-in-docker",
		"hadolint",
		"git-delta",
		"proxy-preload",
		"NODE_OPTIONS",
		"openclaw.ai/install.sh",
		"openclaw.mjs",
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

	for _, svc := range []string{"envoy:", "openclaw-gateway:"} {
		if !strings.Contains(body, svc) {
			t.Errorf("compose.yaml missing service %q", svc)
		}
	}

	for _, net := range []string{"openclaw-internal:", "openclaw-egress:"} {
		if !strings.Contains(body, net) {
			t.Errorf("compose.yaml missing network %q", net)
		}
	}

	if !strings.Contains(body, "dockerfile: Dockerfile") {
		t.Error("compose.yaml should reference local Dockerfile via build directive")
	}
	if strings.Contains(body, "image: ${OPENCLAW_IMAGE}") {
		t.Error("compose.yaml should not use image tag — should build from Dockerfile")
	}

	for _, s := range []string{
		"envoyproxy/envoy:",
		"compose/envoy/envoy.yaml:/etc/envoy/envoy.yaml",
		"compose/envoy/server-cert.pem:/etc/envoy/certs/server-cert.pem",
		"compose/envoy/server-key.pem:/etc/envoy/certs/server-key.pem",
		"443:443",
		"context: ./compose/openclaw",
		"internal: true",
		"cap_add:",
		"NET_ADMIN",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("compose.yaml missing expected content: %q", s)
		}
	}

	// Old squid/nginx artifacts and Node.js hacks should not be present.
	for _, s := range []string{"nginx:", "squid:", "NODE_EXTRA_CA_CERTS", "squid-log", "squid-cache", "NODE_OPTIONS", "proxy-preload"} {
		if strings.Contains(body, s) {
			t.Errorf("compose.yaml contains unexpected legacy content: %q", s)
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
		"HTTP_PROXY=http://envoy:10000",
		"HTTPS_PROXY=http://envoy:10000",
		"NO_PROXY=localhost,127.0.0.1,envoy,openclaw-gateway",
	}
	for _, v := range expectedVars {
		if !strings.Contains(body, v) {
			t.Errorf(".env.openclaw missing %q", v)
		}
	}

	// Dead vars and Node.js hacks should not be present.
	for _, v := range []string{"OPENCLAW_EXTRA_MOUNTS", "OPENCLAW_HOME_VOLUME", "NODE_OPTIONS", "proxy-preload"} {
		if strings.Contains(body, v) {
			t.Errorf(".env.openclaw contains unexpected legacy var: %q", v)
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

	if !strings.Contains(body, "up -d") {
		t.Error("setup.sh should start services")
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
		"compose/openclaw/entrypoint.sh",
		"compose/openclaw/openclaw.json",
		"compose/envoy/envoy.yaml",
		"compose/envoy/server-cert.pem",
		"compose/envoy/server-key.pem",
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

func TestGenerateEnvoyConfigContent(t *testing.T) {
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

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "envoy", "envoy.yaml"))
	if err != nil {
		t.Fatalf("read envoy.yaml: %v", err)
	}
	body := string(content)

	for _, s := range []string{
		"ingress",
		"egress",
		"port_value: 443",
		"port_value: 10000",
		"openclaw_gateway",
		"dynamic_forward_proxy",
		"websocket",
		"CONNECT",
		"Forbidden",
		"openclaw.ai:443",
		"api.anthropic.com:443",
		"api.openai.com:443",
		"generativelanguage.googleapis.com:443",
		"openrouter.ai:443",
		"api.x.ai:443",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("envoy.yaml missing expected content: %q", s)
		}
	}
}

func TestGenerateEnvoyAllowedDomains(t *testing.T) {
	h := &harness.Harness{T: t}
	setup := h.NewIsolatedFS()

	seedManifest(t, setup.BaseDir)
	outputDir := filepath.Join(setup.BaseDir, "deploy")

	result := h.Run(
		"generate",
		"--dangerous-inline",
		"--output", outputDir,
		"--allowed-domains", "api.anthropic.com,custom.example.com",
	)
	if result.Err != nil {
		t.Fatalf("generate failed: %v", result.Err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "compose", "envoy", "envoy.yaml"))
	if err != nil {
		t.Fatalf("read envoy.yaml: %v", err)
	}
	body := string(content)

	for _, domain := range []string{"api.anthropic.com:443", "custom.example.com:443", "openclaw.ai:443"} {
		if !strings.Contains(body, domain) {
			t.Errorf("envoy.yaml missing allowed domain: %q", domain)
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
		`"controlUi"`,
		`"allowedOrigins"`,
		`"https://localhost"`,
	} {
		if !strings.Contains(body, s) {
			t.Errorf("openclaw.json missing expected content: %q", s)
		}
	}
}

func TestGenerateTLSCertGeneration(t *testing.T) {
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

	certPath := filepath.Join(outputDir, "compose", "envoy", "server-cert.pem")
	keyPath := filepath.Join(outputDir, "compose", "envoy", "server-key.pem")

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read server-cert.pem: %v", err)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read server-key.pem: %v", err)
	}

	if !strings.Contains(string(certData), "BEGIN CERTIFICATE") {
		t.Error("server-cert.pem missing PEM certificate header")
	}
	if !strings.Contains(string(keyData), "BEGIN EC PRIVATE KEY") {
		t.Error("server-key.pem missing PEM EC private key header")
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
		t.Fatalf("read server-cert.pem after second generate: %v", err)
	}
	if string(certData) != string(certData2) {
		t.Error("TLS cert changed between generates — should be preserved")
	}
}

func TestGenerateEntrypointContent(t *testing.T) {
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

	path := filepath.Join(outputDir, "compose", "openclaw", "entrypoint.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("entrypoint.sh missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("entrypoint.sh should be executable")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read entrypoint.sh: %v", err)
	}
	body := string(content)

	for _, s := range []string{
		"#!/bin/bash",
		"iptables -P OUTPUT DROP",
		"127.0.0.11",
		"getent hosts envoy",
		"ESTABLISHED,RELATED",
		"gosu node",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("entrypoint.sh missing expected content: %q", s)
		}
	}
}
