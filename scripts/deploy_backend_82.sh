#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

REMOTE_HOST="${REMOTE_HOST:-82.156.54.232}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/gaokao-ai-backend/backend}"
PROD_PORT="${PROD_PORT:-8080}"
PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-http://82.156.54.232:80}"
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
require_command curl

echo "[1/2] promoting remote candidate to production port ${PROD_PORT}"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "REMOTE_DIR='${REMOTE_DIR}' PROD_PORT='${PROD_PORT}' BUILD_OUTPUT='${BUILD_OUTPUT}' bash -s" <<'EOF'
set -euo pipefail

wait_for_port_release() {
  local port="$1"
  local pid="${2:-}"
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
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

systemd_service_exists() {
  local service_name="$1"
  systemctl list-unit-files "${service_name}" >/dev/null 2>&1
}

cd "${REMOTE_DIR}"
if [[ ! -f "${BUILD_OUTPUT}" ]]; then
  echo "candidate binary not found: ${BUILD_OUTPUT}" >&2
  exit 1
fi

timestamp="$(date +%Y%m%d_%H%M%S)"
if [[ -f gaokao-api ]]; then
  cp gaokao-api "gaokao-api.bak.${timestamp}"
fi
mv "${BUILD_OUTPUT}" gaokao-api
service_name="gaokao-api.service"
if systemd_service_exists "${service_name}"; then
  sudo systemctl stop "${service_name}" || true
  sudo systemctl reset-failed "${service_name}" || true
fi

current_pid="$(lsof -tiTCP:${PROD_PORT} -sTCP:LISTEN || true)"
if [[ -n "${current_pid}" ]]; then
  kill "${current_pid}" || true
fi
if ! wait_for_port_release "${PROD_PORT}" "${current_pid}"; then
  echo "production port ${PROD_PORT} did not release after stopping pid ${current_pid}" >&2
  lsof -iTCP:${PROD_PORT} -sTCP:LISTEN -n -P >&2 || true
  exit 1
fi

if systemd_service_exists "${service_name}"; then
  sudo systemctl start "${service_name}"
else
  nohup env SERVER_ADDR=":${PROD_PORT}" ./gaokao-api >/tmp/gaokao-api-${PROD_PORT}.log 2>&1 &
fi
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "http://127.0.0.1:${PROD_PORT}/healthz" >/dev/null; then
    exit 0
  fi
  sleep 1
done
echo "production health check failed" >&2
if systemd_service_exists "${service_name}"; then
  sudo systemctl --no-pager --full status "${service_name}" >&2 || true
  sudo journalctl -u "${service_name}" -n 80 --no-pager >&2 || true
else
  tail -n 80 /tmp/gaokao-api-${PROD_PORT}.log >&2 || true
fi
exit 1
EOF

echo "[2/2] verifying public endpoint"
curl -fsS "${PUBLIC_BASE_URL}/healthz" >/dev/null
curl -fsS "${PUBLIC_BASE_URL}/api/colleges?province=%E9%BB%91%E9%BE%99%E6%B1%9F&subject=%E5%8E%86%E5%8F%B2&year=2025&page=1&limit=3" >/dev/null

echo "deployment completed successfully"