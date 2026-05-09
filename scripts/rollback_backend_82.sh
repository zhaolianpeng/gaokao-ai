#!/usr/bin/env bash

set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-82.156.54.232}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/gaokao-ai-backend/backend}"
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

require_command sshpass
require_command curl

echo "[1/2] rolling back production binary on port ${PROD_PORT}"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "REMOTE_DIR='${REMOTE_DIR}' PROD_PORT='${PROD_PORT}' bash -s" <<'EOF'
set -euo pipefail
cd "${REMOTE_DIR}"

latest_backup="$(ls -1t gaokao-api.bak.* 2>/dev/null | head -n 1 || true)"
if [[ -z "${latest_backup}" ]]; then
  echo "no backup binary found under ${REMOTE_DIR}" >&2
  exit 1
fi

cp "${latest_backup}" gaokao-api
current_pid="$(lsof -tiTCP:${PROD_PORT} -sTCP:LISTEN || true)"
if [[ -n "${current_pid}" ]]; then
  kill "${current_pid}" || true
  sleep 1
fi

nohup env SERVER_ADDR=":${PROD_PORT}" ./gaokao-api >/tmp/gaokao-api-${PROD_PORT}.log 2>&1 &
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:${PROD_PORT}/healthz" >/dev/null; then
    echo "rollback source: ${latest_backup}"
    exit 0
  fi
  sleep 1
done

echo "rollback health check failed" >&2
tail -n 80 /tmp/gaokao-api-${PROD_PORT}.log >&2 || true
exit 1
EOF

echo "[2/2] verifying public endpoint after rollback"
curl -fsS "${PUBLIC_BASE_URL}/healthz" >/dev/null
curl -fsS "${PUBLIC_BASE_URL}/api/colleges?province=%E9%BB%91%E9%BE%99%E6%B1%9F&subject=%E5%8E%86%E5%8F%B2&year=2025&page=1&limit=3" >/dev/null

echo "rollback completed successfully"