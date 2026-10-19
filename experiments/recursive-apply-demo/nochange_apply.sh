#!/usr/bin/env bash
# Separate check after the demonstration: Apply M2 again on the accepted A2
# (no model change), on the final head. Expected: Verify only, no Execute,
# no_change publication, provider homes removed.
# Usage: nochange_apply.sh BUILD_DIR DEMO_ATTEMPT NEW_ATTEMPT
set -uo pipefail
BUILD="$1"; DEMO_ATTEMPT="$2"; ATTEMPT="$3"
DEMO="$(cd "$(dirname "$0")" && pwd)"
bash "$DEMO/ra_bind.sh" "$BUILD" "$ATTEMPT" > /dev/null || exit 2
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
mkdir -p "$ATTEMPT/out"
cp -a "$DEMO_ATTEMPT/repo" "$ATTEMPT/repo"
cd "$ATTEMPT/repo"
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" add -A
git -c user.name="Markitect demo owner" -c user.email="demo@markitect.invalid" commit --quiet -m "Take the A2 publication unchanged"
echo "a2_commit=$(git rev-parse HEAD)" > "$ATTEMPT/timing.txt"
cd - > /dev/null
DEADLINE=$(date -u -d "+20 minutes" +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" n1 "$SHA" "$DEADLINE" > /dev/null
start=$(date +%s)
timeout --kill-after=30 1260 "$ATTEMPT/markitect" apply --repo "$ATTEMPT/repo" --model .markitect/model.yaml \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --host-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-n1.json" \
  > "$ATTEMPT/out/apply-n1.json" 2> "$ATTEMPT/out/apply-n1.stderr"
echo "apply exit=$? seconds=$(( $(date +%s) - start ))"
bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" | grep containers
RUN=$(ls "$ATTEMPT/state-n1/runs")
echo "leftover volumes of $RUN: $(docker volume ls --format '{{.Name}}' | grep -c "^$RUN" || true)"
git -C "$ATTEMPT/repo" status --porcelain | wc -l
python3 "$DEMO/analyze.py" "$ATTEMPT/state-n1" "$ATTEMPT/out/apply-n1.json" > "$ATTEMPT/out/analysis-n1.txt" 2>&1
python3 - "$ATTEMPT/out/apply-n1.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
r = d.get("run") or {}
print("status", d.get("status"), "run", r.get("status"), "publication", (d.get("publication") or {}).get("Outcome"), "retained", d.get("retainedWork"))
print("turns", len(r.get("turns") or []), {k: sum(1 for t in r["turns"] if t["kind"] == k) for k in ("execute", "verify")})
print("rounds", [(x["number"], x["green"]) for x in r.get("rounds") or []], "findings", len(r.get("findings") or []))
print("usage", r.get("usage"))
PY
