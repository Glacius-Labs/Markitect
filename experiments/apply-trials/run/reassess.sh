#!/usr/bin/env bash
# Re-assess finished attempts with one trial setup, so all results share the same checks.
# Usage: reassess.sh BUILD EXPERIMENTS_DIR [ATTEMPT_PREFIX]   (writes out/assess-final.json per attempt)
set -u
BUILD="$1"; EXP="$2/apply-trials"
WORK="${TRIALS_WORK:-$HOME/mk-tr}"
PYIMG=$(grep '^python_image=' "$BUILD/build.txt" | cut -d= -f2)
for A in "$WORK"/${3:-T}*/; do
  A=${A%/}
  [ -f "$A/out/apply.json" ] && [ -f "$A/timing.txt" ] && grep -q '^assess_exit=' "$A/timing.txt" || continue
  CASE=$(sed -n 's/^case=\([^ ]*\) .*/\1/p' "$A/binding.txt")
  PREV=$(sed -n 's/.* previous=\([^ ]*\) .*/\1/p' "$A/binding.txt")
  docker run --rm --entrypoint "" --network none -u 1000:1000 -e HOME=/tmp -v "$WORK:$WORK" -v "$EXP:$EXP:ro" -w "$EXP" "$PYIMG" \
    python3 -B assess.py "$CASE" --repo "$A/repo" --report "$A/out/apply.json" ${PREV:+--previous "$PREV"} \
    --output "$A/out/assess-final.json" > /dev/null 2> "$A/out/assess-final.stderr"
  echo "$(basename "$A") $CASE exit=$?"
done
