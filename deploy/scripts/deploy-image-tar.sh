#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/sub2api/deploy}"
SCRIPTS_DIR="${SCRIPTS_DIR:-/opt/sub2api/scripts}"
IMAGES_DIR="${IMAGES_DIR:-/opt/sub2api/images}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:18080/health}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
SLEEP_SECONDS="${SLEEP_SECONDS:-5}"
STATE_FILE="${STATE_FILE:-${SCRIPTS_DIR}/current-successful-release}"
COMPOSE_ARGS=(-f docker-compose.local.yml -f docker-compose.vps.yml)
TAR_PATH="${1:-}"
IMAGE_TAG="${2:-}"

if [[ -z "$TAR_PATH" || -z "$IMAGE_TAG" ]]; then
  echo "usage: $0 <image-tar.gz> <image-tag>" >&2
  exit 1
fi

if [[ "$TAR_PATH" != /* ]]; then
  TAR_PATH="${IMAGES_DIR}/${TAR_PATH}"
fi

[[ -f "$TAR_PATH" ]] || { echo "missing image tar: $TAR_PATH" >&2; exit 1; }
[[ -f "$DEPLOY_DIR/docker-compose.local.yml" ]] || { echo "missing compose: $DEPLOY_DIR/docker-compose.local.yml" >&2; exit 1; }
[[ -f "$DEPLOY_DIR/docker-compose.vps.yml" ]] || { echo "missing compose: $DEPLOY_DIR/docker-compose.vps.yml" >&2; exit 1; }
mkdir -p "$SCRIPTS_DIR"

echo "loading image: $TAR_PATH"
gzip -dc "$TAR_PATH" | docker load

cd "$DEPLOY_DIR"
echo "starting image: $IMAGE_TAG"
SUB2API_IMAGE="$IMAGE_TAG" docker compose "${COMPOSE_ARGS[@]}" up -d sub2api

DEADLINE=$(( $(date +%s) + TIMEOUT_SECONDS ))
until curl -fsS "$HEALTH_URL" >/dev/null 2>&1; do
  if (( $(date +%s) >= DEADLINE )); then
    echo "health check failed: $HEALTH_URL" >&2
    exit 1
  fi
  sleep "$SLEEP_SECONDS"
done

{
  echo "image=$IMAGE_TAG"
  echo "tar=$TAR_PATH"
  echo "deployed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "health_url=$HEALTH_URL"
} > "$STATE_FILE"

echo "deployment succeeded: $IMAGE_TAG"
