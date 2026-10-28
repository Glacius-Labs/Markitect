#!/usr/bin/env bash
# Arm P Library smoke chain (smoke-dispatch.md): S1 fresh M1, S2 M1 -> M2,
# S3 unchanged re-apply, S4 seeded defect, one attempt each, sequential,
# then the frozen assessment of every candidate exactly as for arms M and B.
#   bash smoke-chain.sh EVIDENCE_DIR
# Run from Git Bash on Windows, detached from any Claude Code session.
set -u
EV="$1"
HERE="$(cd "$(dirname "$0")" && pwd)"
SRC="$(git -C "$HERE" rev-parse --show-toplevel)"
COMPARISON="$SRC/tools/comparison"
export TMP="$LOCALAPPDATA/Temp" TEMP="$LOCALAPPDATA/Temp" MSYS_NO_PATHCONV=1

# Binding: arm M's inputs, the frozen assessor and oracle, the shared engine lock.
REQ='C:/Users/Consiliari/Documents/Codex/Markitect-V1-Requalification/e11e6cff/packets'
E='C:/Users/Consiliari/Documents/Codex/Markitect-V1-Comparison/e11e6cff883b'
S1_BUNDLE="$REQ/m-s1-c1/attempt-01/start.bundle"; S1_SHA=ab50684023b90d0cfe84343905b2c6ab32c5cc2f134494f0711dd2187832c9fb
S1_COMMIT=ee7c5d981291fd5f9197108c696d9f1fbdce9325
S4_BUNDLE="$E/s4/m-s4-input.bundle"; S4_SHA=0cb5e9622fd3743efa79734ed2e67b3b488fa46f41862db26989c5d20f53255d
S4_COMMIT=b079bec14a50bb2ebe71eab6e48af76e8f6743c6
SEED="$E/s4/seeded-lines.json"; SEED_SHA=9a8c9e54371fa50d58fc270d817b41f7acc3d87be0ff3f8c77dcde620426e654
R=74b7d7df946ee35b8c746dff9aefeeb13405b1f2
F=e11e6cff883ba16da2cddab8d54c0367cae8a5d8
GO_IMAGE=sha256:1271a363ef2dfe861fe4fe1842d8434007f3d58072458837e7117b50bf600ba9
LOCK='C:\Users\Consiliari\AppData\Roaming\Claude\side-session-notes\local_199078bb-6883-47fa-8219-a00bc9e8a848\NATIVE-RUN.lock'
ENGINE='C:\Program Files\Docker\Docker\resources\bin\docker.exe'
WORK='C:/Users/Consiliari/AppData/Local/Temp/mk-armp'
MODEL=claude-sonnet-5-5
EFFORT=high
DEADLINE=2700
MAX_AGENTS=150

say() { echo "=== $* $(date -u +%FT%TZ)"; }
die() { say "STOP: $*"; exit 1; }

[ -e "$EV" ] && die "$EV exists"
[ -z "$(git -C "$SRC" status --porcelain)" ] || die "the source worktree has local changes"
mkdir -p "$EV/bin" "$EV/inputs" "$EV/runs" "$EV/assess" "$EV/blind" "$EV/blind-key" "$WORK"
echo $$ > "$EV/chain.pid"
SOURCE=$(git -C "$SRC" rev-parse HEAD)
check_sha() { [ "$(sha256sum "$1" | cut -d' ' -f1)" = "$2" ] || die "sha256 of $1 differs"; }
check_sha "$S1_BUNDLE" "$S1_SHA"; check_sha "$S4_BUNDLE" "$S4_SHA"; check_sha "$SEED" "$SEED_SHA"
cp "$S1_BUNDLE" "$EV/inputs/p-s1-c1.bundle"; cp "$S4_BUNDLE" "$EV/inputs/p-s4-r1.bundle"

# The compiler is arm M's frozen source F plus frozen-tree.patch (`markitect
# tree`): arm M's inputs carry the Core of F, which later Core versions refuse.
PATCH="$HERE/frozen-tree.patch"
BUILD="$WORK/frozen-build"
say "build markitect from $F + frozen-tree.patch (plugin source $SOURCE)"
rm -rf "$BUILD"; git -C "$SRC" worktree prune
git -C "$SRC" worktree add -q --detach "$BUILD" "$F" || die "frozen worktree failed"
git -C "$BUILD" apply "$PATCH" || die "frozen-tree.patch does not apply to $F"
(cd "$BUILD" && env -u GOFLAGS GOWORK=off GOENV=off CGO_ENABLED=0 go build -trimpath -buildvcs=true -o "$EV/bin/markitect.exe" ./cmd/markitect) || die "build failed"
git -C "$SRC" worktree remove --force "$BUILD" || die "frozen worktree not removed"
cat > "$EV/binding.json" <<EOF
{"source": "$SOURCE", "markitect_source": "$F", "markitect_patch_sha256": "$(sha256sum "$PATCH" | cut -d' ' -f1)",
 "markitect_sha256": "$(sha256sum "$EV/bin/markitect.exe" | cut -d' ' -f1)",
 "claude": "$(claude --version)", "model": "$MODEL", "effort": "$EFFORT", "deadline_seconds": $DEADLINE,
 "max_agents": $MAX_AGENTS, "s1": "$S1_COMMIT", "s4": "$S4_COMMIT", "assessor_rev": "$R", "oracle_rev": "$F", "go_image": "$GO_IMAGE"}
