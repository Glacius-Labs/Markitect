#!/usr/bin/env bash
# Create the Library M1 target repository: protected model, dependency and
# control inputs only, committed as the requirement baseline.
# Usage: ra_project.sh SRC TARGET
set -euo pipefail
SRC="$1"
TARGET="$2"
DEMO="$(cd "$(dirname "$0")" && pwd)"
test ! -e "$TARGET"
mkdir -p "$TARGET/.markitect"
M1="$SRC/docs/design/canonical-model/library-example/m1"
R2="$SRC/experiments/test-projects/library-r2/project/.markitect"
cp "$M1/model.yaml" "$M1/model-lock.yaml" "$TARGET/.markitect/"
cp -r "$M1/model" "$TARGET/.markitect/model"
cp -r "$R2/dependencies" "$TARGET/.markitect/dependencies"
cmp "$SRC/docs/design/canonical-model/library-example/vocabulary/model.yaml" "$TARGET/.markitect/dependencies/software/model.yaml"
diff -r "$SRC/docs/design/canonical-model/library-example/vocabulary/model/software" "$TARGET/.markitect/dependencies/software/model/software"
install -m 0644 "$DEMO/library-execution.yaml" "$TARGET/.markitect/execution.yaml"
cd "$TARGET"
git init --quiet --initial-branch=main
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" add .
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" commit --quiet -m "Library M1 requirement baseline"
git rev-parse HEAD
