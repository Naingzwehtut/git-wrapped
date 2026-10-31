#!/usr/bin/env bash
# Cross-language test: Go, Python and JavaScript must produce byte-identical
# terminal output, JSON and SVG for every case, matching tests/golden/.
#
#   tests/run.sh            run all checks
#   tests/run.sh --update   regenerate golden files from the Python version first
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$HERE")"
GOLDEN="$HERE/golden"
UPDATE=0
[[ "${1:-}" == "--update" ]] && UPDATE=1

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
"$HERE/make_fixture.sh" "$WORK/fx" > /dev/null
(cd "$ROOT/go" && go build -o "$WORK/gw" ./cmd/git-wrapped)

run_impl() { # <impl> <args...>
  local impl="$1"; shift
  case "$impl" in
    python) PYTHONPATH="$ROOT/python/src" python3 -m git_wrapped "$@" ;;
    js)     node "$ROOT/js/bin/git-wrapped.js" "$@" ;;
    go)     "$WORK/gw" "$@" ;;
  esac
}

# name | working dir (relative to fixture) | args
CASES=(
  "default|app|--year 2026"
  "default_json|app|--year 2026 --json"
  "everyone_multi|.|--year 2026 --everyone"
  "everyone_multi_json|.|--year 2026 --everyone --json"
  "author_unicode|.|--year 2026 --author zoë"
  "author_eq_syntax|.|--year=2026 --author=NAING tools app"
  "empty_year|app|--year 2020"
  "empty_year_json|app|--year 2020 --json"
)

mkdir -p "$GOLDEN"
fails=0
for c in "${CASES[@]}"; do
  IFS='|' read -r name dir args <<< "$c"
  # shellcheck disable=SC2086
  for impl in python js go; do
    (cd "$WORK/fx/$dir" && run_impl "$impl" $args --svg "$WORK/$name.$impl.svg") \
      > "$WORK/$name.$impl.out" 2> /dev/null
  done
  if [[ $UPDATE == 1 ]]; then
    cp "$WORK/$name.python.out" "$GOLDEN/$name.out"
    cp "$WORK/$name.python.svg" "$GOLDEN/$name.svg"
  fi
  for impl in python js go; do
    if cmp -s "$WORK/$name.$impl.out" "$GOLDEN/$name.out" && cmp -s "$WORK/$name.$impl.svg" "$GOLDEN/$name.svg"; then
      printf "  ok    %-6s %s\n" "$impl" "$name"
    else
      printf "  FAIL  %-6s %s\n" "$impl" "$name"
      diff "$GOLDEN/$name.out" "$WORK/$name.$impl.out" | head -20 || true
      diff "$GOLDEN/$name.svg" "$WORK/$name.$impl.svg" | head -20 || true
      fails=$((fails + 1))
    fi
  done
done

# Exit codes for bad input must match too
for bad in "--nope" "--year abc" "--year" "/definitely/not/here" "--everyone=1"; do
  codes=""
  for impl in python js go; do
    # shellcheck disable=SC2086
    set +e; (cd "$WORK/fx" && run_impl "$impl" $bad) > /dev/null 2>&1; codes+="$? "; set -e
  done
  if [[ "$(echo $codes | tr ' ' '\n' | sort -u | wc -l)" == 1 && "${codes%% *}" != 0 ]]; then
    printf "  ok    exit   %-22s (%s)\n" "$bad" "${codes% }"
  else
    printf "  FAIL  exit   %-22s (%s)\n" "$bad" "${codes% }"
    fails=$((fails + 1))
  fi
done

if [[ $fails -gt 0 ]]; then
  echo "$fails check(s) failed"
  exit 1
fi
echo "All checks passed."
