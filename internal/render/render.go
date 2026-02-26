package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/schmitthub/openclaw-docker/internal/versions"
)

type Options struct {
	Manifest     versions.Manifest
	OutputDir    string
	TemplatesDir string
	Cleanup      bool
	Requested    []string
}

func Generate(opts Options) error {
	if opts.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	selected := selectVersions(opts.Manifest, opts.Requested)
	selectedSet := make(map[string]struct{}, len(selected))

	for _, version := range selected {
		selectedSet[version] = struct{}{}
		meta := opts.Manifest.Entries[version]

		variantNames := make([]string, 0, len(meta.Variants))
		for variant := range meta.Variants {
			variantNames = append(variantNames, variant)
		}
		sort.Strings(variantNames)

		for _, variant := range variantNames {
			target := filepath.Join(opts.OutputDir, version, variant)
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create output path %q: %w", target, err)
			}

			dockerfilePath := filepath.Join(target, "Dockerfile")
			content := dockerfileFor(meta, variant, opts.TemplatesDir)
			if err := os.WriteFile(dockerfilePath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write dockerfile %q: %w", dockerfilePath, err)
			}
		}
	}

	if opts.Cleanup {
		if err := cleanupObsolete(opts.OutputDir, selectedSet); err != nil {
			return err
		}
	}

	return nil
}

func selectVersions(manifest versions.Manifest, requested []string) []string {
	if len(requested) == 0 {
		return append([]string(nil), manifest.Order...)
	}

	selection := make([]string, 0, len(requested))
	seen := make(map[string]struct{})
	for _, version := range requested {
		if _, ok := manifest.Entries[version]; !ok {
			continue
		}
		if _, dup := seen[version]; dup {
			continue
		}
		seen[version] = struct{}{}
		selection = append(selection, version)
	}

	if len(selection) > 0 {
		ordered := make([]string, 0, len(selection))
		selectionSet := make(map[string]struct{}, len(selection))
		for _, value := range selection {
			selectionSet[value] = struct{}{}
		}
		for _, value := range manifest.Order {
			if _, ok := selectionSet[value]; ok {
				ordered = append(ordered, value)
			}
		}
		selection = ordered
	}

	if len(selection) == 0 {
		return append([]string(nil), manifest.Order...)
	}

	return selection
}

func cleanupObsolete(outputDir string, keep map[string]struct{}) error {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("read output directory for cleanup: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, ok := keep[entry.Name()]; ok {
			continue
		}

		if err := os.RemoveAll(filepath.Join(outputDir, entry.Name())); err != nil {
			return fmt.Errorf("remove obsolete output dir %q: %w", entry.Name(), err)
		}
	}

	return nil
}

