#!/usr/bin/env bash
# One Model assessment attempt end to end: verify the frozen inputs, check the
# series ceilings and that no other provider series uses this Docker host, bind
# (Sonnet, effort high), clone the frozen packet, run the grant-gated harness
# with an operator deadline in a closed Run (no owner channel), clean up the
# attempt's containers, classify and score the attempt outside the Agents'
# context and record it in the series ledger.
#
# The ledger line is written before anything starts, charged with the whole cap
# and marked running; the end of the attempt replaces it with the actual
# charge. An attempt that is killed keeps its pessimistic line, and a running
# line refuses every further attempt until the operator marks it abandoned, so
# no attempt escapes the ceilings. Name attempts neutrally (C-01, ...): the
# control stays in the ledger, never in a name an Agent could see.
# Usage: attempt.sh FROZEN CONTROL ATTEMPT_NAME
# Env: MARKITECT_RUNTIME_ENV, the dedicated Claude runtime token file (never printed).
# ASSESS_SMOKE_NO_CREDENTIAL=1 removes the bound token before the run: every
# turn then fails at its credential read, so the wiring runs without a provider.
# Use it only on a smoke freeze, whose ledger is not the series'.
set -uo pipefail
FROZEN="$(realpath "$1")"; CONTROL="$2"; NAME="$3"
KIT="$FROZEN/kit"
LEDGER="$FROZEN/ledger.txt"
ATTEMPT="$FROZEN/attempts/$NAME"
PACKET="$FROZEN/packets/$CONTROL"
PASS="$KIT/model-assessment/assessment-pass.json"
GO_SENTENCE=" Use only the Go standard library for the realization."
python3 -B "$KIT/model-assessment/run/freeze.py" verify "$FROZEN" "$CONTROL" || exit 2
BUILD=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["build"])' "$FROZEN/manifest.json")
HEAD=$(git -C "$PACKET" rev-parse HEAD)
test ! -e "$ATTEMPT" || { echo "$ATTEMPT exists"; exit 2; }

