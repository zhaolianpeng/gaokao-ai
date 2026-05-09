#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKEND_DIR="${BACKEND_DIR:-${REPO_ROOT}/backend}"
SYNC_SCRIPT="${SYNC_SCRIPT:-${SCRIPT_DIR}/sync_backend_82.sh}"
POLL_SECONDS="${POLL_SECONDS:-2}"

if [[ ! -x "${SYNC_SCRIPT}" ]]; then
  echo "sync script not found or not executable: ${SYNC_SCRIPT}" >&2
  exit 1
fi

snapshot() {
  find "${BACKEND_DIR}" \
    -type f \
    ! -name '.env' \
    ! -name '.env.local' \
    ! -path '*/certs/*' \
    ! -name 'gaokao-api' \
    ! -name 'gaokao-api.new' \
    ! -name 'gaokao-api-canary' \
    ! -name 'gaokao-api-linux' \
    -print0 \
    | sort -z \
    | xargs -0 shasum
}

last_snapshot="$(snapshot)"
echo "watching ${BACKEND_DIR}"
echo "poll interval: ${POLL_SECONDS}s"
echo "press Ctrl+C to stop"

while true; do
  sleep "${POLL_SECONDS}"
  current_snapshot="$(snapshot)"
  if [[ "${current_snapshot}" == "${last_snapshot}" ]]; then
    continue
  fi

  echo "change detected at $(date '+%F %T')"
  if "${SYNC_SCRIPT}"; then
    last_snapshot="${current_snapshot}"
    echo "sync completed"
  else
    echo "sync failed; waiting for next change to retry" >&2
  fi
done