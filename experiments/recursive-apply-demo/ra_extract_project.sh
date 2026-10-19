#!/usr/bin/env bash
# Create a case-3 target: the published library realization of an earlier
# demonstration without any model or registry, plus the imported vocabulary, a
# structural manifest and lock with no local Definitions, and the Extract
# execution controls. Nothing of the original model's intent is copied.
# Usage: ra_extract_project.sh SOURCE_REPO SRC TARGET
set -euo pipefail
SOURCE="$1"
SRC="$2"
TARGET="$3"
DEMO="$(cd "$(dirname "$0")" && pwd)"
test ! -e "$TARGET"
mkdir -p "$TARGET/.markitect"
( cd "$SOURCE" && git ls-files -co --exclude-standard realization | tar -cf - -T - ) | tar -xf - -C "$TARGET"
cp -r "$SRC/experiments/test-projects/library-r2/project/.markitect/dependencies" "$TARGET/.markitect/dependencies"
M1="$SRC/docs/design/canonical-model/library-example/m1"
cp "$M1/model-lock.yaml" "$TARGET/.markitect/model-lock.yaml"
cat > "$TARGET/.markitect/model.yaml" <<'YAML'
formatVersion: 1
id: extracted-project
root: model
dependencies:
  markitect-core: "1.0.0"
  software-basics: "0.1.0"
YAML
install -m 0644 "$DEMO/extract-execution.yaml" "$TARGET/.markitect/execution.yaml"
# Git tracks no empty directory; a placeholder keeps the empty model root.
mkdir -p "$TARGET/.markitect/model" && : > "$TARGET/.markitect/model/.gitkeep"
cd "$TARGET"
git init --quiet --initial-branch=main
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" add .
git -c user.name="Markitect demo" -c user.email="demo@markitect.invalid" commit --quiet -m "Existing library realization without a model (case 3)"
git rev-parse HEAD
