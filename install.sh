#!/bin/sh
# Installs the mcp-wizard scaffold binary from the latest GitHub release.
set -e
OWNER="sairaph"
REPO="mcp-wizard"
BIN="mcp-wizard"

OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS" in
  Linux*)  os=linux ;;
  Darwin*) os=darwin ;;
  *) printf '\n  Unsupported OS: %s\n' "$OS" >&2; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64)  arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) printf '\n  Unsupported architecture: %s\n' "$ARCH" >&2; exit 1 ;;
esac

ASSET="${BIN}-${os}-${arch}"
URL="https://github.com/${OWNER}/${REPO}/releases/latest/download/${ASSET}"
INSTALL_DIR="$HOME/.${REPO}/bin"
TARGET="$INSTALL_DIR/$BIN"
mkdir -p "$INSTALL_DIR"

printf '\n  %s installer\n\n  Downloading %s...\n' "$BIN" "$ASSET"

TEMP="${TARGET}.new"
trap 'rm -f "$TEMP"' EXIT HUP INT TERM

download_failed() {
  printf '\n  Download failed. Please check your connection and try again.\n' >&2
  printf '  URL: %s\n' "$URL" >&2
  [ -n "$1" ] && printf '  Reason: %s\n' "$1" >&2
  exit 1
}

fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O - "$1"
  else
    return 127
  fi
}

if ! fetch "$URL" > "$TEMP"; then
  if command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1; then
    download_failed
  fi
  download_failed "neither curl nor wget is available"
fi

if command -v sha256sum >/dev/null 2>&1; then
  SHA256_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA256_CMD="shasum -a 256"
else
  SHA256_CMD=""
fi

if [ -n "$SHA256_CMD" ]; then
  EXPECTED=$(fetch "${URL%/*}/SHA256SUMS.txt" 2>/dev/null | grep " $ASSET\$" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    ACTUAL=$($SHA256_CMD "$TEMP" | awk '{print $1}')
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      printf '\n  SHA256 mismatch.\n' >&2
      exit 1
    fi
  else
    printf '  Warning: could not fetch SHA256SUMS.txt; the download was not verified.\n' >&2
  fi
else
  printf '  Warning: no sha256 tool found; the download was not verified.\n' >&2
fi

if [ ! -s "$TEMP" ]; then
  printf '  Download did not complete; nothing was installed.\n' >&2
  exit 1
fi
chmod +x "$TEMP"
mv -f "$TEMP" "$TARGET"
trap - EXIT HUP INT TERM

case ":$PATH:" in
  *":$INSTALL_DIR:"*) on_path=1 ;;
  *) on_path=0 ;;
esac

if [ "$on_path" -eq 0 ]; then
  line="export PATH=\"$INSTALL_DIR:\$PATH\""
  for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile" "$HOME/.bash_profile"; do
    [ -f "$rc" ] || continue
    if ! grep -qF "$INSTALL_DIR" "$rc" 2>/dev/null; then
      printf '\n# added by %s installer\n%s\n' "$BIN" "$line" >> "$rc"
    fi
    on_path=2
  done
fi

printf '\n  Installed %s to %s\n' "$BIN" "$TARGET"
printf '  Generate a project with:\n    %s new --name <name> --owner <github-owner>\n' "$BIN"
if [ "$on_path" -eq 0 ]; then
  printf '\n  Add this to your shell profile:\n    export PATH="%s:$PATH"\n' "$INSTALL_DIR"
elif [ "$on_path" -eq 2 ]; then
  printf '\n  Open a new terminal so `%s` is on your PATH.\n' "$BIN"
fi
