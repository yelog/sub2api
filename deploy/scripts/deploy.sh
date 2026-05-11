#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/opt/sub2api/repo}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/sub2api/deploy}"
SCRIPTS_DIR="${SCRIPTS_DIR:-/opt/sub2api/scripts}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:18080/health}"
BRANCH="${BRANCH:-origin/main}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
SLEEP_SECONDS="${SLEEP_SECONDS:-5}"
STATE_FILE="${STATE_FILE:-${SCRIPTS_DIR}/current-successful-release}"
PREV_FILE="${PREV_FILE:-${SCRIPTS_DIR}/previous-release}"
COMPOSE_ARGS=(-f docker-compose.local.yml -f docker-compose.vps.yml)
DRY_RUN="false"

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN="true"
fi

run() {
  if [[ "$DRY_RUN" == "true" ]]; then
    printf '[dry-run] %s\n' "$*"
  else
    "$@"
  fi
}

require_file() {
  local path="$1"
  [[ -e "$path" ]] || { echo "missing required path: $path" >&2; exit 1; }
}

require_file "$REPO_DIR/.git"
require_file "$DEPLOY_DIR/docker-compose.local.yml"
require_file "$DEPLOY_DIR/docker-compose.vps.yml"
mkdir -p "$SCRIPTS_DIR"

cd "$REPO_DIR"
CURRENT_COMMIT="$(git rev-parse HEAD)"
printf '%s\n' "$CURRENT_COMMIT" > "$PREV_FILE"
echo "current commit: $CURRENT_COMMIT"

run git fetch origin
run git reset --hard "$BRANCH"
TARGET_COMMIT="$(git rev-parse HEAD)"
echo "target commit: $TARGET_COMMIT"

cd "$DEPLOY_DIR"
run docker compose "${COMPOSE_ARGS[@]}" build sub2api
run docker compose "${COMPOSE_ARGS[@]}" up -d sub2api

if [[ "$DRY_RUN" == "true" ]]; then
  exit 0
fi

DEADLINE=$(( $(date +%s) + TIMEOUT_SECONDS ))
until curl -fsS "$HEALTH_URL" >/dev/null 2>&1; do
  if (( $(date +%s) >= DEADLINE )); then
    echo "health check failed: $HEALTH_URL" >&2
    exit 1
  fi
  sleep "$SLEEP_SECONDS"
done

{
  echo "commit=$TARGET_COMMIT"
  echo "deployed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "health_url=$HEALTH_URL"
} > "$STATE_FILE"

echo "deployment succeeded: $TARGET_COMMIT"
