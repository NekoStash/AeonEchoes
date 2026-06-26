#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
export ROOT_DIR
. "$ROOT_DIR/infra/scripts/env.sh"

exec "$NODE_BIN" "$ROOT_DIR/infra/fake-provider/server.js" "$@"
