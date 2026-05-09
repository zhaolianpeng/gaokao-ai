#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

REMOTE_HOST="${REMOTE_HOST:-82.156.54.232}"
REMOTE_USER="${REMOTE_USER:-ubuntu}"
REMOTE_DIR="${REMOTE_DIR:-/home/ubuntu/gaokao-ai-backend/backend}"
REMOTE_GO_BIN="${REMOTE_GO_BIN:-/home/ubuntu/.local/go/bin/go}"
LOCAL_BACKEND_DIR="${LOCAL_BACKEND_DIR:-${REPO_ROOT}/backend}"
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

if [[ ! -d "${LOCAL_BACKEND_DIR}" ]]; then
  echo "backend directory not found: ${LOCAL_BACKEND_DIR}" >&2
  exit 1
fi

require_command sshpass
require_command rsync

echo "[1/3] syncing backend source to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}"
SSHPASS="${SSH_PASSWORD}" sshpass -e rsync -az --delete \
  --exclude '.env' \
  --exclude '.env.local' \
  --exclude 'certs/' \
  --exclude 'gaokao-api' \
  --exclude 'gaokao-api.new' \
  --exclude 'gaokao-api-canary' \
  --exclude 'gaokao-api-linux' \
  -e "ssh ${SSH_OPTS[*]}" \
  "${LOCAL_BACKEND_DIR}/" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DIR}/"

echo "[2/3] verifying remote Go environment"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "bash -lc '${REMOTE_GO_BIN} version'"

echo "[3/3] building backend on server"
SSHPASS="${SSH_PASSWORD}" sshpass -e ssh "${SSH_OPTS[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "bash -lc 'cd ${REMOTE_DIR} && ${REMOTE_GO_BIN} build ./... && ${REMOTE_GO_BIN} build -o ${BUILD_OUTPUT} ./cmd/main.go && ls -lh ${BUILD_OUTPUT}'"

echo "sync and remote build completed"