func dockerfileFor(meta versions.ReleaseMeta, variant, templatesDir string) string {
	isAlpine := strings.HasPrefix(variant, "alpine")
	alpineVersion := strings.TrimPrefix(variant, "alpine")

	baseImage := fmt.Sprintf("buildpack-deps:%s-scm", meta.DebianDefault)
	if isAlpine {
		baseImage = fmt.Sprintf("alpine:%s", alpineVersion)
	}

	var packageBlock, dockerCLIBlock, localeBlock, userBlock, deltaBlock, hadolintBlock, fzfBlock string
	var dockerGroupBlock string

	if isAlpine {
		packageBlock = `# Alpine pkgs
RUN apk add --no-cache \
  bash \
  less \
  git \
  procps \
  sudo \
  fzf \
  zsh \
  man-db \
  unzip \
  gnupg \
  iptables \
  ipset \
  iproute2 \
  bind-tools \
  jq \
  nano \
  vim \
  wget \
  curl \
  github-cli \
  musl-locales \
  musl-locales-lang \
  && rm -rf /var/cache/apk/*`

		dockerCLIBlock = `# Install Docker CLI (not daemon - we use host's Docker via socket mount)
RUN apk add --no-cache \
  docker-cli \
  docker-cli-buildx \
  docker-cli-compose`

		localeBlock = `# Alpine uses musl-locales, no locale-gen needed`

		userBlock = `RUN addgroup -g 1001 ${USERNAME} \
  && adduser -D -u 1001 -G ${USERNAME} -s /bin/zsh ${USERNAME}`

		deltaBlock = `RUN apk add --no-cache delta`

		hadolintBlock = `RUN ARCH=$(uname -m) && \
  if [ "$ARCH" = "x86_64" ]; then ARCH="x86_64"; \
  elif [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi && \
  wget "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-${ARCH}" -O /usr/local/bin/hadolint && \
  chmod +x /usr/local/bin/hadolint`

		fzfBlock = `  -a "source /usr/share/fzf/key-bindings.zsh" \
  -a "source /usr/share/fzf/completion.zsh" \
  -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  -x`

		dockerGroupBlock = `RUN addgroup -g 999 docker 2>/dev/null || addgroup docker 2>/dev/null || true && adduser ${USERNAME} docker`
	} else {
		packageBlock = `# TODO figure out which of these are already installed in buildpack-deps
RUN apt-get update && apt-get install -y --no-install-recommends \
  less \
  git \
  procps \
  sudo \
  fzf \
  zsh \
  man-db \
  unzip \
  gnupg2 \
  iptables \
  ipset \
  iproute2 \
  dnsutils \
  aggregate \
  jq \
  nano \
  vim \
  wget \
  curl \
  gh \
  locales \
  locales-all \
  && apt-get clean && rm -rf /var/lib/apt/lists/*`

		dockerCLIBlock = `# Install Docker CLI (not daemon - we use host's Docker via socket mount)
RUN install -m 0755 -d /etc/apt/keyrings && \
  curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && \
  chmod a+r /etc/apt/keyrings/docker.asc && \
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" > /etc/apt/sources.list.d/docker.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends docker-ce-cli docker-buildx-plugin docker-compose-plugin && \
  apt-get clean && rm -rf /var/lib/apt/lists/*`

		localeBlock = `RUN locale-gen en_US.UTF-8`

		userBlock = `RUN groupadd --gid 1001 ${USERNAME} \
  && useradd --uid 1001 --gid ${USERNAME} --shell /bin/zsh --create-home ${USERNAME}`

		deltaBlock = `ARG GIT_DELTA_VERSION=0.18.2
RUN ARCH=$(dpkg --print-architecture) && \
  wget "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"`

		hadolintBlock = `RUN ARCH=$(dpkg --print-architecture) && \
  if [ "$ARCH" = "amd64" ]; then ARCH="x86_64"; fi && \
  wget "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-${ARCH}" -O /usr/local/bin/hadolint && \
  chmod +x /usr/local/bin/hadolint`

		fzfBlock = `  -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
  -a "source /usr/share/doc/fzf/examples/completion.zsh" \
  -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  -x`

		dockerGroupBlock = `RUN groupadd -g 999 docker 2>/dev/null || groupadd docker 2>/dev/null || true && usermod -aG docker ${USERNAME}`
	}

	return fmt.Sprintf(`#
# NOTE: THIS DOCKERFILE IS GENERATED VIA "openclaw-docker"
#
# PLEASE DO NOT EDIT IT DIRECTLY.
#

FROM %s AS build

ARG TZ
ENV TZ="$TZ"

ENV OPENCLAW_VERSION=%s

%s

%s

%s

ARG USERNAME=openclaw

# Setup OpenClaw user
%s

# Create docker group and add user for socket access
%s

# Create workspace and config directories and set permissions
RUN mkdir -p /home/${USERNAME}/.openclaw /workspace && \
  chown -R ${USERNAME}:${USERNAME} /workspace /home/${USERNAME}/.openclaw

# Persist bash history.
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  && mkdir /commandhistory \
  && touch /commandhistory/.bash_history \
  && chown -R $USERNAME /commandhistory

WORKDIR /workspace

# Install git-delta
%s

# Install hadolint (Dockerfile linter)
ARG HADOLINT_VERSION=2.12.0
%s

# Set up non-root user
USER ${USERNAME}

# Set environment config
ENV SHELL=/bin/zsh
ENV EDITOR=nano
ENV VISUAL=nano
ENV PATH=/home/${USERNAME}/.local/bin:$PATH

# Use zsh-in-docker to configure plugins and theme
ARG ZSH_IN_DOCKER_VERSION=1.2.0
ARG ZSH_THEME=default
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
  -t ${ZSH_THEME} \
  -p git \
  -p fzf \
%s

ARG TEMPLATE_DIR=%s

# Copy and set up firewall script
COPY ${TEMPLATE_DIR}/docker-init-firewall.sh /usr/local/bin/init-firewall.sh
USER root
RUN chmod +x /usr/local/bin/init-firewall.sh && \
  echo "${USERNAME} ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/${USERNAME}-firewall && \
  chmod 0440 /etc/sudoers.d/${USERNAME}-firewall
USER ${USERNAME}

# Install OpenClaw
RUN OPENCLAW_VERSION="${OPENCLAW_VERSION}" OPENCLAW_NO_PROMPT=1 OPENCLAW_NO_ONBOARD=1 \
  curl -fsSL "https://openclaw.ai/install.sh" | \
  bash -s -- --version "${OPENCLAW_VERSION}" --no-prompt --no-onboard

# Copy and set up entrypoint script
COPY ${TEMPLATE_DIR}/docker-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["openclaw"]
`, baseImage, meta.FullVersion, packageBlock, dockerCLIBlock, localeBlock, userBlock, dockerGroupBlock, deltaBlock, hadolintBlock, fzfBlock, templatesDir)
}
