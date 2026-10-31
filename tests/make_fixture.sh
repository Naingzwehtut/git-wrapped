#!/usr/bin/env bash
# Builds a deterministic set of fixture repos in $1 (default: tests/.fixture).
#   <dir>/app    main repo, two authors, many edge cases
#   <dir>/tools  second repo, so multi-repo discovery is exercised
set -euo pipefail

ROOT="${1:-$(dirname "$0")/.fixture}"
rm -rf "$ROOT"
mkdir -p "$ROOT"
ROOT="$(cd "$ROOT" && pwd)"

ME_NAME="Naing Zwe Htut"; ME_EMAIL="naing@example.com"
YOU_NAME="Zoë Müller";    YOU_EMAIL="zoe@example.com"

# commit <iso-date> <name> <email> <message>
commit() {
  git add -A
  GIT_AUTHOR_DATE="$1" GIT_COMMITTER_DATE="$1" \
    git -c user.name="$2" -c user.email="$3" -c commit.gpgsign=false \
    commit -q --allow-empty -m "$4"
}
# add <file> <count> <tag>  — append <count> lines
add() { mkdir -p "$(dirname "$1")"; for i in $(seq 1 "$2"); do echo "$3 line $i" >> "$1"; done; }
# drop <file> <count>        — delete the first <count> lines (portable, no sed -i)
drop() { tail -n +"$(( $2 + 1 ))" "$1" > "$1.tmp"; mv "$1.tmp" "$1"; }

init() {
  mkdir -p "$1"; cd "$1"
  git init -q -b main
  git config user.email "$ME_EMAIL"
  git config user.name "$ME_NAME"
}

# ---------- app ----------
init "$ROOT/app"

# Late 2025 commit: inside the git --since window but must be filtered out by year
add README.md 5 intro;              commit "2025-12-30T10:00:00+07:00" "$ME_NAME" "$ME_EMAIL" "Initial commit"

# A 14-day streak of late-night Go work in March (Bangkok time)
for d in $(seq 1 14); do
  day=$(printf "%02d" "$d")
  add src/main.go $(( d + 3 )) "night$d"
  commit "2026-03-${day}T23:${day}:00+07:00" "$ME_NAME" "$ME_EMAIL" "Add feature $d"
done
# Busiest day: 4 more commits on Mar 10 at 1am
for k in 1 2 3 4; do
  add src/util.go 2 "util$k"
  commit "2026-03-10T01:1${k}:00+07:00" "$ME_NAME" "$ME_EMAIL" "fix: edge case $k"
done

# Weekend work in New York time (Sat 2026-06-06, Sun 2026-06-07)
add web/app.ts 30 ts;                commit "2026-06-06T14:00:00-04:00" "$ME_NAME" "$ME_EMAIL" "Build web dashboard"
add web/style.css 12 css;            commit "2026-06-07T15:30:00-04:00" "$ME_NAME" "$ME_EMAIL" "Style the dashboard"

# Lockfile and node_modules churn must be ignored
add web/package-lock.json 500 lock
add web/node_modules/left-pad/index.js 50 dep
add web/app.ts 3 ts2;                commit "2026-06-08T09:00:00-04:00" "$ME_NAME" "$ME_EMAIL" "Add deps & bump <things>"

# Binary file (numstat shows "-")
printf '\x89PNG\x00\x01\x02' > logo.png
commit "2026-07-01T11:00:00+07:00" "$ME_NAME" "$ME_EMAIL" "Add logo"

# File with a space, a dotfile, Dockerfile, deletions
add "docs/my notes.md" 20 notes
add .env.example 2 env
add Dockerfile 4 docker;             commit "2026-08-15T09:45:00+07:00" "$ME_NAME" "$ME_EMAIL" "Docs, docker and config"
drop src/main.go 40;                 commit "2026-08-16T10:00:00+07:00" "$ME_NAME" "$ME_EMAIL" "Refactor: remove dead code from the main loop, it was never called by anything anymore"

# Authored 23:30 on Dec 31 in UTC-5: local year is 2026 even though it is 2027 in UTC
add src/main.go 1 nye;               commit "2026-12-31T23:30:00-05:00" "$ME_NAME" "$ME_EMAIL" "Last one of the year"

# Second author on a branch, then merged (merge commit must be skipped)
git checkout -q -b feature
add lib/parser.py 25 py;             commit "2026-04-02T10:00:00+02:00" "$YOU_NAME" "$YOU_EMAIL" "Add parser"
add lib/parser.py 5 py2;             commit "2026-04-03T11:00:00+02:00" "$YOU_NAME" "$YOU_EMAIL" "Fix bug in parser"
git checkout -q main
GIT_AUTHOR_DATE="2026-04-04T12:00:00+07:00" GIT_COMMITTER_DATE="2026-04-04T12:00:00+07:00" \
  git -c user.name="$ME_NAME" -c user.email="$ME_EMAIL" merge -q --no-ff feature -m "Merge feature"

# Unmerged branch commit (still counted: we read all branches)
git checkout -q -b experiment
add src/exp.rs 8 rust;               commit "2026-09-09T09:09:00+07:00" "$ME_NAME" "$ME_EMAIL" "Try Rust"
git checkout -q main

# ---------- tools ----------
init "$ROOT/tools"
add scripts/deploy.sh 10 sh;         commit "2026-02-01T08:00:00+07:00" "$ME_NAME" "$ME_EMAIL" "Deploy script"
add scripts/report.py 18 py;         commit "2026-02-02T08:30:00+07:00" "$ME_NAME" "$ME_EMAIL" "Weekly report"
add scripts/report.py 2 py2;         commit "2026-02-03T19:00:00+07:00" "$YOU_NAME" "$YOU_EMAIL" "Tweak report"

# A plain folder (not a repo) must be ignored by discovery
mkdir -p "$ROOT/notes" && echo hi > "$ROOT/notes/todo.txt"

echo "$ROOT"
