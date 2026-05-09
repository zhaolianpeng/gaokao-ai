#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SYNC_SCRIPT="${SYNC_SCRIPT:-${SCRIPT_DIR}/sync_backend_82.sh}"
VALIDATE_SCRIPT="${VALIDATE_SCRIPT:-${SCRIPT_DIR}/validate_backend_82.sh}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT:-${SCRIPT_DIR}/deploy_backend_82.sh}"

require_executable() {
  if [[ ! -x "$1" ]]; then
    echo "script not found or not executable: $1" >&2
    exit 1
  fi
}

if [[ -z "${SSHPASS:-}" ]]; then
  echo "SSHPASS is required" >&2
  exit 1
fi

require_executable "${SYNC_SCRIPT}"
require_executable "${VALIDATE_SCRIPT}"
require_executable "${DEPLOY_SCRIPT}"

echo "[1/3] sync backend source"
"${SYNC_SCRIPT}"

echo "[2/3] validate candidate build"
"${VALIDATE_SCRIPT}"

echo "[3/3] deploy validated candidate"
"${DEPLOY_SCRIPT}"

echo "release pipeline completed successfully"