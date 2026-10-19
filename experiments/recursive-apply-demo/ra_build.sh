#!/usr/bin/env bash
# Build the product images, the oracle check image and the CLI from one exact
# commit. This is product build work, outside the timed demonstration.
# Usage: ra_build.sh SHA OUTDIR
set -euo pipefail
SHA="$1"
OUT="$2"
DEMO="$(cd "$(dirname "$0")" && pwd)"
# Go and Docker must be on PATH. MARKITECT_SOURCE_REPO is the repository to build
# from (default: the one containing this script).
SOURCE="${MARKITECT_SOURCE_REPO:-$(git -C "$DEMO" rev-parse --show-toplevel)}"
SRC="${MARKITECT_DEMO_WORK:-$HOME/mk-ra}/src-$SHA"
mkdir -p "$OUT"
start=$(date +%s)
if [ ! -d "$SRC" ]; then
  git clone --quiet "$SOURCE" "$SRC"
fi
cd "$SRC"
git fetch --quiet "$SOURCE" "+refs/heads/*:refs/remotes/local/*" "+refs/remotes/origin/*:refs/remotes/upstream/*"
git checkout --quiet --detach "$SHA"
test "$(git rev-parse HEAD)" = "$SHA"
test -z "$(git status --porcelain)"
TAG="${SHA:0:12}"
docker build --quiet -f packaging/runtime/Dockerfile -t "markitect-runtime:ra-$TAG" . > "$OUT/build-runtime.log" 2>&1
docker build --quiet -f packaging/runtime/examples/go/Dockerfile --build-arg "BASE=markitect-runtime:ra-$TAG" -t "markitect-runtime-go:ra-$TAG" . > "$OUT/build-runtime-go.log" 2>&1
rm -rf "$OUT/oracle-context"
mkdir -p "$OUT/oracle-context/evaluator/expectations"
cp experiments/library-playground/assess.py experiments/library-playground/driver.py experiments/library-playground/oracle.py "$OUT/oracle-context/evaluator/"
cp experiments/library-playground/expectations/*.json "$OUT/oracle-context/evaluator/expectations/"
cp "$DEMO/oracle/check.py" "$OUT/oracle-context/check.py"
docker build --quiet -f "$DEMO/oracle/Dockerfile" --build-arg "BASE=markitect-runtime-go:ra-$TAG" -t "markitect-oracle:ra-$TAG" "$OUT/oracle-context" > "$OUT/build-oracle.log" 2>&1
go build -o "$OUT/markitect" ./cmd/markitect
{
  echo "sha=$SHA"
  echo "go=$(go version)"
  echo "docker=$(docker version --format '{{.Server.Version}}')"
  for img in "markitect-runtime:ra-$TAG" "markitect-runtime-go:ra-$TAG" "markitect-oracle:ra-$TAG"; do
    echo "image $img $(docker image inspect --format '{{.Id}}' "$img")"
  done
  cid=$(docker create "markitect-runtime-go:ra-$TAG")
  docker cp "$cid:/opt/markitect/runtime.env" - | tar -xO > "$OUT/runtime.env"
  docker rm "$cid" > /dev/null
  sed 's/^/runtime.env /' "$OUT/runtime.env"
  echo "cli_sha256=$(sha256sum "$OUT/markitect" | cut -d' ' -f1)"
  echo "oracle_check_sha256=$(sha256sum "$DEMO/oracle/check.py" | cut -d' ' -f1)"
  echo "build_seconds=$(( $(date +%s) - start ))"
} > "$OUT/build.txt"
cat "$OUT/build.txt"
