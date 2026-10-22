#!/bin/bash
# 构建 MC Launcher（CLI + 服务）到 dist/
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="$ROOT/dist"
mkdir -p "$DIST"

build_cli() {
  local goos=$1 goarch=$2 out=$3
  (cd "$ROOT/Core/mc-launcher" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
    go build -trimpath -ldflags="-s -w" -o "$DIST/$out" ./cmd/mc-launcher)
}

build_srv() {
  local goos=$1 goarch=$2 out=$3
  (cd "$ROOT/backend" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
    go build -trimpath -ldflags="-s -w" -o "$DIST/$out" .)
}

case "${1:-all}" in
  linux-amd64)
    build_cli linux amd64 mc-launcher-linux-amd64
    build_srv linux amd64 mc-launcher-server-linux-amd64 ;;
  linux-arm64)
    build_cli linux arm64 mc-launcher-linux-arm64
    build_srv linux arm64 mc-launcher-server-linux-arm64 ;;
  windows-amd64)
    build_cli windows amd64 mc-launcher-windows-amd64.exe
    build_srv windows amd64 mc-launcher-server-windows-amd64.exe ;;
  all)
    build_cli linux amd64 mc-launcher-linux-amd64
    build_cli linux arm64 mc-launcher-linux-arm64
    build_cli windows amd64 mc-launcher-windows-amd64.exe
    build_srv linux amd64 mc-launcher-server-linux-amd64
    build_srv linux arm64 mc-launcher-server-linux-arm64
    build_srv windows amd64 mc-launcher-server-windows-amd64.exe ;;
  *)
    echo "用法: $0 [linux-amd64|linux-arm64|windows-amd64|all]"; exit 1 ;;
esac

ls -lh "$DIST"
