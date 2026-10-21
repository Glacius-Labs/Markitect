#!/usr/bin/env bash
# One Model assessment attempt end to end: verify the frozen inputs, check the
# series ceilings and that no other provider series uses this Docker host, bind
# (Sonnet, effort high), clone the frozen packet, run the grant-gated harness
# with an operator deadline, answer every question of Root with one fixed
# deferral, clean up the attempt's containers, classify and score the attempt
# outside the Agents' context and append it to the series ledger. Every started
# attempt is recorded, a failed start too. Name attempts neutrally (C-01, ...):
# the control stays in the ledger, never in a name an Agent could see.
# Usage: attempt.sh FROZEN CONTROL ATTEMPT_NAME
# Env: MARKITECT_RUNTIME_ENV, the dedicated Claude runtime token file (never printed).
set -uo pipefail
FROZEN="$1"; CONTROL="$2"; NAME="$3"
KIT="$FROZEN/kit"
LEDGER="$FROZEN/ledger.txt"
ATTEMPT="$FROZEN/attempts/$NAME"
PACKET="$FROZEN/packets/$CONTROL"
PASS="$KIT/model-assessment/assessment-pass.json"
DEFERRAL="No owner decision is available during this assessment. Do not assume an answer. Conclude your assessment within the bound model and keep a genuine finding for each decision it leaves open."
field() {
  python3 -c 'import json, sys
value = json.load(open(sys.argv[1]))
for key in sys.argv[2:]:
    value = value[key]
print(value)' "$FROZEN/manifest.json" "$@"
}
BUILD=$(field build)
HEAD=$(field controls "$CONTROL" head) || { echo "unknown control $CONTROL"; exit 2; }
test "$(sha256sum "$PASS" | cut -d' ' -f1)" = "$(field assessmentPass)" || { echo "the assessment Pass is not the frozen one"; exit 2; }
test "$(sha256sum "$BUILD/assessment.test" | cut -d' ' -f1)" = "$(field harness)" || { echo "the harness is not the frozen one"; exit 2; }
test "$(git -C "$PACKET" rev-parse HEAD)" = "$HEAD" && test -z "$(git -C "$PACKET" status --porcelain)" || { echo "the packet is not the frozen one"; exit 2; }
test ! -e "$ATTEMPT" || { echo "$ATTEMPT exists"; exit 2; }

# Ceilings: at most 12 attempts, 90 summed minutes, 10 minutes per attempt.
read -r COUNT SUMMED < <(awk '{n++; for (i = 1; i <= NF; i++) if ($i ~ /^charged=/) {split($i, a, "="); s += a[2]}} END {printf "%d %d\n", n, s}' "$LEDGER")
REMAINING=$(( 5400 - SUMMED ))
CAP=$(( REMAINING < 600 ? REMAINING : 600 ))
if [ "$COUNT" -ge 12 ] || [ "$CAP" -lt 180 ]; then echo "ceiling reached: $COUNT attempts, $SUMMED s"; exit 3; fi
# One provider series per Docker host; never touch another session's containers.
BUSY=$(docker ps --filter label=markitect.store --format '{{.Label "markitect.store"}} {{.Names}}') || { echo "docker is unavailable"; exit 4; }
if [ -n "$BUSY" ]; then echo "the Docker host is busy:"; echo "$BUSY"; exit 4; fi

START=$(date +%s)
RUN=""; CONCLUSION="not-started"; MET="none"
record() {
  echo "$CONTROL $NAME run=${RUN:-none} conclusion=$CONCLUSION met=$MET charged=$(( $(date +%s) - START )) cap=$CAP started=$(date -u -d "@$START" +%FT%TZ)" >> "$LEDGER"
}
trap record EXIT
mkdir -p "$FROZEN/attempts"
export MARKITECT_DEMO_WORK="$FROZEN" MARKITECT_AGENT_MODEL=claude-sonnet-5-5 MARKITECT_STORE="ma-$(echo "$NAME" | tr 'A-Z' 'a-z')"
export MARKITECT_RUNTIME_ENV="${MARKITECT_RUNTIME_ENV:-/mnt/c/Users/Consiliari/.markitect/runtime.env}"
bash "$KIT/recursive-apply-demo/ra_bind.sh" "$BUILD" "$ATTEMPT" > /dev/null || { CONCLUSION=bind-failed; exit 5; }
mkdir -p "$ATTEMPT/out"
git clone --quiet --no-hardlinks "$PACKET" "$ATTEMPT/repo" && git -C "$ATTEMPT/repo" remote remove origin || { CONCLUSION=packet-failed; exit 5; }
if grep -q 'execute: project-agent' "$ATTEMPT/repo/.markitect/execution.yaml"; then
  python3 "$KIT/apply-trials/bind_profiles.py" "$ATTEMPT" "$(grep '^python_image=' "$BUILD/build.txt" | cut -d= -f2)" > /dev/null || { CONCLUSION=bind-failed; exit 5; }
