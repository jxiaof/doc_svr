#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="${APP_NAME:-doc-svr}"
PORT="${PORT:-3000}"
LOG_FILE="${LOG_FILE:-$ROOT_DIR/bin/${APP_NAME}.log}"

mkdir -p "$ROOT_DIR/bin"

pushd "$ROOT_DIR" >/dev/null
CGO_ENABLED=0 GOFLAGS="${GOFLAGS:--mod=vendor}" go build -trimpath -o "bin/${APP_NAME}" .
nohup env PORT="$PORT" "$ROOT_DIR/bin/${APP_NAME}" >"$LOG_FILE" 2>&1 &
popd >/dev/null

printf 'started %s on :%s, log=%s\n' "$APP_NAME" "$PORT" "$LOG_FILE"