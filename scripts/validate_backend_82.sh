#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

REMOTE_HOST="${REMOTE_HOST:-82.156.54.232}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/gaokao-ai-backend/backend}"
TEMP_PORT="${TEMP_PORT:-18083}"
BUILD_OUTPUT="${BUILD_OUTPUT:-gaokao-api.new}"
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

echo "[1/2] starting candidate on temporary port ${TEMP_PORT}"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "REMOTE_DIR='${REMOTE_DIR}' TEMP_PORT='${TEMP_PORT}' BUILD_OUTPUT='${BUILD_OUTPUT}' bash -s" <<'EOF'
set -euo pipefail

wait_for_port_release() {
  local port="$1"
  local pid="${2:-}"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if ! lsof -tiTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  if [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1; then
    kill -9 "${pid}" || true
  fi
  for _ in 1 2 3 4 5; do
    if ! lsof -tiTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

cd "${REMOTE_DIR}"
if [[ ! -f "${BUILD_OUTPUT}" ]]; then
  echo "candidate binary not found: ${BUILD_OUTPUT}" >&2
  exit 1
fi

existing_pid="$(lsof -tiTCP:${TEMP_PORT} -sTCP:LISTEN || true)"
if [[ -n "${existing_pid}" ]]; then
  kill "${existing_pid}" || true
  if ! wait_for_port_release "${TEMP_PORT}" "${existing_pid}"; then
    echo "temporary port ${TEMP_PORT} did not release after stopping pid ${existing_pid}" >&2
    lsof -iTCP:${TEMP_PORT} -sTCP:LISTEN -n -P >&2 || true
    exit 1
  fi
fi

nohup env SERVER_ADDR=":${TEMP_PORT}" "./${BUILD_OUTPUT}" >/tmp/gaokao-api-${TEMP_PORT}.log 2>&1 &
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:${TEMP_PORT}/healthz" >/dev/null; then
    exit 0
  fi
  sleep 1
done

echo "temporary validation failed" >&2
tail -n 60 /tmp/gaokao-api-${TEMP_PORT}.log >&2 || true
exit 1
EOF

echo "[2/2] verifying temporary candidate endpoints"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "TEMP_PORT='${TEMP_PORT}' bash -s" <<'EOF'
set -euo pipefail
curl -fsS "http://127.0.0.1:${TEMP_PORT}/healthz" >/dev/null
curl -fsS "http://127.0.0.1:${TEMP_PORT}/api/colleges?province=%E9%BB%91%E9%BE%99%E6%B1%9F&subject=%E5%8E%86%E5%8F%B2&year=2025&page=1&limit=3" >/dev/null
EOF

echo "temporary validation completed successfully"