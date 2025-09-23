#!/usr/bin/env bash
set -euo pipefail

# Prefer mounted /tests; fallback to baked-in /app
if [[ -d "/tests" ]]; then
  WORKDIR="/tests"
else
  WORKDIR="/app"
fi

cd "$WORKDIR"

# Load .env if present
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi

# If first arg is a known test file name or empty, use run-loadtest.sh
if [[ $# -eq 0 ]]; then
  exec bash ./run-loadtest.sh
fi

case "$1" in
  *.js)
    # Run k6 directly with provided script
    shift
    exec k6 run "$@"
    ;;
  run|run-all)
    shift || true
    exec bash ./run-loadtest.sh "$@"
    ;;
  *)
    # Pass through to k6 for custom commands
    exec k6 "$@"
    ;;
esac