EOF

field() { python -c "import json,sys; v=json.load(open(sys.argv[1])).get(sys.argv[2]); print(v if v else '')" "$1" "$2"; }

run_step() {  # LABEL INPUT_BUNDLE COMMIT [OWNER_BRANCH]
  say "run $1"
  local branch=(); [ -n "${4:-}" ] && branch=(--branch "$4")
  python -I -B "$COMPARISON/armp.py" run --input "$2" --commit "$3" --out "$EV/runs/$1" --label "$1" \
    --markitect "$EV/bin/markitect.exe" --work-root "$WORK" --model "$MODEL" --effort "$EFFORT" \
    --deadline-seconds "$DEADLINE" --max-agents "$MAX_AGENTS" --lock "$LOCK" "${branch[@]}"
  say "run $1 exit $?"
  touch "$EV/runs/$1.done"
}

assess_step() {  # LABEL RUN_ID PROFILE [SEED]
  local rc; rc=$(field "$EV/runs/$1/result.json" result_commit)
  [ -n "$rc" ] || { say "assess $2 skipped: no candidate"; return; }
  local token; token=$(python -c "import secrets; print(secrets.token_hex(6))")
  echo "$token" > "$EV/blind-key/$2.token"
  local args=(--candidate "$EV/runs/$1/result.bundle" --commit "$rc" --start "$EV/runs/$1/start.bundle"
              --start-commit "$(field "$EV/runs/$1/result.json" input_commit)" --profile "$3" --run-id "$2"
              --out "$EV/assess/$2" --rev "$R" --oracle-rev "$F" --go-image "$GO_IMAGE" --repo "$E/source"
              --engine "$ENGINE" --lock "$LOCK" --blind "$EV/blind" --token "$token")
  [ -n "${4:-}" ] && args+=(--seed-lines "$4")
  say "assess $2"
  python -I -B "$E/source/tools/comparison/assess_candidate.py" "${args[@]}"
  say "assess $2 exit $?"
}

run_step p-s1-c1 "$EV/inputs/p-s1-c1.bundle" "$S1_COMMIT"
S1_RESULT=$(field "$EV/runs/p-s1-c1/result.json" result_commit)
if [ -n "$S1_RESULT" ]; then
  python -I -B "$COMPARISON/armp.py" input m2 --from "$EV/runs/p-s1-c1/result.bundle" --commit "$S1_RESULT" \
    --out "$EV/inputs/p-s2-c1.bundle" > "$EV/inputs/p-s2-c1.json" || die "S2 input failed"
  run_step p-s2-c1 "$EV/inputs/p-s2-c1.bundle" "$(field "$EV/inputs/p-s2-c1.json" commit)"
  S2_RESULT=$(field "$EV/runs/p-s2-c1/result.json" result_commit)
  if [ -n "$S2_RESULT" ]; then
    run_step p-s3-c1 "$EV/runs/p-s2-c1/result.bundle" "$S2_RESULT" dev/library-m2
  else say "S3 skipped: S2 has no candidate"; fi
else say "S2 and S3 skipped: S1 has no candidate"; fi
run_step p-s4-r1 "$EV/inputs/p-s4-r1.bundle" "$S4_COMMIT"

assess_step p-s1-c1 P-c1-S1 m1
assess_step p-s2-c1 P-c1-S2 m2
assess_step p-s3-c1 P-c1-S3 m2
assess_step p-s4-r1 P-r1-S4 m2 "$SEED"

python - "$EV" <<'EOF'
import json, pathlib, sys
ev = pathlib.Path(sys.argv[1]); rows = []
for run, aid in (("p-s1-c1", "P-c1-S1"), ("p-s2-c1", "P-c1-S2"), ("p-s3-c1", "P-c1-S3"), ("p-s4-r1", "P-r1-S4")):
    r = ev / "runs" / run / "result.json"
    if not r.exists():
        rows.append({"run": run, "ran": False}); continue
    r = json.loads(r.read_text(encoding="utf-8"))
    a = ev / "assess" / aid / "assessment.json"
    a = json.loads(a.read_text(encoding="utf-8")) if a.exists() else {}
    pub = r.get("publication") or {}
    rows.append({"run": run, "status": r.get("status"), "technical_pass": r.get("technical_pass"), "error": r.get("error"),
                 "outcome": pub.get("outcome"), "agents": pub.get("agents"), "seconds": (r.get("session") or {}).get("seconds"),
                 "no_change": r.get("result_tree") == r.get("input_tree") if r.get("result_tree") else None,
                 "tokens": (r.get("usage") or {}).get("total"), "mechanical_pass": a.get("mechanical_pass"),
                 "deliverable": a.get("deliverable"), "residue": r.get("residue"), "quarantine": r.get("quarantine")})
(ev / "summary.json").write_text(json.dumps(rows, indent=1), encoding="utf-8")
print(json.dumps(rows, indent=1))
EOF
say "chain done"
touch "$EV/chain.done"
