#!/usr/bin/env bash
# ============================================================
# SSH Manager — Cross-compilation script for Linux/macOS
#
# Prerequisites:
#   - Go 1.20
#   - MinGW-w64 (Ubuntu: apt install gcc-mingw-w64-x86-64)
#   - UPX (optional, for compression)
#
# Usage:
#   ./build.sh              # Debug build
#   ./build.sh --release    # Release build (UPX compressed)
# ============================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

MODE="${1:-debug}"
APP_NAME="SSHManager.exe"
OUT_DIR="dist"

# Colors (if supported)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BOLD=''; NC=''
fi

info()  { echo -e "  ${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "  ${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "  ${RED}[ERROR]${NC} $1"; }

echo ""
echo "╔═══════════════════════════════════════╗"
echo "║   SSH Manager  Cross Build ($MODE)"
echo "╚═══════════════════════════════════════╝"
echo ""

# ---- Check prerequisites ----
command -v go >/dev/null 2>&1 || { error "Go not found."; exit 1; }
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || {
    error "MinGW-w64 not found."
    echo "  Install: sudo apt install gcc-mingw-w64-x86-64 (Ubuntu/Debian)"
    echo "  Or:      brew install mingw-w64 (macOS)"
    exit 1
}

echo -e "  ${BOLD}Go version:${NC}"
go version
echo ""

# ---- Step 1: GOPROXY ----
info "[1/4] Setting go proxy..."
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,direct
echo "  GOPROXY = $GOPROXY"

# ---- Step 2: Embed resources ----
info "[2/4] Embedding resources (icon + manifest)..."
if command -v x86_64-w64-mingw32-windres >/dev/null 2>&1; then
    (cd assets && x86_64-w64-mingw32-windres -o ../app.syso -i app.rc)
    info "app.syso generated"
elif command -v windres >/dev/null 2>&1; then
    (cd assets && windres -o ../app.syso -i app.rc)
    info "app.syso generated (system windres)"
else
    warn "windres not found; skipping resource embedding"
fi
echo ""

# ---- Step 3: Dependencies ----
info "[3/4] Downloading dependencies..."
go mod tidy
echo ""

# ---- Step 4: Build ----
info "[4/4] Compiling..."
mkdir -p "$OUT_DIR"

LDFLAGS="-s -w -H windowsgui -extldflags=-static"
if [ "$MODE" = "--release" ]; then
    LDFLAGS="$LDFLAGS -X main.version=1.0"
fi

export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc

go build -ldflags="$LDFLAGS" -o "$OUT_DIR/$APP_NAME" .
info "Output: $OUT_DIR/$APP_NAME"

# ---- Optional: UPX compression ----
if [ "$MODE" = "--release" ] && command -v upx >/dev/null 2>&1; then
    info "Compressing with UPX..."
    upx --best "$OUT_DIR/$APP_NAME" >/dev/null 2>&1
    info "UPX done"
elif [ "$MODE" = "--release" ]; then
    warn "UPX not installed; skipping compression"
fi

# ---- Show file size ----
if [ -f "$OUT_DIR/$APP_NAME" ]; then
    sz=$(stat -c%s "$OUT_DIR/$APP_NAME" 2>/dev/null || stat -f%z "$OUT_DIR/$APP_NAME" 2>/dev/null)
    echo ""
    echo "========================================"
    echo -e "  ${BOLD}DONE!${NC}"
    echo "  Output: $SCRIPT_DIR/$OUT_DIR/$APP_NAME"
    if [ "$sz" -ge 1048576 ]; then
        echo "  Size:   $((sz / 1048576)).$(((sz % 1048576) * 10 / 1048576)) MB"
    else
        echo "  Size:   $((sz / 1024)) KB"
    fi
    echo "========================================"
fi
