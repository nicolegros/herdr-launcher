#!/bin/sh
# Build herdr-launcher for `herdr plugin install`. herdr runs this as the
# manifest's [[build]] step after cloning the repo.
#
# Prefers a local Go toolchain (builds from source). Falls back to downloading
# the latest prebuilt binary from GitHub Releases.
set -eu

REPO="nicolegros/herdr-launcher"
BINARY="herdr-launcher"

mkdir -p bin

if command -v go >/dev/null 2>&1; then
	echo "herdr-launcher: building from source (go build)…" >&2
	exec go build -o bin/herdr-launcher .
fi

echo "herdr-launcher: no Go toolchain — downloading prebuilt binary…" >&2

# Detect platform
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
	linux) OS="linux" ;;
	darwin) OS="darwin" ;;
	*) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Resolve latest release tag
if command -v curl >/dev/null 2>&1; then
	TAG=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" | grep -o '[^/]*$')
elif command -v wget >/dev/null 2>&1; then
	TAG=$(wget --max-redirect=0 --server-response -O /dev/null "https://github.com/${REPO}/releases/latest" 2>&1 | grep -i 'Location:' | grep -o '[^/]*$' | tr -d '\r\n')
else
	echo "Need curl or wget to download release" >&2
	exit 1
fi

if [ -z "$TAG" ]; then
	echo "Could not determine latest release" >&2
	exit 1
fi

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

echo "  Downloading ${URL}" >&2

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

if command -v curl >/dev/null 2>&1; then
	curl -fsSL -o "$TMP/$ARCHIVE" "$URL"
else
	wget -q -O "$TMP/$ARCHIVE" "$URL"
fi

tar -xzf "$TMP/$ARCHIVE" -C bin/
chmod +x "bin/$BINARY"
echo "herdr-launcher: installed prebuilt binary." >&2
