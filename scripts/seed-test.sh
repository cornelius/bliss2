#!/usr/bin/env bash
# seed-test.sh — set up a reproducible test store with known paths.
# Usage: source scripts/seed-test.sh
#   or:  bash scripts/seed-test.sh && export BLISS_STORE=/tmp/bliss-test/store
#
# All paths are fixed under /tmp/bliss-test/ so they survive between shell
# sessions and can be referenced by name without losing track of UUIDs.

set -euo pipefail

ROOT=/tmp/bliss-test
STORE=$ROOT/store
B="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/bliss"

if [ ! -f "$B" ]; then
  echo "bliss binary not found at $B — run: go build -o bliss ./cmd/bliss/" >&2
  exit 1
fi

# Wipe and recreate
rm -rf "$ROOT"
mkdir -p "$ROOT"

export BLISS_STORE=$STORE

# Context project dirs (fixed paths)
CTX_API=$ROOT/api
CTX_FRONTEND=$ROOT/frontend
CTX_INFRA=$ROOT/infra
CTX_MOBILE=$ROOT/mobile
mkdir -p "$CTX_API" "$CTX_FRONTEND" "$CTX_INFRA" "$CTX_MOBILE"

echo "--- initializing contexts"
(cd "$CTX_API"      && "$B" init --name api)
(cd "$CTX_FRONTEND" && "$B" init --name frontend)
(cd "$CTX_INFRA"    && "$B" init --name infra)
(cd "$CTX_MOBILE"   && "$B" init --name mobile)

# ── api ──────────────────────────────────────────────────────────────────────
echo "--- seeding api"
(cd "$CTX_API" && \
  "$B" add "Implement JWT refresh token rotation"        --list today     && \
  "$B" add "Fix /users endpoint returning 500 on empty DB" --list today   && \
  "$B" add "Write OpenAPI spec for auth endpoints"       --list this-week && \
  "$B" add "Set up rate limiting middleware"             --list this-week && \
  "$B" add "Add integration tests for login flow"        --list this-week && \
  "$B" add "Profile slow SQL queries in production"      --list next-week && \
  "$B" add "Document deployment runbook"                 --list next-week && \
  "$B" add "Migrate from REST to GraphQL (spike)"        --list later     && \
  "$B" add "Evaluate Temporal for background jobs"       --list later     && \
  "$B" add "Fix CORS headers on OPTIONS preflight"       --list bugs      && \
  "$B" add "Session cookie not expiring correctly"       --list bugs      && \
  "$B" add "Race condition in concurrent token refresh"  --list bugs      && \
  "$B" add "Check in with DevOps about K8s quota increase"                && \
  "$B" add "Reply to sec audit findings email")

# ── frontend ─────────────────────────────────────────────────────────────────
echo "--- seeding frontend"
(cd "$CTX_FRONTEND" && \
  "$B" add "Migrate component library to shadcn/ui"          --list today     && \
  "$B" add "Fix mobile nav overflow on small screens"        --list today     && \
  "$B" add "Add dark mode toggle to settings page"           --list this-week && \
  "$B" add "Lazy-load dashboard charts"                      --list this-week && \
  "$B" add "Write Storybook stories for Button and Input"    --list next-week && \
  "$B" add "A/B test new onboarding flow"                    --list later     && \
  "$B" add "Audit bundle size — target <200 kB gzipped"      --list later     && \
  "$B" add "Dropdown closes on outside click in Safari"      --list bugs      && \
  "$B" add "Chart tooltip misaligned on retina displays"     --list bugs      && \
  "$B" add "Ask design for updated error state illustrations"                 && \
  "$B" add "Sync with PM on Q2 roadmap priorities")

# ── infra ─────────────────────────────────────────────────────────────────────
echo "--- seeding infra"
(cd "$CTX_INFRA" && \
  "$B" add "Upgrade Kubernetes to 1.30"                      --list today     && \
  "$B" add "Rotate expiring TLS certificates"                --list today     && \
  "$B" add "Set up Loki log aggregation"                     --list this-week && \
  "$B" add "Write runbook for on-call rotation"              --list this-week && \
  "$B" add "Migrate CI to self-hosted runners"               --list next-week && \
  "$B" add "Set up cost alerting in AWS"                     --list next-week && \
  "$B" add "Evaluate Pulumi vs Terraform for new infra"      --list later     && \
  "$B" add "Clean up unused S3 buckets"                      --list later     && \
  "$B" add "Bump base Docker images to Debian bookworm"                       && \
  "$B" add "Check Dependabot alerts on infra repo")

