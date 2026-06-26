#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
export ROOT_DIR
. "$ROOT_DIR/infra/scripts/env.sh"

SERVER_DIR="$ROOT_DIR/apps/server"
ACTION="${1:-run}"
shift || true

if [ ! -f "$SERVER_DIR/go.mod" ]; then
  printf '%s\n' "Server module was not found at $SERVER_DIR" >&2
  exit 1
fi

cd "$SERVER_DIR"

case "$ACTION" in
  run)
    if [ -d "$SERVER_DIR/cmd" ]; then
      exec "$GO_BIN" run ./cmd/... "$@"
    fi
    exec "$GO_BIN" run . "$@"
    ;;
  test)
    exec "$GO_BIN" test ./... "$@"
    ;;
  build)
    mkdir -p "$ROOT_DIR/dist"
    if [ -d "$SERVER_DIR/cmd" ]; then
      exec "$GO_BIN" build -o "$ROOT_DIR/dist/server" ./cmd/...
    fi
    exec "$GO_BIN" build -o "$ROOT_DIR/dist/server" .
    ;;
  mod-tidy)
    exec "$GO_BIN" mod tidy "$@"
    ;;
  *)
    printf '%s\n' "Unsupported server action: $ACTION" >&2
    printf '%s\n' "Usage: scripts/go-server.sh [run|test|build|mod-tidy]" >&2
    exit 2
    ;;
esac
