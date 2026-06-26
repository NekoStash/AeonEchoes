#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
export ROOT_DIR
. "$ROOT_DIR/infra/scripts/env.sh"

WEB_DIR="$ROOT_DIR/apps/web"

if [ ! -f "$WEB_DIR/package.json" ]; then
  printf '%s\n' "Web package was not found at $WEB_DIR" >&2
  exit 1
fi

if [ ! -d "$ROOT_DIR/node_modules" ]; then
  printf '%s\n' "Installing workspace dependencies in $ROOT_DIR"
  "$YARN_BIN" --cwd "$ROOT_DIR" install
fi

exec "$YARN_BIN" --cwd "$ROOT_DIR" workspace @aeon-echoes/web dev --host "${WEB_HOST:-127.0.0.1}" --port "${WEB_PORT:-3000}"