# ── mobile ───────────────────────────────────────────────────────────────────
echo "--- seeding mobile"
(cd "$CTX_MOBILE" && \
  "$B" add "Fix crash on deep link when app is backgrounded" --list today     && \
  "$B" add "Submit build to TestFlight"                      --list today     && \
  "$B" add "Implement biometric auth fallback"               --list this-week && \
  "$B" add "Add offline mode for core screens"               --list this-week && \
  "$B" add "Reduce cold start time below 800 ms"             --list next-week && \
  "$B" add "Write UI tests for checkout flow"                --list next-week && \
  "$B" add "Investigate memory leak in image cache"          --list bugs      && \
  "$B" add "Push notification not delivered on Android 14"   --list bugs      && \
  "$B" add "Keyboard covers input on older iPhones"          --list bugs      && \
  "$B" add "Port to iPad layout"                             --list later     && \
  "$B" add "Reply to App Store review about login issue"                      && \
  "$B" add "Check analytics drop on v2.3 release")

# ── personal (no context) ─────────────────────────────────────────────────────
echo "--- seeding personal"
(cd /tmp && \
  "$B" add "Renew car insurance"       --list errands  && \
  "$B" add "Pick up prescription"      --list errands  && \
  "$B" add "Book dentist appointment"  --list errands  && \
  "$B" add "Read «Shape Up»"           --list someday  && \
  "$B" add "Learn Nix"                 --list someday  && \
  "$B" add "Try out Ghostty terminal"  --list someday  && \
  "$B" add "Set up home NAS with ZFS"  --list someday  && \
  "$B" add "Call landlord about heating"               && \
  "$B" add "Buy birthday gift for mom")

# ── add list sections via direct file edits ───────────────────────────────────
echo "--- adding sections"

add_section() {
  # add_section <file> <after-index (0-based)> <section-name>
  # Inserts "--- <name>" after item at given index.
  local file=$1 after=$2 name=$3
  local uuids=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && uuids+=("$line")
  done < "$file"

  {
    for i in "${!uuids[@]}"; do
      echo "${uuids[$i]}"
      if [[ $i -eq $after ]]; then
        if [[ -n "$name" ]]; then
          echo "--- $name"
        else
          echo "---"
        fi
      fi
    done
  } > "$file"
}

API_UUID=$(cat "$CTX_API/.bliss-context")
FE_UUID=$(cat "$CTX_FRONTEND/.bliss-context")
INFRA_UUID=$(cat "$CTX_INFRA/.bliss-context")
MOBILE_UUID=$(cat "$CTX_MOBILE/.bliss-context")

# api: this-week — first 2 items then "--- backlog"
add_section "$STORE/contexts/$API_UUID/lists/this-week.txt"   1 "backlog"
# api: bugs — first 2 items then "--- nice to fix"
add_section "$STORE/contexts/$API_UUID/lists/bugs.txt"        1 "nice to fix"
# frontend: this-week — unnamed divider after first item
add_section "$STORE/contexts/$FE_UUID/lists/this-week.txt"    0 ""
# infra: this-week — "--- review" after first item
add_section "$STORE/contexts/$INFRA_UUID/lists/this-week.txt" 0 "review"
# mobile: bugs — "--- low priority" after first item
add_section "$STORE/contexts/$MOBILE_UUID/lists/bugs.txt"     0 "low priority"

# Commit the section edits
(cd "$STORE" && git add -A && git commit --author="bliss <bliss@local>" -m "bliss: add list sections")

echo ""
echo "Done. Test store is ready."
echo ""
echo "  export BLISS_STORE=$STORE"
echo ""
echo "  api:      cd $CTX_API      && bliss list"
echo "  frontend: cd $CTX_FRONTEND && bliss list"
echo "  infra:    cd $CTX_INFRA    && bliss list"
echo "  mobile:   cd $CTX_MOBILE   && bliss list"
echo "  personal: cd /tmp          && bliss list"
