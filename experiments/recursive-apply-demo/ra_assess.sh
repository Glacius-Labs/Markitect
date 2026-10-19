#!/usr/bin/env bash
# Independent oracle assessment of the published A1 and A2, after the timed run.
# Usage: ra_assess.sh ATTEMPT_DIR
set -uo pipefail
ATTEMPT="$1"
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
SRC="${MARKITECT_DEMO_WORK:-$HOME/mk-ra}/src-$SHA"
TAG="${SHA:0:12}"
GOIMG="markitect-runtime-go:ra-$TAG"
A1=$(grep '^a1_commit=' "$ATTEMPT/timing.txt" | cut -d= -f2)
rm -rf "$ATTEMPT/a1-tree" && mkdir -p "$ATTEMPT/a1-tree"
git -C "$ATTEMPT/repo" archive "$A1" | tar -x -C "$ATTEMPT/a1-tree"
for pair in "a1 m1 $ATTEMPT/a1-tree" "a2 m2 $ATTEMPT/repo" "a2-as-m1 m1 $ATTEMPT/repo"; do
  set -- $pair
  mkdir -p "$ATTEMPT/assess-$1"
  python3 -I "$SRC/experiments/runtime-baseline/assess.py" --realization "$3" --profile "$2" --out "$ATTEMPT/assess-$1" --label "$1" \
    --repo "$SRC" --rev "$SHA" --engine /usr/bin/docker --go-image "$GOIMG" > "$ATTEMPT/assess-$1/summary.json" 2> "$ATTEMPT/assess-$1/stderr.txt"
  echo "assess $1 profile=$2 exit=$?"
  python3 -c "import json,sys; d=json.load(open(sys.argv[1])); print('  passed', d.get('passed'), 'cases', d.get('cases_passed'), '/', d.get('cases'), 'discrimination', d.get('discrimination'), 'errors', d.get('errors'))" "$ATTEMPT/assess-$1/summary.json"
done
