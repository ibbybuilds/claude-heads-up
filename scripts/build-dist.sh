#!/usr/bin/env sh
# Builds dist/heads-up-<os>-<arch>[.exe] for every supported target, or only
# the current machine with --local. The release workflow and local plugin
# testing both use this, so they ship the same layout.
set -eu

cd "$(dirname "$0")/.."
version="${VERSION:-dev}"
ldflags="-s -w -X main.version=$version"

targets="windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"
if [ "${1:-}" = "--local" ]; then
  targets="$(go env GOOS)/$(go env GOARCH)"
fi

mkdir -p dist
for target in $targets; do
  os="${target%/*}"
  arch="${target#*/}"
  out="dist/heads-up-$os-$arch"
  [ "$os" = windows ] && out="$out.exe"
  echo "building $out"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$ldflags" -o "$out" ./cmd/heads-up
done
