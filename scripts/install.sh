#!/usr/bin/env sh
set -eu

REPO="${REPO:-motoryang/velo-ssh}"
APP_NAME="${APP_NAME:-vssh}"
VERSION="${VERSION:-latest}"
PREFIX="${PREFIX:-/usr/local}"
BINDIR="${BINDIR:-$PREFIX/bin}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "Unsupported OS: $(uname -s). Use scripts/install.ps1 on Windows." >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo "amd64" ;;
    arm64 | aarch64) echo "arm64" ;;
    *)
      echo "Unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

download() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
    return
  fi
  echo "curl or wget is required" >&2
  exit 1
}

install_file() {
  src="$1"
  dst="$2"
  dir="$(dirname "$dst")"
  if [ -d "$dir" ] && [ -w "$dir" ]; then
    install -m 0755 "$src" "$dst"
    return
  fi
  if [ ! -d "$dir" ]; then
    if mkdir -p "$dir" 2>/dev/null; then
      install -m 0755 "$src" "$dst"
      return
    fi
  fi
  need sudo
  sudo mkdir -p "$dir"
  sudo install -m 0755 "$src" "$dst"
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
ASSET="velossh-$OS-$ARCH.tar.gz"
BIN_NAME="velossh-$OS-$ARCH"

if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/$ASSET"
else
  URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT INT TERM

echo "Downloading $APP_NAME $VERSION for $OS/$ARCH..."
download "$URL" "$TMPDIR/$ASSET"

tar -xzf "$TMPDIR/$ASSET" -C "$TMPDIR"
if [ ! -f "$TMPDIR/$BIN_NAME" ]; then
  echo "Release archive did not contain $BIN_NAME" >&2
  exit 1
fi

BIN_PATH="$BINDIR/$APP_NAME"
install_file "$TMPDIR/$BIN_NAME" "$BIN_PATH"

echo "Installed $APP_NAME to $BIN_PATH"
if ! command -v "$APP_NAME" >/dev/null 2>&1; then
  echo "Note: $BINDIR is not on PATH in this shell."
fi
echo "Run: $APP_NAME"
