#!/usr/bin/env bash
# Build the candidate of a Model assessment series: the trial kit's CLI, runtime
# and Python images, plus the grant-gated assessment harness binary from the
# same clean clone. Usage: build.sh SHA
# Env: MARKITECT_SOURCE_REPO (a checkout whose refs hold SHA), ASSESS_WORK
# (default ~/mk-model-assessment); Go and Docker on PATH.
set -euo pipefail
SHA="$1"
WORK="${ASSESS_WORK:-$HOME/mk-model-assessment}"
HERE="$(cd "$(dirname "$0")" && pwd)"
TRIALS_WORK="$WORK" bash "$HERE/../../apply-trials/run/build.sh" "$SHA" > /dev/null
SRC="$WORK/src-$SHA"
BUILD="$WORK/build-${SHA:0:12}"
( cd "$SRC" && test "$(git rev-parse HEAD)" = "$SHA" && test -z "$(git status --porcelain)" \
  && go test -c -o "$BUILD/assessment.test" ./internal/bootstrap )
echo "assessment_harness_sha256=$(sha256sum "$BUILD/assessment.test" | cut -d' ' -f1)" >> "$BUILD/build.txt"
cat "$BUILD/build.txt"
