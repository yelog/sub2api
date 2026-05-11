#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/data/workspace/sub2api}"
OUTPUT_DIR="${OUTPUT_DIR:-/data/tmp/sub2api-images}"
IMAGE_REPO="${IMAGE_REPO:-yelog/sub2api}"
PLATFORM="${PLATFORM:-linux/amd64}"
COMMIT="${COMMIT:-}"

if [[ -z "$COMMIT" ]]; then
  COMMIT="$(git -C "$REPO_DIR" rev-parse --short HEAD)"
fi

IMAGE_TAG="${IMAGE_TAG:-${IMAGE_REPO}:${COMMIT}}"
TAR_PATH="${TAR_PATH:-${OUTPUT_DIR}/sub2api-${COMMIT}.tar.gz}"

mkdir -p "$OUTPUT_DIR"

echo "repo: $REPO_DIR"
echo "image: $IMAGE_TAG"
echo "platform: $PLATFORM"
echo "tar: $TAR_PATH"

cd "$REPO_DIR"
docker build --platform "$PLATFORM" -t "$IMAGE_TAG" .
docker save "$IMAGE_TAG" | gzip -c > "$TAR_PATH"
sha256sum "$TAR_PATH" > "${TAR_PATH}.sha256"

echo "built image tar: $TAR_PATH"
echo "checksum: ${TAR_PATH}.sha256"