fi
SHA=$(grep '^sha=' "$BUILD/build.txt" | cut -d= -f2)
DEADLINE=$(date -u -d "@$(( START + CAP - 90 ))" +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" run "$SHA" "$DEADLINE" "Assess the committed model." > /dev/null &&
  python3 -B "$KIT/apply-trials/run/patch_runconfig.py" "$ATTEMPT/run-config-run.json" "project-$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')" ||
  { CONCLUSION=config-failed; exit 5; }
{
  echo "control=$CONTROL packet_head=$HEAD packet_tree=$(git -C "$PACKET" rev-parse 'HEAD^{tree}')"
  echo "assessment_pass_sha256=$(sha256sum "$PASS" | cut -d' ' -f1) harness_sha256=$(sha256sum "$BUILD/assessment.test" | cut -d' ' -f1)"
  echo "deferral_sha256=$(printf '%s' "$DEFERRAL" | sha256sum | cut -d' ' -f1) cap=$CAP deadline=$DEADLINE store=$MARKITECT_STORE"
} >> "$ATTEMPT/binding.txt"

STATE="$ATTEMPT/state-run"
STOP="$ATTEMPT/owner.stop"
(
  while [ ! -e "$STOP" ]; do
    for q in "$STATE"/questions/*.json; do
      [ -e "$q" ] || continue
      id=$(basename "$q" .json)
      grep -q "^$id " "$ATTEMPT/out/owner.log" 2>/dev/null && continue
      cp "$q" "$ATTEMPT/out/question-$id.json"
      "$ATTEMPT/markitect" answer "$id" --run "$(cat "$STATE/current-run")" --text "$DEFERRAL" --run-config "$ATTEMPT/run-config-run.json" > "$ATTEMPT/out/answer-$id.json" 2>&1
      echo "$id deferral exit=$? at=$(date -u +%FT%TZ)" >> "$ATTEMPT/out/owner.log"
    done
    sleep 3
  done
) &
OWNER=$!
ARGS=(apply --repo "$ATTEMPT/repo" --model .markitect/model.yaml)
for dep in "$ATTEMPT"/repo/.markitect/dependencies/*/model.yaml; do
  ARGS+=(--dependency "${dep#"$ATTEMPT/repo/"}")
done
ARGS+=(--runtime-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-run.json")
echo "started=$(date -u -d "@$START" +%FT%TZ) deadline=$DEADLINE" > "$ATTEMPT/timing.txt"
MARKITECT_MODEL_ASSESSMENT=run-model-assessment MARKITECT_MODEL_ASSESSMENT_PASS="$PASS" \
  MARKITECT_MODEL_ASSESSMENT_ARGS="$(python3 -c 'import json, sys; print(json.dumps(sys.argv[1:]))' "${ARGS[@]}")" \
  timeout --kill-after=15 $(( CAP - 45 )) "$BUILD/assessment.test" -test.run '^TestModelAssessmentHarness$' -test.count=1 -test.timeout=$(( CAP - 50 ))s \
  > "$ATTEMPT/out/harness.stdout" 2> "$ATTEMPT/out/harness.stderr"
echo "exit=$?" >> "$ATTEMPT/timing.txt"
touch "$STOP"
wait "$OWNER" 2>/dev/null
bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
echo "cleanup_done=$(date -u +%FT%TZ) alive_after=$(docker ps -q --filter "label=markitect.store=$MARKITECT_STORE" | grep -c . || true) wall_seconds=$(( $(date +%s) - START ))" >> "$ATTEMPT/timing.txt"
RUN=$(cat "$STATE/current-run" 2>/dev/null)
EVIDENCE="$STATE/runs/$RUN"
REPORT="$EVIDENCE/report.json"
[ -n "$RUN" ] && [ -s "$REPORT" ] || REPORT="$ATTEMPT/out/harness.stdout"
python3 -B "$KIT/model-assessment/assess.py" attempt --report "$REPORT" --events "$EVIDENCE/events.jsonl" \
  --control "$FROZEN/controls/$CONTROL.json" --timing "$ATTEMPT/timing.txt" --output "$ATTEMPT/out/assessment.json" > /dev/null 2> "$ATTEMPT/out/assess.stderr"
CONCLUSION=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["classification"]["conclusion"])' "$ATTEMPT/out/assessment.json" 2>/dev/null || echo unassessed)
MET=$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1])).get("expectationMet"))' "$ATTEMPT/out/assessment.json" 2>/dev/null || echo none)
cat "$ATTEMPT/timing.txt"
