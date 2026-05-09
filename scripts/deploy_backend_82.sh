#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SYNC_SCRIPT="${SYNC_SCRIPT:-${SCRIPT_DIR}/sync_backend_82.sh}"

REMOTE_HOST="${REMOTE_HOST:-82.156.54.232}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/gaokao-ai-backend/backend}"
TEMP_PORT="${TEMP_PORT:-18083}"
PROD_PORT="${PROD_PORT:-8080}"
PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-http://82.156.54.232:80}"
SSH_PASSWORD="${SSHPASS:-}"
SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o ConnectionAttempts=3
  -o ConnectTimeout=10
)

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing command: $1" >&2
    exit 1
  fi
}

if [[ -z "${SSH_PASSWORD}" ]]; then
  echo "SSHPASS is required" >&2
  exit 1
fi

if [[ ! -x "${SYNC_SCRIPT}" ]]; then
  echo "sync script not found or not executable: ${SYNC_SCRIPT}" >&2
  exit 1
fi

require_command sshpass
require_command curl

echo "[1/4] syncing source and building remote candidate"
"${SYNC_SCRIPT}"

echo "[2/4] validating candidate on temporary port ${TEMP_PORT}"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "bash -s" <<EOF
set -euo pipefail
cd "${REMOTE_DIR}"
pkill -f "SERVER_ADDR=:${TEMP_PORT} ./gaokao-api.new" || true
nohup env SERVER_ADDR=":${TEMP_PORT}" ./gaokao-api.new >/tmp/gaokao-api-${TEMP_PORT}.log 2>&1 &
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:${TEMP_PORT}/healthz" >/dev/null; then
    exit 0
  fi
  sleep 1
done
echo "temporary validation failed" >&2
tail -n 50 /tmp/gaokao-api-${TEMP_PORT}.log >&2 || true
exit 1
EOF

echo "[3/4] promoting candidate to production port ${PROD_PORT}"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "bash -s" <<EOF
set -euo pipefail
cd "${REMOTE_DIR}"
timestamp="4(date +%Y%m%d_%H%M%S)"
if [[ -f gaokao-api ]]; then
  cp gaokao-api "gaokao-api.bak.4timestamp"
fi
mv gaokao-api.new gaokao-api
current_pid="4(ss -ltnp 2>/dev/null | awk '/:${PROD_PORT} / { if (match(40, /pid=([0-9]+)/, m)) { print m[1]; exit } }')"
if [[ -n "4current_pid" ]]; then
  kill "4current_pid" || true
  sleep 1
fi
nohup env SERVER_ADDR=":${PROD_PORT}" ./gaokao-api >/tmp/gaokao-api-${PROD_PORT}.log 2>&1 &
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:${PROD_PORT}/healthz" >/dev/null; then
    exit 0
  fi
  sleep 1
done
echo "production health check failed" >&2
tail -n 80 /tmp/gaokao-api-${PROD_PORT}.log >&2 || true
exit 1
EOF

echo "[4/4] verifying public endpoint"
curl -fsS "${PUBLIC_BASE_URL}/healthz" >/dev/null
curl -fsS "${PUBLIC_BASE_URL}/api/colleges?province=%E9%BB%91%E9%BE%99%E6%B1%9F&subject=%E5%8E%86%E5%8F%B2&year=2025&page=1&limit=3" >/dev/null

echo "deployment completed successfully"