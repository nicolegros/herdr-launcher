#!/bin/sh
# Build herdr-launcher for `herdr plugin install`. herdr runs this as the
# manifest's [[build]] step after cloning the repo.
set -eu

mkdir -p bin

if command -v go >/dev/null 2>&1; then
	echo "herdr-launcher: building from source (go build)…" >&2
	exec go build -o bin/herdr-launcher .
fi

echo "herdr-launcher: Go toolchain required to build this plugin." >&2
exit 1
