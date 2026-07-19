#!/usr/bin/env bash
set -euo pipefail

# ---------- Comment‑from‑file mode ----------
if [ -n "${CBOM_FILE:-}" ]; then
  exec /usr/local/bin/vyala \
    -comment-from-file "${CBOM_FILE}" \
    -post-pr-comment \
    -pr-number "${PR_NUMBER}" \
    -head-sha "${HEAD_SHA}" \
    -severity-threshold "${INPUT_SEVERITY_THRESHOLD:-medium}"
fi

# ---------- Normal scan mode ----------
git config --global --add safe.directory "${GITHUB_WORKSPACE:-/github/workspace}" || true

if [ -z "${GITHUB_EVENT_PATH:-}" ] || [ ! -f "${GITHUB_EVENT_PATH:-}" ]; then
  echo "vyala: GITHUB_EVENT_PATH not found" >&2
  exit 1
fi

BASE_SHA=$(python3 -c "import json; print(json.load(open('${GITHUB_EVENT_PATH}'))['pull_request']['base']['sha'])")

if ! git -C "${GITHUB_WORKSPACE}" cat-file -e "${BASE_SHA}^{commit}" 2>/dev/null; then
  git -C "${GITHUB_WORKSPACE}" fetch --depth=1 --no-tags origin "${BASE_SHA}" 2>/dev/null \
    || git -C "${GITHUB_WORKSPACE}" fetch --unshallow --no-tags origin
fi

ARGS=(
  -path "${GITHUB_WORKSPACE}"
  -diff-base "${BASE_SHA}"
  -severity-threshold "${INPUT_SEVERITY_THRESHOLD:-medium}"
  -json "${GITHUB_WORKSPACE}/vyala-cbom.json"
)

if [ "${POST_PR_COMMENT:-true}" = "true" ]; then
  ARGS+=(-post-pr-comment)
fi

if [ -n "${INPUT_FAIL_ON:-}" ]; then
  ARGS+=(-fail-on "${INPUT_FAIL_ON}")
fi

exec /usr/local/bin/vyala "${ARGS[@]}"
