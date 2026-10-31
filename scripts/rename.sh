#!/usr/bin/env bash
# Replace the YOUR_USERNAME placeholder with your GitHub username everywhere.
#   scripts/rename.sh <github-username>
set -euo pipefail
[[ $# -eq 1 ]] || { echo "usage: $0 <github-username>"; exit 1; }
cd "$(dirname "$0")/.."
grep -rl --exclude-dir=.git --exclude=rename.sh YOUR_USERNAME . | while read -r f; do
  perl -pi -e "s/YOUR_USERNAME/$1/g" "$f"
  echo "updated $f"
done
if command -v rsvg-convert >/dev/null; then
  rsvg-convert -w 1200 assets/example.svg -o assets/example.png && echo "re-rendered assets/example.png"
else
  echo "note: install librsvg and run: rsvg-convert -w 1200 assets/example.svg -o assets/example.png"
fi
