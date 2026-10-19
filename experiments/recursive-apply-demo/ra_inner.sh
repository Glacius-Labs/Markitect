#!/usr/bin/env bash
# Inner sequence of the timed demonstration; ra_demo.sh bounds it.
# Usage: ra_inner.sh ATTEMPT SRC SHA DEADLINE
set -uo pipefail
ATTEMPT="$1"
SRC="$2"
SHA="$3"
DEADLINE="$4"
REPO="$ATTEMPT/repo"
mark() { echo "$1=$(date +%s)" >> "$ATTEMPT/timing.txt"; }
apply_once() {
  local label="$1"
  python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" "$label" "$SHA" "$DEADLINE" > /dev/null
  mark "${label}_start"
  "$ATTEMPT/markitect" apply --repo "$REPO" --model .markitect/model.yaml \
    --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
    --host-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-$label.json" \
    > "$ATTEMPT/out/apply-$label.json" 2> "$ATTEMPT/out/apply-$label.stderr"
  local code=$?
  mark "${label}_end"
  echo "${label}_exit=$code" >> "$ATTEMPT/timing.txt"
  return $code
}
mark projection_start
bash "$ATTEMPT/frozen/ra_project.sh" "$SRC" "$REPO" > "$ATTEMPT/out/baseline.txt" 2>&1 || exit 3
mark projection_end
apply_once a1 || { echo "outcome=A1_NOT_APPLIED" >> "$ATTEMPT/timing.txt"; exit 4; }
# The owner's two steps between the Applies: take A1 as it is, change only the limit.
cd "$REPO"
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" add -A
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" commit --quiet -m "Take the A1 publication unchanged"
A1=$(git rev-parse HEAD)
POLICY=.markitect/model/lending/loan-policy.yaml
test "$(grep -c 'maxOpenLoans: 5' "$POLICY")" = 1 || exit 5
sed -i 's/maxOpenLoans: 5/maxOpenLoans: 3/' "$POLICY"
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" commit --quiet -am "M2: lower the open loan limit from 5 to 3"
M2=$(git rev-parse HEAD)
test "$(git diff --name-only "$A1" "$M2")" = "$POLICY" || exit 6
{ echo "a1_commit=$A1"; echo "m2_commit=$M2"; } >> "$ATTEMPT/timing.txt"
cd - > /dev/null
apply_once a2 || { echo "outcome=A2_NOT_APPLIED" >> "$ATTEMPT/timing.txt"; exit 7; }
echo "outcome=BOTH_APPLIES_RETURNED_0" >> "$ATTEMPT/timing.txt"
