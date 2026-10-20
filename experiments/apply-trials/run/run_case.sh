#!/usr/bin/env bash
# One trial run end to end: bind, materialize, Apply, cleanup, assess (the assessment runs inside the
# Python image, so Agent-written code never runs on the host).
# Usage: run_case.sh BUILD CASE ATTEMPT [FROM_REPO] [PREVIOUS_REALIZATION]
# Env: TRIALS_SRC (an experiments directory to freeze into the attempt; default the build's source),
#      LANES (concurrency note), MARKITECT_AGENT_MODEL (default claude-haiku-5-5),
#      MARKITECT_RUNTIME_ENV (the file holding CLAUDE_CODE_OAUTH_TOKEN), TRIALS_WORK (default ~/mk-tr).
set -uo pipefail
BUILD="$1"; CASE="$2"; ATTEMPT="$3"; FROM="${4:-}"; PREV="${5:-}"
WORK="${TRIALS_WORK:-$HOME/mk-tr}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SHA=$(grep '^sha=' "$BUILD/build.txt" | cut -d= -f2)
SRC="$WORK/src-$SHA"
EXPERIMENTS="${TRIALS_SRC:-$SRC/experiments}"
export MARKITECT_DEMO_WORK="$WORK"
export MARKITECT_AGENT_MODEL="${MARKITECT_AGENT_MODEL:-claude-haiku-5-5}"
export MARKITECT_STORE="tr-$(basename "$ATTEMPT" | tr 'A-Z' 'a-z')"
PYIMG=$(grep '^python_image=' "$BUILD/build.txt" | cut -d= -f2)
test ! -e "$ATTEMPT" || { echo "$ATTEMPT exists"; exit 2; }
bash "$SRC/experiments/recursive-apply-demo/ra_bind.sh" "$BUILD" "$ATTEMPT" > /dev/null || exit 2
mkdir -p "$ATTEMPT/out" "$ATTEMPT/exp"
cp -r "$EXPERIMENTS/apply-trials" "$EXPERIMENTS/test-projects" "$ATTEMPT/exp/"
find "$ATTEMPT/exp" -name __pycache__ -prune -exec rm -rf {} +
{
  echo "trials_source=$EXPERIMENTS"
  echo "trials_digest=$(cd "$ATTEMPT/exp" && find . -type f | sort | xargs sha256sum | sha256sum | cut -d' ' -f1)"
  echo "case=$CASE from=$FROM previous=$PREV lanes=${LANES:-1} agent_model=$MARKITECT_AGENT_MODEL"
} >> "$ATTEMPT/binding.txt"
T="$ATTEMPT/exp/apply-trials"
python3 "$T/bind_profiles.py" "$ATTEMPT" "$PYIMG" > /dev/null || exit 3
python3 -B "$T/materialize.py" "$CASE" "$ATTEMPT/repo" ${FROM:+--from-repo "$FROM"} > "$ATTEMPT/out/materialize.json" \
  2> "$ATTEMPT/out/materialize.stderr" || exit 4
MIN=$(sed -n 's/^  executionTime: *\([0-9]*\)m$/\1/p' "$ATTEMPT/repo/.markitect/execution.yaml")
DEADLINE=$(date -u -d "+$(( MIN + 2 )) minutes" +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" run "$SHA" "$DEADLINE" "Apply the committed model." > /dev/null || exit 5
python3 -B "$HERE/patch_runconfig.py" "$ATTEMPT/run-config-run.json" "trial-$(basename "$ATTEMPT")" || exit 5
start=$(date +%s)
echo "started=$(date -u +%FT%TZ)" > "$ATTEMPT/timing.txt"
timeout --kill-after=30 $(( (MIN + 6) * 60 )) "$ATTEMPT/markitect" apply --repo "$ATTEMPT/repo" --model .markitect/model.yaml \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --host-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-run.json" \
  > "$ATTEMPT/out/apply.json" 2> "$ATTEMPT/out/apply.stderr"
code=$?
echo "apply_exit=$code wall_seconds=$(( $(date +%s) - start )) ended=$(date -u +%FT%TZ)" >> "$ATTEMPT/timing.txt"
bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
docker run --rm --entrypoint "" --network none -u 1000:1000 -e HOME=/tmp -v "$WORK:$WORK" -w "$T" "$PYIMG" \
  python3 -B assess.py "$CASE" --repo "$ATTEMPT/repo" --report "$ATTEMPT/out/apply.json" ${PREV:+--previous "$PREV"} \
  --output "$ATTEMPT/out/assess.json" > /dev/null 2> "$ATTEMPT/out/assess.stderr"
echo "assess_exit=$?" >> "$ATTEMPT/timing.txt"
cat "$ATTEMPT/timing.txt"
