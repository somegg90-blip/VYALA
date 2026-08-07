#!/usr/bin/env bash
set -euo pipefail

# FIX: GitHub Actions forces HOME=/github/home, which is owned by the runner (UID 1001).
# Since we run as the 'vyala' user (UID 1000), Semgrep gets a Permission Denied error
# when it tries to create its ~/.semgrep log directory.
# Overriding HOME to /tmp solves this cleanly without compromising security.
export HOME=/tmp

# If args are provided (like "scan --path . --upload-url ..."), run them directly.
# The Go binary handles all the logic.
if [ -n "${INPUT_ARGS:-}" ]; then
  echo "Running VYALA with args: ${INPUT_ARGS}"
  exec /usr/local/bin/vyala ${INPUT_ARGS}
fi

# ---------- Comment-from-file mode ----------
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

EVENT_NAME="${GITHUB_EVENT_NAME:-}"
BASE_SHA=""

if [ "$EVENT_NAME" = "pull_request" ] || [ "$EVENT_NAME" = "pull_request_target" ]; then
  BASE_SHA=$(python3 -c "import json; print(json.load(open('${GITHUB_EVENT_PATH}'))['pull_request']['base']['sha'])")
elif [ "$EVENT_NAME" = "push" ]; then
  BASE_SHA=$(python3 -c "import json; print(json.load(open('${GITHUB_EVENT_PATH}')).get('before', ''))")
else
  echo "vyala: Unsupported event type '${EVENT_NAME}' for diff scanning. Running full scan." >&2
  BASE_SHA=""
fi

if [ -n "$BASE_SHA" ]; then
  if ! git -C "${GITHUB_WORKSPACE}" cat-file -e "${BASE_SHA}^{commit}" 2>/dev/null; then
    echo "Fetching missing base commit ${BASE_SHA}..."
    git -C "${GITHUB_WORKSPACE}" fetch --depth=1 --no-tags origin "${BASE_SHA}" 2>/dev/null \
      || git -C "${GITHUB_WORKSPACE}" fetch --unshallow --no-tags origin 2>/dev/null || true
  fi
fi

ARGS=(
  -path "${GITHUB_WORKSPACE}"
  -severity-threshold "${INPUT_SEVERITY_THRESHOLD:-medium}"
  -json "${GITHUB_WORKSPACE}/vyala-cbom.json"
)

if [ -n "$BASE_SHA" ]; then
  ARGS+=(-diff-base "${BASE_SHA}")
fi

if [ "${POST_PR_COMMENT:-true}" = "true" ]; then
  ARGS+=(-post-pr-comment)
fi

if [ -n "${INPUT_FAIL_ON:-}" ]; then
  ARGS+=(-fail-on "${INPUT_FAIL_ON}")
fi

exec /usr/local/bin/vyala "${ARGS[@]}"