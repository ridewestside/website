#!/usr/bin/env bash
# Install the Typst CLI for the current build, used by cmd/buildpdf to render
# events.pdf. Pins to $TYPST_VERSION (default 0.14.2) and places the binary in
# a directory that's on $PATH for the rest of the build.
set -euo pipefail

VERSION="${TYPST_VERSION:-0.14.2}"

if command -v typst >/dev/null 2>&1; then
  existing="$(typst --version | awk '{print $2}')"
  if [ "$existing" = "$VERSION" ]; then
    echo "typst $VERSION already installed"
    exit 0
  fi
fi

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  asset="typst-x86_64-unknown-linux-musl" ;;
  Linux-aarch64) asset="typst-aarch64-unknown-linux-musl" ;;
  Darwin-x86_64) asset="typst-x86_64-apple-darwin" ;;
  Darwin-arm64)  asset="typst-aarch64-apple-darwin" ;;
  *) echo "unsupported platform: $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac

# Prefer a writable system location; fall back to a per-user dir.
if [ -w /usr/local/bin ]; then
  bindir="/usr/local/bin"
else
  bindir="${HOME}/.local/bin"
  mkdir -p "$bindir"
  case ":$PATH:" in
    *":$bindir:"*) ;;
    *) export PATH="$bindir:$PATH" ;;
  esac
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

url="https://github.com/typst/typst/releases/download/v${VERSION}/${asset}.tar.xz"
echo "Downloading $url"
curl -fsSL "$url" -o "$tmp/typst.tar.xz"
tar -xJf "$tmp/typst.tar.xz" -C "$tmp"
install -m 0755 "$tmp/$asset/typst" "$bindir/typst"
echo "Installed typst $VERSION to $bindir/typst"
"$bindir/typst" --version
