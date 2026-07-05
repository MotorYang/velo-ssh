#!/usr/bin/env sh
set -eu

APP_NAME="${APP_NAME:-vssh}"
PREFIX="${PREFIX:-/usr/local}"
BINDIR="${BINDIR:-$PREFIX/bin}"
VERSION_LDFLAGS="${VERSION_LDFLAGS:-}"

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
BIN_PATH="$BINDIR/$APP_NAME"

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to build VeloSSH" >&2
  exit 1
fi

mkdir -p "$BINDIR"
echo "Building $APP_NAME..."
go build -trimpath -ldflags "$VERSION_LDFLAGS" -o "$BIN_PATH" "$ROOT_DIR"
chmod 0755 "$BIN_PATH"
echo "Installed $APP_NAME to $BIN_PATH"
echo "Run: $APP_NAME"
