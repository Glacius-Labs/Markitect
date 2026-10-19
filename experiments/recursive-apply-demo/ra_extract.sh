#!/usr/bin/env bash
# Extract on the library: project a case-3 target from a published A2, run
# `markitect extract` under a hard wall-clock limit outside every prompt, then,
# outside the limit, adopt the published candidate by a scripted commit and run
# `markitect apply` with the extracted model on the unchanged artifacts as a
# round-trip check (expected: no change).
# Usage: ra_extract.sh ATTEMPT_DIR SRC_DIR SOURCE_REPO
set -uo pipefail
ATTEMPT="$1"
SRC="$2"
SOURCE="$3"
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
REPO="$ATTEMPT/repo"
mkdir -p "$ATTEMPT/out"
mark() { echo "$1=$(date +%s)" >> "$ATTEMPT/timing.txt"; }
T0=$(date +%s)
echo "start_utc=$(date -u -d @$T0 +%FT%TZ) epoch=$T0" > "$ATTEMPT/timing.txt"
bash "$ATTEMPT/frozen/ra_extract_project.sh" "$SOURCE" "$SRC" "$REPO" > "$ATTEMPT/out/baseline.txt" 2>&1 || exit 3
DEADLINE=$(date -u -d @$((T0 + 1560)) +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" x1 "$SHA" "$DEADLINE" "Extract the model of the existing library artifacts." > /dev/null
mark extract_start
timeout --kill-after=45 1650 "$ATTEMPT/markitect" extract --repo "$REPO" \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --host-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-x1.json" \
  > "$ATTEMPT/out/extract-x1.json" 2> "$ATTEMPT/out/extract-x1.stderr"
echo "extract_exit=$?" >> "$ATTEMPT/timing.txt"
mark extract_end
timeout 60 bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
echo "extract_total_seconds=$(( $(date +%s) - T0 ))" >> "$ATTEMPT/timing.txt"
# Outside the limit: scripted adoption of the published candidate, then the round trip.
cd "$REPO"
if [ -z "$(git status --porcelain)" ]; then echo "outcome=NOTHING_PUBLISHED" >> "$ATTEMPT/timing.txt"; exit 4; fi
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" add -A
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" commit --quiet -m "Adopt the extracted model candidate (scripted for evaluation)"
echo "adopted_commit=$(git rev-parse HEAD)" >> "$ATTEMPT/timing.txt"
cd - > /dev/null
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" r1 "$SHA" "$(date -u -d '+21 minutes' +%FT%TZ)" "Apply the committed Library model." > /dev/null
mark roundtrip_start
timeout --kill-after=45 1500 "$ATTEMPT/markitect" apply --repo "$REPO" --model .markitect/model.yaml \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --host-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-r1.json" \
  > "$ATTEMPT/out/apply-r1.json" 2> "$ATTEMPT/out/apply-r1.stderr"
echo "roundtrip_exit=$?" >> "$ATTEMPT/timing.txt"
mark roundtrip_end
timeout 60 bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
(cd "$REPO" && echo "roundtrip_changed_files=$(git status --porcelain | wc -l)") >> "$ATTEMPT/timing.txt"
cat "$ATTEMPT/timing.txt"
