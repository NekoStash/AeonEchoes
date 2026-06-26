#!/usr/bin/env sh
set -eu

if [ "${AEON_ENV_LOADED:-0}" = "1" ]; then
  return 0 2>/dev/null || exit 0
fi

AEON_ENV_LOADED=1
export AEON_ENV_LOADED

if [ -z "${ROOT_DIR:-}" ]; then
  ROOT_DIR=$(pwd)
fi
export ROOT_DIR

DEFAULT_GO_BIN='C:/Users/25945/sdk/go1.26.4/bin/go'
GO_BIN=${GO_BIN:-$DEFAULT_GO_BIN}
export GO_BIN

if [ ! -x "$GO_BIN" ]; then
  if command -v go >/dev/null 2>&1; then
    GO_BIN=$(command -v go)
    export GO_BIN
  else
    printf '%s\n' "Go executable was not found. Set GO_BIN or install Go. Tried: $DEFAULT_GO_BIN" >&2
    exit 127
  fi
fi

NODE_BIN=${NODE_BIN:-node}
YARN_BIN=${YARN_BIN:-yarn}
export NODE_BIN YARN_BIN
