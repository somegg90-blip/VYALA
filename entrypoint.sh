#!/usr/bin/env bash
set -euo pipefail
echo "vyala: entrypoint started" >&2

git config --global --add safe.directory "${GITHUB_WORKSPACE:-/github/workspace}" || true
echo "vyala: safe directory set" >&2

if [ -z "${GITHUB_EVENT_PATH:-}" ] || [ ! -f "${GITHUB_EVENT_PATH:-}" ]; then
  echo "vyala: GITHUB_EVENT_PATH not found" >&2
  exit 1
fi

BASE_SHA=$(python3 -c "import json; print(json.load(open('${GITHUB_EVENT_PATH}'))['pull_request']['base']['sha'])")
echo "vyala: base SHA = $BASE_SHA" >&2

if ! git -C "${GITHUB_WORKSPACE}" cat-file -e "${BASE_SHA}^{commit}" 2>/dev/null; then
  echo "vyala: fetching base commit" >&2
  git -C "${GITHUB_WORKSPACE}" fetch --depth=1 --no-tags origin "${BASE_SHA}" 2>/dev/null \
    || git -C "${GITHUB_WORKSPACE}" fetch --unshallow --no-tags origin
fi

ARGS=(
  -path "${GITHUB_WORKSPACE}"
  -diff-base "${BASE_SHA}"
  -severity-threshold "${INPUT_SEVERITY_THRESHOLD:-medium}"
  -post-pr-comment
  -json /tmp/vyala-cbom.json
)

if [ -n "${INPUT_FAIL_ON:-}" ]; then
  ARGS+=(-fail-on "${INPUT_FAIL_ON}")
fi

echo "vyala: running: /usr/local/bin/vyala ${ARGS[*]}" >&2
exec /usr/local/bin/vyala "${ARGS[@]}"
