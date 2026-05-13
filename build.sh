#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="${APP_NAME:-doc-svr}"
PORT="${PORT:-4000}"
GOFLAGS_VALUE="${GOFLAGS:--mod=vendor}"
VERSION="${VERSION:-$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X main.version=${VERSION}"

log() {
	printf '[doc-svr][%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

usage() {
	cat <<'EOF'
Usage:
  ./build.sh generate-home  Render homepage into public/index.html
  ./build.sh build        Build binary into ./bin/
  ./build.sh run          Run local server
  ./build.sh check        Validate compilation
  ./build.sh clean        Remove generated artifacts
  ./build.sh tidy         Sync go.mod/go.sum
  ./build.sh vendor       Refresh vendor directory
  ./build.sh docker       Build Docker image
EOF
}

run_generate_home() {
	log "rendering homepage into public/index.html"
	pushd "$ROOT_DIR" >/dev/null
	GOFLAGS="$GOFLAGS_VALUE" go run -ldflags "$LDFLAGS" . generate-home --output public/index.html
	popd >/dev/null
	log "homepage generated at $ROOT_DIR/public/index.html"
}

run_build() {
	log "building ${APP_NAME} version=${VERSION}"
	run_generate_home
	mkdir -p "$ROOT_DIR/bin"
	pushd "$ROOT_DIR" >/dev/null
	CGO_ENABLED=0 GOFLAGS="$GOFLAGS_VALUE" go build -trimpath -ldflags "$LDFLAGS" -o "bin/${APP_NAME}" .
	popd >/dev/null
	log "binary ready at $ROOT_DIR/bin/${APP_NAME}"
}

run_server() {
	log "starting local server on :${PORT}"
	run_generate_home
	pushd "$ROOT_DIR" >/dev/null
	PORT="$PORT" GOFLAGS="$GOFLAGS_VALUE" go run -ldflags "$LDFLAGS" .
	popd >/dev/null
}

run_check() {
	log "running go build validation"
	pushd "$ROOT_DIR" >/dev/null
	GOFLAGS="$GOFLAGS_VALUE" go build ./...
	popd >/dev/null
	log "validation passed"
}

run_tidy() {
	log "syncing module files"
	pushd "$ROOT_DIR" >/dev/null
	go mod tidy
	popd >/dev/null
	log "go.mod and go.sum updated"
}

run_vendor() {
	log "refreshing vendor directory"
	pushd "$ROOT_DIR" >/dev/null
	go mod vendor
	popd >/dev/null
	log "vendor synced"
}

run_clean() {
	log "removing generated artifacts"
	pushd "$ROOT_DIR" >/dev/null
	rm -rf bin doc_svr
	find . -name '.DS_Store' -delete
	popd >/dev/null
	log "generated artifacts removed"
}

run_docker() {
	log "building docker image ${APP_NAME}:${VERSION}"
	pushd "$ROOT_DIR" >/dev/null
	docker build --build-arg VERSION="$VERSION" -t "${APP_NAME}:${VERSION}" -f dockerfile .
	popd >/dev/null
	log "docker image ready"
}

COMMAND="${1:-build}"

case "$COMMAND" in
	generate-home)
		run_generate_home
		;;
	build)
		run_build
		;;
	run)
		run_server
		;;
	check)
		run_check
		;;
	clean)
		run_clean
		;;
	tidy)
		run_tidy
		;;
	vendor)
		run_vendor
		;;
	docker)
		run_docker
		;;
	help|-h|--help)
		usage
		;;
	*)
		usage
		exit 1
		;;
esac
