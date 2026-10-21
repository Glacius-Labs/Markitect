#!/usr/bin/env bash
# Extract on the library: project a case-3 target from a library realization,
# run `markitect extract` under a hard wall-clock limit outside every prompt,
# with the scripted owner answering questions, then, outside the limit, adopt
# the published candidate by a scripted commit. With REBUILD=rebuild the
# adopted model is applied to an empty realization (case 1) under its own limit
# and the independent oracle judges the rebuilt program.
# Usage: ra_extract.sh ATTEMPT_DIR SRC_DIR SOURCE_REPO [rebuild|norebuild]
set -uo pipefail
ATTEMPT="$1"
SRC="$2"
SOURCE="$3"
REBUILD="${4:-rebuild}"
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
REPO="$ATTEMPT/repo"
mkdir -p "$ATTEMPT/out"
mark() { echo "$1=$(date +%s)" >> "$ATTEMPT/timing.txt"; }
T0=$(date +%s)
echo "start_utc=$(date -u -d @$T0 +%FT%TZ) epoch=$T0" > "$ATTEMPT/timing.txt"
bash "$ATTEMPT/frozen/ra_extract_project.sh" "$SOURCE" "$SRC" "$REPO" > "$ATTEMPT/out/baseline.txt" 2>&1 || exit 3
DEADLINE=$(date -u -d @$((T0 + 1560)) +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" x1 "$SHA" "$DEADLINE" "Extract the model of the existing library artifacts." > /dev/null
mkdir -p "$ATTEMPT/state-x1"
mark extract_start
timeout --kill-after=45 1650 "$ATTEMPT/markitect" extract --repo "$REPO" \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --runtime-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-x1.json" \
  > "$ATTEMPT/out/extract-x1.json" 2> "$ATTEMPT/out/extract-x1.stderr"
echo "extract_exit=$?" >> "$ATTEMPT/timing.txt"
mark extract_end
timeout 60 bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
echo "extract_total_seconds=$(( $(date +%s) - T0 ))" >> "$ATTEMPT/timing.txt"
# Outside the limit: scripted adoption of the published candidate.
cd "$REPO"
if [ -z "$(git status --porcelain)" ]; then echo "outcome=NOTHING_PUBLISHED" >> "$ATTEMPT/timing.txt"; exit 4; fi
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" add -A
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" commit --quiet -m "Adopt the extracted model candidate (scripted for evaluation)"
echo "adopted_commit=$(git rev-parse HEAD)" >> "$ATTEMPT/timing.txt"
cd - > /dev/null
[ "$REBUILD" = rebuild ] || { cat "$ATTEMPT/timing.txt"; exit 0; }
# Rebuild: the adopted model alone, applied to an empty realization.
bash "$ATTEMPT/frozen/rx_rebuild_project.sh" "$REPO" "$ATTEMPT/rebuild" > "$ATTEMPT/out/rebuild-baseline.txt" 2>&1 || { echo "outcome=REBUILD_PROJECT_FAILED" >> "$ATTEMPT/timing.txt"; exit 5; }
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" b1 "$SHA" "$(date -u -d '+26 minutes' +%FT%TZ)" "Apply the committed model." > /dev/null
mark rebuild_start
timeout --kill-after=45 1650 "$ATTEMPT/markitect" apply --repo "$ATTEMPT/rebuild" --model .markitect/model.yaml \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --runtime-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-b1.json" \
  > "$ATTEMPT/out/apply-b1.json" 2> "$ATTEMPT/out/apply-b1.stderr"
echo "rebuild_exit=$?" >> "$ATTEMPT/timing.txt"
mark rebuild_end
timeout 60 bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" > /dev/null 2>&1
# Outside every limit: the independent oracle on the rebuilt program (M2 policy: 3 loans, 14 days).
TAG="${SHA:0:12}"
mkdir -p "$ATTEMPT/assess-rebuild"
if [ -f "$ATTEMPT/rebuild/realization/go.mod" ]; then
  python3 -I "$SRC/experiments/runtime-baseline/assess.py" --realization "$ATTEMPT/rebuild" --profile m2 --out "$ATTEMPT/assess-rebuild" --label rebuild \
    --repo "$SRC" --rev "$SHA" --engine /usr/bin/docker --go-image "markitect-runtime-go:ra-$TAG" > "$ATTEMPT/assess-rebuild/summary.json" 2> "$ATTEMPT/assess-rebuild/stderr.txt"
  echo "oracle_exit=$?" >> "$ATTEMPT/timing.txt"
else
  echo "oracle=NO_REALIZATION" >> "$ATTEMPT/timing.txt"
fi
cat "$ATTEMPT/timing.txt"
