#!/usr/bin/env bash
set -euo pipefail

GO_VERSION="${GO_VERSION:-1.26.3}"
GO_DOWNLOAD_BASE="${GO_DOWNLOAD_BASE:-https://golang.google.cn/dl}"
GO_PROXY="${GO_PROXY:-https://goproxy.cn,direct}"
GO_SUMDB="${GO_SUMDB:-sum.golang.google.cn}"
INSTALL_ROOT="${INSTALL_ROOT:-/usr/local}"
PROXY_URL="${PROXY_URL:-}"
BASHRC_PATH="${BASHRC_PATH:-$HOME/.bashrc}"

if [[ -n "$PROXY_URL" ]]; then
  export http_proxy="$PROXY_URL"
  export https_proxy="$PROXY_URL"
  export all_proxy="$PROXY_URL"
fi


ARCHIVE="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
DOWNLOAD_URL="${GO_DOWNLOAD_BASE}/${ARCHIVE}"
TMP_ARCHIVE="/tmp/${ARCHIVE}"

ensure_bashrc_settings() {
  local managed_block
  managed_block=$(cat <<EOF
# >>> go env >>>
export PATH=/usr/local/go/bin:\$PATH
export GOPROXY=${GO_PROXY}
export GOSUMDB=${GO_SUMDB}
# <<< go env <<<
EOF
)

  if [[ -f "$BASHRC_PATH" ]] && grep -q '# >>> go env >>>' "$BASHRC_PATH"; then
    python3 - "$BASHRC_PATH" "$managed_block" <<'PY'
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
block = sys.argv[2]
text = path.read_text()
start = text.index("# >>> go env >>>")
end = text.index("# <<< go env <<<", start) + len("# <<< go env <<<")
replacement = block.rstrip("\n")
updated = text[:start] + replacement + text[end:]
if not updated.endswith("\n"):
    updated += "\n"
path.write_text(updated)
PY
    return
  fi

  if [[ ! -f "$BASHRC_PATH" ]]; then
    touch "$BASHRC_PATH"
  fi

  printf '\n%s\n' "$managed_block" >> "$BASHRC_PATH"
}

echo "downloading ${DOWNLOAD_URL}"
curl -fL --retry 3 --connect-timeout 15 -o "$TMP_ARCHIVE" "$DOWNLOAD_URL"
printf '%s  %s\n' "$GO_SHA256" "$TMP_ARCHIVE" | sha256sum -c -

rm -rf "${INSTALL_ROOT}/go"
tar -C "$INSTALL_ROOT" -xzf "$TMP_ARCHIVE"
ln -sf "${INSTALL_ROOT}/go/bin/go" /usr/local/bin/go
ln -sf "${INSTALL_ROOT}/go/bin/gofmt" /usr/local/bin/gofmt

/usr/local/bin/go env -w GOPROXY="$GO_PROXY"
/usr/local/bin/go env -w GOSUMDB="$GO_SUMDB"
ensure_bashrc_settings

echo "Go installed: $(/usr/local/bin/go version)"
echo "GOPROXY set to: $(/usr/local/bin/go env GOPROXY)"
echo "GOSUMDB set to: $(/usr/local/bin/go env GOSUMDB)"
echo "bashrc updated: ${BASHRC_PATH}"