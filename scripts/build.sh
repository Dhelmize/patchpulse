#!/usr/bin/env bash
set -e

mkdir -p dist
export CGO_ENABLED=0

TARGETS=(
  "windows/arm64/.exe"
  "windows/amd64/.exe"
  "linux/arm64/"
  "linux/amd64/"
  "darwin/arm64/"
)

for target in "${TARGETS[@]}"; do
  IFS='/' read -r os arch ext <<< "$target"
  out="dist/patchpulse-${os}-${arch}${ext}"
  echo "Building ${out} (${os}/${arch})..."
  GOOS="${os}" GOARCH="${arch}" go build -ldflags "-s -w" -o "${out}" ./cmd/patchpulse
done

echo "Cross-compilation completed successfully in ./dist/"

