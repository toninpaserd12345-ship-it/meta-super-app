#!/usr/bin/env bash

set -u
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-quick}"
LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/meta-super-app-check.XXXXXX")"
GO_CACHE_DIR="${TMPDIR:-/tmp}/meta-super-app-go-cache"

passed=0

run_check() {
  local label="$1"
  local workdir="$2"
  shift 2
  local log_file="${LOG_DIR}/${passed}.log"

  if (cd "$workdir" && "$@") >"$log_file" 2>&1; then
    printf 'PASS  %s\n' "$label"
    passed=$((passed + 1))
    return 0
  fi

  printf 'FAIL  %s\n' "$label"
  printf 'Log: %s\n' "$log_file"
  tail -n 80 "$log_file"
  exit 1
}

case "$MODE" in
  quick|full) ;;
  *)
    printf 'Usage: %s [quick|full]\n' "$0"
    exit 2
    ;;
esac

run_check "Backend tests" "$ROOT_DIR/backend" env GOCACHE="$GO_CACHE_DIR" go test ./...
run_check "Frontend typecheck" "$ROOT_DIR/frontend" npm run typecheck
run_check "Source diff" "$ROOT_DIR" git diff --check

if [[ "$MODE" == "full" ]]; then
  run_check "Backend vet" "$ROOT_DIR/backend" env GOCACHE="$GO_CACHE_DIR" go vet ./...
  run_check "Backend build" "$ROOT_DIR/backend" env GOCACHE="$GO_CACHE_DIR" go build ./cmd/api
  run_check "Frontend production build" "$ROOT_DIR/frontend" npm run build
fi

rm -rf "$LOG_DIR"
printf 'OK    %s checks passed (%d)\n' "$MODE" "$passed"
