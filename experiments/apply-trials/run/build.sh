#!/usr/bin/env bash
# Build the CLI and runtime images of one commit with the demonstration harness, plus the Python
# example image on that base. Usage: build.sh SHA
# Env: MARKITECT_SOURCE_REPO (a checkout whose refs hold SHA), TRIALS_WORK (default ~/mk-tr).
set -euo pipefail
SHA="$1"
WORK="${TRIALS_WORK:-$HOME/mk-tr}"
export MARKITECT_DEMO_WORK="$WORK"
HERE="$(cd "$(dirname "$0")" && pwd)"
TAG="${SHA:0:12}"
BUILD="$WORK/build-$TAG"
mkdir -p "$WORK"
bash "$HERE/../../recursive-apply-demo/ra_build.sh" "$SHA" "$BUILD"
SRC="$WORK/src-$SHA"
docker build --quiet -f "$SRC/packaging/runtime/examples/python/Dockerfile" --build-arg "BASE=markitect-runtime:ra-$TAG" \
  -t "markitect-runtime-python:ra-$TAG" "$SRC" > "$BUILD/build-runtime-python.log" 2>&1
echo "python_image=$(docker image inspect --format '{{.Id}}' "markitect-runtime-python:ra-$TAG")" >> "$BUILD/build.txt"
cat "$BUILD/build.txt"
