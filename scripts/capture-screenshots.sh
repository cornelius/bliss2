#!/usr/bin/env bash
# capture-screenshots.sh — rebuild test store and capture output of every
# relevant command into screenshots/.
#
# Usage: bash scripts/capture-screenshots.sh
#
# Run this before and after a redesign to see what changed.

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
B="$REPO/bliss"
SHOTS="$REPO/screenshots"

ROOT=/tmp/bliss-test
STORE=$ROOT/store

CTX_API=$ROOT/api
CTX_FRONTEND=$ROOT/frontend
CTX_INFRA=$ROOT/infra
CTX_MOBILE=$ROOT/mobile

# ── build ─────────────────────────────────────────────────────────────────────
echo "--- building"
(cd "$REPO" && go build -o bliss ./cmd/bliss/)

# ── seed ──────────────────────────────────────────────────────────────────────
echo "--- seeding test store"
bash "$REPO/scripts/seed-test.sh" > /dev/null

export BLISS_STORE=$STORE
mkdir -p "$SHOTS"

cap() {
  # cap <filename> <dir> <bliss args...>
  local file=$1 dir=$2; shift 2
  echo "  $file"
  (cd "$dir" && "$B" "$@") > "$SHOTS/$file" 2>&1 || true
}

# ── status ────────────────────────────────────────────────────────────────────
cap status-ctx.txt     "$CTX_API"  status
cap status-personal.txt /tmp       status

# ── list (full, per context) ──────────────────────────────────────────────────
cap list-api.txt      "$CTX_API"      list
cap list-frontend.txt "$CTX_FRONTEND" list
cap list-infra.txt    "$CTX_INFRA"    list
cap list-mobile.txt   "$CTX_MOBILE"   list
cap list-personal.txt /tmp            list

# ── list (filtered) ───────────────────────────────────────────────────────────
cap list-filter-today.txt     "$CTX_API" list today
cap list-filter-this-week.txt "$CTX_API" list this-week
cap list-filter-bugs.txt      "$CTX_API" list bugs
cap list-filter-inbox.txt     "$CTX_API" list inbox

# ── list --all ────────────────────────────────────────────────────────────────
cap list-all.txt "$CTX_API" list --all

# ── add ───────────────────────────────────────────────────────────────────────
cap add-inbox.txt   "$CTX_API" add "Check in with security team"
cap add-to-list.txt "$CTX_API" add "Review PR #412" --list today
cap add-urgent.txt  "$CTX_API" add "Hotfix: prod is down" --list today --urgent

# ── done / move ───────────────────────────────────────────────────────────────
# Re-list to refresh session, then complete item 1.
(cd "$CTX_API" && "$B" list > /dev/null)
cap done.txt "$CTX_API" done 1
(cd "$CTX_API" && "$B" list > /dev/null)
cap move.txt "$CTX_API" move 1 --list later

# ── history ───────────────────────────────────────────────────────────────────
cap history-ctx.txt "$CTX_API" history
cap history-all.txt "$CTX_API" history --all

echo ""
echo "Screenshots written to $SHOTS/"