# Ceilings (at most 12 attempts, 90 summed minutes, 10 minutes each) and the
# busy check run under the ledger lock together with the write-ahead line.
exec 9>> "$FROZEN/ledger.lock"
flock 9
read -r COUNT SUMMED RUNNING < <(awk '{n++; if ($0 ~ / status=running /) r++; for (i = 1; i <= NF; i++) if ($i ~ /^charged=/) {split($i, a, "="); s += a[2]}}
  END {printf "%d %d %d\n", n, s, r}' "$LEDGER")
REMAINING=$(( 5400 - SUMMED ))
CAP=$(( REMAINING < 600 ? REMAINING : 600 ))
if [ "$RUNNING" -gt 0 ]; then echo "an attempt is still running or was killed: mark its ledger line abandoned first"; exit 3; fi
if [ "$COUNT" -ge 12 ] || [ "$CAP" -lt 180 ]; then echo "ceiling reached: $COUNT attempts, $SUMMED s"; exit 3; fi
# One provider series per Docker host; never touch another session's containers.
BUSY=$(docker ps --filter label=markitect.store --format '{{.Label "markitect.store"}} {{.Names}}') || { echo "docker is unavailable"; exit 4; }
if [ -n "$BUSY" ]; then echo "the Docker host is busy:"; echo "$BUSY"; exit 4; fi
START=$(date +%s)
STARTED=$(date -u -d "@$START" +%FT%TZ)
echo "$CONTROL $NAME status=running charged=$CAP cap=$CAP started=$STARTED" >> "$LEDGER"
flock -u 9

RUN=""; CONCLUSION="not-started"; MET="none"; CLEANED=0
STATE="$ATTEMPT/state-run"
export MARKITECT_DEMO_WORK="$FROZEN" MARKITECT_AGENT_MODEL=claude-sonnet-5-5 MARKITECT_STORE="ma-$(echo "$NAME" | tr 'A-Z' 'a-z')"
export MARKITECT_RUNTIME_ENV="${MARKITECT_RUNTIME_ENV:-/mnt/c/Users/Consiliari/.markitect/runtime.env}"
cleanup() {
  [ "$CLEANED" = 1 ] && return
  CLEANED=1
  mkdir -p "$ATTEMPT"
  [ -e "$ATTEMPT/host/host.yaml" ] && bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
  echo "cleanup_done=$(date -u +%FT%TZ) alive_after=$(docker ps -q --filter "label=markitect.store=$MARKITECT_STORE" | grep -c . || true) wall_seconds=$(( $(date +%s) - START ))" >> "$ATTEMPT/timing.txt"
}
finish() {
  cleanup
  local line="$CONTROL $NAME status=done run=${RUN:-none} conclusion=$CONCLUSION met=$MET charged=$(( $(date +%s) - START )) cap=$CAP started=$STARTED"
  flock "$FROZEN/ledger.lock" python3 -c 'import sys
path, name, line = sys.argv[1:]
lines = open(path).read().splitlines()
hits = [i for i, l in enumerate(lines) if l.split()[1:2] == [name] and " status=running " in l]
lines[hits[-1]] = line
open(path + ".new", "w").write("\n".join(lines) + "\n")
import os; os.replace(path + ".new", path)' "$LEDGER" "$NAME" "$line"
}
trap finish EXIT
trap 'exit 130' INT TERM

mkdir -p "$FROZEN/attempts"
bash "$KIT/recursive-apply-demo/ra_bind.sh" "$BUILD" "$ATTEMPT" > /dev/null || { CONCLUSION=bind-failed; exit 5; }
if [ "${ASSESS_SMOKE_NO_CREDENTIAL:-}" = 1 ]; then
  rm -f "$ATTEMPT/secret/claude-token"
  echo "smoke=no credential: provider-free wiring smoke" >> "$ATTEMPT/binding.txt"
fi
mkdir -p "$ATTEMPT/out"
git clone --quiet --no-hardlinks "$PACKET" "$ATTEMPT/repo" && git -C "$ATTEMPT/repo" remote remove origin || { CONCLUSION=packet-failed; exit 5; }
if grep -q 'execute: project-agent' "$ATTEMPT/repo/.markitect/execution.yaml"; then
  python3 "$KIT/apply-trials/bind_profiles.py" "$ATTEMPT" "$(grep '^python_image=' "$BUILD/build.txt" | cut -d= -f2)" > /dev/null || { CONCLUSION=bind-failed; exit 5; }
fi
# The demonstration profile names a technology the model may not state; an assessment gets no such hint.
python3 -c 'import sys
path, sentence = sys.argv[1:]
text = open(path).read()
open(path, "w").write(text.replace(sentence, ""))' "$ATTEMPT/host/host.yaml" "$GO_SENTENCE" || { CONCLUSION=bind-failed; exit 5; }
SHA=$(grep '^sha=' "$BUILD/build.txt" | cut -d= -f2)
DEADLINE=$(date -u -d "@$(( START + CAP - 90 ))" +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" run "$SHA" "$DEADLINE" "Assess the committed model." > /dev/null &&
  python3 -B "$KIT/apply-trials/run/patch_runconfig.py" "$ATTEMPT/run-config-run.json" "project-$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')" ||
  { CONCLUSION=config-failed; exit 5; }
{
  echo "control=$CONTROL packet_head=$HEAD packet_tree=$(git -C "$PACKET" rev-parse 'HEAD^{tree}')"
  echo "assessment_pass_sha256=$(sha256sum "$PASS" | cut -d' ' -f1) harness_sha256=$(sha256sum "$BUILD/assessment.test" | cut -d' ' -f1)"
  echo "closed_run=no owner channel, no owner decisions cap=$CAP deadline=$DEADLINE store=$MARKITECT_STORE"
  echo "host_profile_sha256_assessment=$(sha256sum "$ATTEMPT/host/host.yaml" | cut -d' ' -f1) (Go sentence removed)"
} >> "$ATTEMPT/binding.txt"

ARGS=(apply --repo "$ATTEMPT/repo" --model .markitect/model.yaml)
for dep in "$ATTEMPT"/repo/.markitect/dependencies/*/model.yaml; do
  [ -e "$dep" ] || continue
  ARGS+=(--dependency "${dep#"$ATTEMPT/repo/"}")
done
ARGS+=(--runtime-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-run.json")
LIMIT=$(( START + CAP - 45 - $(date +%s) ))
[ "$LIMIT" -ge 60 ] || { CONCLUSION=setup-too-slow; exit 5; }
echo "started=$STARTED deadline=$DEADLINE harness_limit=$LIMIT" > "$ATTEMPT/timing.txt"
MARKITECT_MODEL_ASSESSMENT=run-model-assessment MARKITECT_MODEL_ASSESSMENT_PASS="$PASS" \
  MARKITECT_MODEL_ASSESSMENT_ARGS="$(python3 -c 'import json, sys; print(json.dumps(sys.argv[1:]))' "${ARGS[@]}")" \
  timeout --kill-after=15 "$LIMIT" "$BUILD/assessment.test" -test.run '^TestModelAssessmentHarness$' -test.count=1 -test.timeout=$(( LIMIT - 5 ))s \
  > "$ATTEMPT/out/harness.stdout" 2> "$ATTEMPT/out/harness.stderr"
echo "exit=$?" >> "$ATTEMPT/timing.txt"
cleanup
RUN=$(cat "$STATE/current-run" 2>/dev/null)
EVIDENCE="$STATE/runs/$RUN"
REPORT="$EVIDENCE/report.json"
[ -n "$RUN" ] && [ -s "$REPORT" ] || REPORT="$ATTEMPT/out/harness.stdout"
python3 -B "$KIT/model-assessment/assess.py" attempt --report "$REPORT" --evidence "$EVIDENCE" \
  --control "$FROZEN/controls/$CONTROL.json" --timing "$ATTEMPT/timing.txt" --output "$ATTEMPT/out/assessment.json" > /dev/null 2> "$ATTEMPT/out/assess.stderr"
CONCLUSION=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["classification"]["conclusion"])' "$ATTEMPT/out/assessment.json" 2>/dev/null || echo unassessed)
MET=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1])).get("expectationMet"))' "$ATTEMPT/out/assessment.json" 2>/dev/null || echo none)
cat "$ATTEMPT/timing.txt"
