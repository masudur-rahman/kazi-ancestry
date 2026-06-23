#!/usr/bin/env bash
# Cross-build the image on Mac (arm64) for the Debian VM, and export a loadable tar.
#
# Usage:
#   tools/build.sh                 # builds linux/amd64 -> kazi-ancestry-amd64.tar
#   PLATFORM=linux/arm64 tools/build.sh
#
# Then copy to the VM and load:
#   scp kazi-ancestry-*.tar user@vm:~
#   ssh user@vm 'docker load -i kazi-ancestry-*.tar \
#       && docker run -d --restart unless-stopped -p 80:80 --name kazi-ancestry kazi-ancestry:latest'
set -euo pipefail

PLATFORM="${PLATFORM:-linux/amd64}"
TAG="${TAG:-kazi-ancestry:latest}"
ARCH="${PLATFORM##*/}"
OUT="kazi-ancestry-${ARCH}.tar"

cd "$(dirname "$0")/.."

# buildx with QEMU emulation (bundled with Docker Desktop) cross-builds for the VM's arch.
docker buildx build \
  --platform "$PLATFORM" \
  --tag "$TAG" \
  --output "type=docker,dest=${OUT}" \
  .

echo "Built $TAG for $PLATFORM -> $OUT"
echo "Copy to the VM, then: docker load -i $OUT"
