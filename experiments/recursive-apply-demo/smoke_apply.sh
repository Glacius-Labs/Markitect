#!/usr/bin/env bash
# Plumbing smoke, not the demonstration: one Apply on Library M1 with a short
# deadline, to see the real chain work before the timed run.
# Usage: smoke_apply.sh BUILD_DIR ATTEMPT_DIR MINUTES
set -uo pipefail
BUILD="$1"; ATTEMPT="$2"; MIN="$3"
DEMO="$(cd "$(dirname "$0")" && pwd)"
bash "$DEMO/ra_bind.sh" "$BUILD" "$ATTEMPT" > /dev/null || exit 2
SHA=$(grep '^product_sha=' "$ATTEMPT/binding.txt" | cut -d= -f2)
SRC="${MARKITECT_DEMO_WORK:-$HOME/mk-ra}/src-$SHA"
mkdir -p "$ATTEMPT/out"
bash "$ATTEMPT/frozen/ra_project.sh" "$SRC" "$ATTEMPT/repo" > "$ATTEMPT/out/baseline.txt" 2>&1 || exit 3
DEADLINE=$(date -u -d "+$MIN minutes" +%FT%TZ)
python3 "$ATTEMPT/frozen/ra_runconfig.py" "$ATTEMPT" s1 "$SHA" "$DEADLINE" > /dev/null
start=$(date +%s)
timeout --kill-after=30 $(( MIN * 60 + 120 )) "$ATTEMPT/markitect" apply --repo "$ATTEMPT/repo" --model .markitect/model.yaml \
  --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml \
  --runtime-profile "$ATTEMPT/host/host.yaml" --run-config "$ATTEMPT/run-config-s1.json" \
  > "$ATTEMPT/out/apply-s1.json" 2> "$ATTEMPT/out/apply-s1.stderr"
echo "apply exit=$? seconds=$(( $(date +%s) - start ))"
bash "$ATTEMPT/frozen/ra_cleanup.sh" "$ATTEMPT" | tail -3
tail -5 "$ATTEMPT/out/apply-s1.stderr"
python3 - "$ATTEMPT/out/apply-s1.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
print("status", d.get("status"), "error", d.get("error"))
r = d.get("run") or {}
if "passes" not in r:
    sys.exit("no run.passes in the Apply report: " + str(d.get("error") or d.get("status")))
print("run", r.get("status"), r.get("reason"))
for t in r.get("turns") or []:
    print(" turn", t["id"], t["pass"], t["kind"], "stopped" if t["stopped"] else "ended", t.get("reason", "")[:120], round(t["usage"].get("costUSD", 0), 3))
for x in r["passes"]:
    print(" pass", x["id"], x["purpose"], "review" if x["review"] else "work", "verdict", x.get("verdict"), "sealed", x["sealed"], [(c["name"], c["passed"]) for c in x.get("checks") or []])
for f in r.get("findings") or []:
    print(" finding", f["id"], f["owner"], f["raisedBy"], f["blocking"], f["closed"], f["obligation"][:80])
print("usage", r.get("usage"))
PY
