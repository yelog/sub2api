#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-/opt/sub2api/repo}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/sub2api/deploy}"
SCRIPTS_DIR="${SCRIPTS_DIR:-/opt/sub2api/scripts}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:18080/health}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
SLEEP_SECONDS="${SLEEP_SECONDS:-5}"
STATE_FILE="${STATE_FILE:-${SCRIPTS_DIR}/current-successful-release}"
COMPOSE_ARGS=(-f docker-compose.local.yml -f docker-compose.vps.yml)
TARGET_COMMIT="${1:-}"

if [[ -z "$TARGET_COMMIT" ]]; then
  echo "usage: $0 <commit>" >&2
  exit 1
fi

require_file() {
  local path="$1"
  [[ -e "$path" ]] || { echo "missing required path: $path" >&2; exit 1; }
}

require_file "$REPO_DIR/.git"
require_file "$DEPLOY_DIR/docker-compose.local.yml"
require_file "$DEPLOY_DIR/docker-compose.vps.yml"
mkdir -p "$SCRIPTS_DIR"

cd "$REPO_DIR"
git fetch origin
git reset --hard "$TARGET_COMMIT"
ROLLED_COMMIT="$(git rev-parse HEAD)"
echo "rollback target: $ROLLED_COMMIT"

cd "$DEPLOY_DIR"
docker compose "${COMPOSE_ARGS[@]}" build sub2api
docker compose "${COMPOSE_ARGS[@]}" up -d sub2api

DEADLINE=$(( $(date +%s) + TIMEOUT_SECONDS ))
until curl -fsS "$HEALTH_URL" >/dev/null 2>&1; do
  if (( $(date +%s) >= DEADLINE )); then
    echo "health check failed after rollback: $HEALTH_URL" >&2
    exit 1
  fi
  sleep "$SLEEP_SECONDS"
done

{
  echo "commit=$ROLLED_COMMIT"
  echo "rolled_back_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "health_url=$HEALTH_URL"
} > "$STATE_FILE"

echo "rollback succeeded: $ROLLED_COMMIT"
