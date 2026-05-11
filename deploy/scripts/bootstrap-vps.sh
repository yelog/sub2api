#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="${BASE_DIR:-/opt/sub2api}"
REPO_DIR="${REPO_DIR:-${BASE_DIR}/repo}"
DEPLOY_DIR="${DEPLOY_DIR:-${BASE_DIR}/deploy}"
SCRIPTS_DIR="${SCRIPTS_DIR:-${BASE_DIR}/scripts}"
REMOTE_URL="${REMOTE_URL:-https://github.com/yelog/sub2api.git}"
SOURCE_REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

mkdir -p "$BASE_DIR" "$SCRIPTS_DIR"

if [[ ! -d "$REPO_DIR/.git" ]]; then
  git clone "$REMOTE_URL" "$REPO_DIR"
else
  echo "repo already exists: $REPO_DIR"
fi

mkdir -p "$DEPLOY_DIR"
if [[ ! -f "$DEPLOY_DIR/docker-compose.local.yml" ]]; then
  install -m 644 "$SOURCE_REPO_DIR/deploy/docker-compose.local.yml" "$DEPLOY_DIR/docker-compose.local.yml"
fi
install -m 644 "$SOURCE_REPO_DIR/deploy/docker-compose.vps.yml" "$DEPLOY_DIR/docker-compose.vps.yml"
install -m 755 "$SOURCE_REPO_DIR/deploy/scripts/deploy.sh" "$SCRIPTS_DIR/deploy.sh"
install -m 755 "$SOURCE_REPO_DIR/deploy/scripts/rollback.sh" "$SCRIPTS_DIR/rollback.sh"

echo "bootstrap complete"
echo "repo: $REPO_DIR"
echo "deploy dir: $DEPLOY_DIR"
echo "scripts dir: $SCRIPTS_DIR"
