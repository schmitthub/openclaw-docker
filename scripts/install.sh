#!/usr/bin/env bash
set -euo pipefail

REPO="schmitthub/openclaw-docker"
VERSION="latest"
INSTALL_MODE="local"
INSTALL_DIR=""
BINARY_NAME="openclaw-docker"

usage() {
  cat <<'EOF'
Install or update openclaw-docker from GitHub releases.

Usage:
  install.sh [options]

Options:
  --local               Install to ~/.local/bin (default)
  --global              Install to /usr/local/bin
  --install-dir <dir>   Install to a custom directory
  --version <version>   Version tag (e.g. v0.2.0) or latest (default)
  --repo <owner/repo>   GitHub repo (default: schmitthub/openclaw-docker)
  -h, --help            Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --local)
      INSTALL_MODE="local"
      INSTALL_DIR=""
      ;;
    --global)
      INSTALL_MODE="global"
      INSTALL_DIR=""
      ;;
    --install-dir)
      [[ $# -ge 2 ]] || { echo "--install-dir requires a value" >&2; exit 1; }
      INSTALL_MODE="custom"
      INSTALL_DIR="$2"
      shift
      ;;
    --version)
      [[ $# -ge 2 ]] || { echo "--version requires a value" >&2; exit 1; }
      VERSION="$2"
      shift
      ;;
    --repo)
      [[ $# -ge 2 ]] || { echo "--repo requires a value" >&2; exit 1; }
      REPO="$2"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
  shift
done

if [[ "$INSTALL_MODE" == "local" ]]; then
  INSTALL_DIR="${HOME}/.local/bin"
elif [[ "$INSTALL_MODE" == "global" ]]; then
  INSTALL_DIR="/usr/local/bin"
fi

if [[ -z "$INSTALL_DIR" ]]; then
  echo "Failed to determine install directory" >&2
  exit 1
fi

case "$(uname -s)" in
  Linux) OS="linux" ;;
  *) echo "This installer currently supports Linux only" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -E '"tag_name"' | head -n1 | sed -E 's/.*"([^"]+)".*/\1/')"
fi

if [[ -z "$VERSION" ]]; then
  echo "Failed to resolve version" >&2
  exit 1
fi

ARTIFACT="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARTIFACT}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_PATH="${TMP_DIR}/${ARTIFACT}"

curl -fsSL "$URL" -o "$ARCHIVE_PATH"

tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

BIN_PATH="${TMP_DIR}/${BINARY_NAME}"
if [[ ! -f "$BIN_PATH" ]]; then
  echo "Downloaded archive does not contain ${BINARY_NAME}" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"

if [[ "$INSTALL_MODE" == "global" ]]; then
  sudo install -m 0755 "$BIN_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
else
  install -m 0755 "$BIN_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "Installed ${BINARY_NAME} ${VERSION} to ${INSTALL_DIR}/${BINARY_NAME}"
if [[ "$INSTALL_MODE" == "local" ]]; then
  echo "Make sure ${INSTALL_DIR} is in your PATH"
fi
