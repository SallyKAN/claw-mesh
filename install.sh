#!/bin/sh
# claw-mesh installer — downloads the latest release for your OS/arch.
# Usage: curl -fsSL https://raw.githubusercontent.com/SallyKAN/claw-mesh/main/install.sh | sh

set -e

REPO="SallyKAN/claw-mesh"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="claw-mesh"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)              echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
echo "Fetching latest release..."
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest release tag."
  exit 1
fi
VERSION="${TAG#v}"

ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

echo "Downloading claw-mesh ${TAG} for ${OS}/${ARCH}..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"
tar xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Install binary
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "claw-mesh ${TAG} installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  claw-mesh init        # Generate config"
echo "  claw-mesh up          # Start coordinator"
echo "  claw-mesh join <url>  # Join from another machine"
