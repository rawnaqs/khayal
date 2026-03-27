#!/bin/sh
set -e

REPO="rawnaqs/khayal"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux|darwin) ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Get latest version if needed
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
  fi
fi

echo "Installing khayal v${VERSION} (${OS}/${ARCH})..."

# Download khayal binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/khayal_${VERSION}_${OS}_${ARCH}.tar.gz"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sfL "$DOWNLOAD_URL" | tar -xz -C "$TMPDIR"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/khayal" "$INSTALL_DIR/khayal"
else
  sudo mv "$TMPDIR/khayal" "$INSTALL_DIR/khayal"
fi
chmod +x "$INSTALL_DIR/khayal"

# Download kl client
KL_URL="https://github.com/$REPO/releases/download/v${VERSION}/khayal-client_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -sfL "$KL_URL" | tar -xz -C "$TMPDIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/kl" "$INSTALL_DIR/kl"
else
  sudo mv "$TMPDIR/kl" "$INSTALL_DIR/kl"
fi
chmod +x "$INSTALL_DIR/kl"

echo "Installed:"
echo "  khayal  $("$INSTALL_DIR/khayal" version 2>/dev/null || echo "v$VERSION")"
echo "  kl      $("$INSTALL_DIR/kl" --help >/dev/null 2>&1 && echo "v$VERSION" || echo "v$VERSION")"
echo ""
echo "Get started:"
echo "  khayal init"
echo "  khayal start"
echo "  kl init"
echo "  kl \"my first thought\""
