#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/data/workspace/sub2api}"
VPS="${VPS:-root@47.236.19.206}"
SSH_CMD="${SSH_CMD:-/data/workspace/openclaw/main/direct_ssh.sh}"
REMOTE_IMAGES_DIR="${REMOTE_IMAGES_DIR:-/opt/sub2api/images}"
REMOTE_SCRIPTS_DIR="${REMOTE_SCRIPTS_DIR:-/opt/sub2api/scripts}"
IMAGE_REPO="${IMAGE_REPO:-yelog/sub2api}"
COMMIT="${COMMIT:-$(git -C "$REPO_DIR" rev-parse --short HEAD)}"
IMAGE_TAG="${IMAGE_TAG:-${IMAGE_REPO}:${COMMIT}}"
OUTPUT_DIR="${OUTPUT_DIR:-/data/tmp/sub2api-images}"
TAR_PATH="${TAR_PATH:-${OUTPUT_DIR}/sub2api-${COMMIT}.tar.gz}"
TAR_NAME="$(basename "$TAR_PATH")"
SCP_BIN="${SCP_BIN:-scp}"
SSH_KEY="${SSH_KEY:-/home/yelog/.ssh/openclaw_vps}"

cd "$REPO_DIR"
COMMIT="$COMMIT" IMAGE_TAG="$IMAGE_TAG" OUTPUT_DIR="$OUTPUT_DIR" bash deploy/scripts/build-image-tar.sh

$SSH_CMD "$VPS" "mkdir -p '$REMOTE_IMAGES_DIR' '$REMOTE_SCRIPTS_DIR'"
$SCP_BIN -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ProxyCommand=none \
  "$TAR_PATH" "${TAR_PATH}.sha256" "$VPS:$REMOTE_IMAGES_DIR/"
$SCP_BIN -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ProxyCommand=none \
  "$REPO_DIR/deploy/docker-compose.vps.yml" "$REPO_DIR/deploy/scripts/deploy-image-tar.sh" "$VPS:/tmp/"
$SSH_CMD "$VPS" "set -e; install -m 644 /tmp/docker-compose.vps.yml /opt/sub2api/deploy/docker-compose.vps.yml; install -m 755 /tmp/deploy-image-tar.sh '$REMOTE_SCRIPTS_DIR/deploy-image-tar.sh'; cd '$REMOTE_IMAGES_DIR'; sha256sum -c '${TAR_NAME}.sha256'; '$REMOTE_SCRIPTS_DIR/deploy-image-tar.sh' '$REMOTE_IMAGES_DIR/$TAR_NAME' '$IMAGE_TAG'"

echo "published: $IMAGE_TAG"
