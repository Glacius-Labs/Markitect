"""Verify new offline evidence and preservation of closed R1/R2/R3 artifacts."""
import hashlib
import json
from pathlib import Path
import sqlite3
import subprocess

AREA = Path(__file__).resolve().parent
ROOT = AREA.parents[1]
REPO = ROOT.parents[1]
BASE = "cd537ac11765097cb46e9326f295d74059e827a3"
assert (AREA / "independent-review.md").is_file(), "independent review is required"


def pin(path):
    return {"path": str(path.resolve()), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}


prefix = "experiments/government-comparison/"
names = subprocess.check_output(
    ["git", "ls-tree", "-r", "--name-only", BASE, "--", prefix + "evidence/native-integration/",
     prefix + "public/"], cwd=REPO, text=True).splitlines()
names = [name for name in names if "/evidence/native-integration/" in name or
         "/public/native-integration" in name]
queries = [(BASE + ":" + name).encode() for name in names]
raw = subprocess.check_output(["git", "cat-file", "--batch"], cwd=REPO,
                              input=b"\n".join(queries) + b"\n")
offset = 0
for name in names:
    end = raw.index(b"\n", offset)
    size = int(raw[offset:end].split()[-1])
    assert (REPO / name).read_bytes() == raw[end + 1:end + 1 + size], name
    offset = end + 1 + size + 1
old_handoff = subprocess.check_output(["git", "show", BASE + ":" + prefix + "native-integration-handoff.md"], cwd=REPO)
assert (ROOT / "native-integration-handoff.md").read_bytes().endswith(old_handoff)
run3 = ROOT / "evidence/native-integration/run-3"
pairs = json.loads((run3 / "external-snapshot-manifest.json").read_bytes())
for item in pairs:
    assert pin(Path(item["original"]["path"])) == item["original"]
    assert pin(Path(item["snapshot"]["path"])) == item["snapshot"]
summary = json.loads((run3 / "quota-and-ledger-summary.json").read_bytes())
budget = Path(summary["nativeBudget"]["path"])
assert pin(budget) == summary["nativeBudget"]
db = sqlite3.connect(budget.resolve().as_uri() + "?mode=ro", uri=True)
try:
    counts = db.execute("SELECT COUNT(*),SUM(reserved_seconds),SUM(finished IS NULL) FROM starts").fetchone()
finally:
    db.close()
assert counts == (11, 1650, 0)
advanced = {str((ROOT / name).resolve()) for name in (
    "runtime/government_roles.py", "runtime/fixtures/government_positive/deterministic_delegate.py")}
freeze = json.loads((run3 / "preflight-freeze.json").read_bytes())
changed = [item for item in freeze["inputs"] if pin(Path(item["path"])) != item]
assert {str(Path(item["path"]).resolve()) for item in changed} == advanced
wire_manifest = json.loads((AREA / "offline-evidence-manifest.json").read_bytes())
for sample in wire_manifest["wireSamples"]:
    for key in ("input", "serializedOutput"):
        assert pin(Path(sample[key]["path"])) == sample[key]
for source in wire_manifest["sourceSnapshots"]:
    assert pin(Path(source["snapshot"]["path"])) == source["snapshot"]
    original = subprocess.check_output(["git", "show", source["sourceCommit"] + ":" + source["sourcePath"]], cwd=REPO)
    assert original == Path(source["snapshot"]["path"]).read_bytes()
paths = {p for p in AREA.rglob("*") if p.is_file()}
paths.update(ROOT / name for name in (
    "native-integration-handoff.md", "runtime/government_roles.py",
    "runtime/fixtures/government_positive/deterministic_delegate.py",
    "runtime/test_government_response_serialization.py"))
value = {
    "taskKey": AREA.name, "acceptedBase": BASE, "hostCommit": wire_manifest["sourceCommit"],
    "historicalEvidenceFilesUnchangedAgainstGit": len(names),
    "historicalR3HandoffSuffixUnchanged": True, "r3ExternalArchivePairsUnchanged": len(pairs),
    "nativeBudgetUnchanged": pin(budget), "cumulativeNativeStarts": counts[0],
    "cumulativeReservedSessionSeconds": counts[1], "unfinishedNativeReservations": counts[2],
    "closedR3FreezeSourcePathsAdvanced": [{"path": item["path"], "acceptedR3Sha256": item["sha256"],
                                         "currentOfflineSha256": pin(Path(item["path"]))["sha256"]} for item in changed],
    "closedR3FreezeReusable": False,
    "rootFocusedTests": 8, "independentReviewerOfflineTests": 14,
    "reviewerExtraPreparationTests": 6, "reviewerCommandDeviationRetained": True,
    "sourceDerivedWirePairsVerified": len(wire_manifest["wireSamples"]),
    "pinnedHostSourceBlobsVerified": len(wire_manifest["sourceSnapshots"]),
    "newNativeProductOrControllerStarts": 0, "newWrapperStarts": 0, "newDelegateProcesses": 0,
    "newStudyActorOrProviderRuns": 0, "newMetadataSessions": 0, "newStudyCells": 0,
    "historicalUnknownUsagePreserved": True, "s1Complete": False,
    "nativeGovernmentSuccessProven": False, "semanticAcceptance": False, "humanAcceptance": False,
    "files": [pin(p) for p in sorted(paths, key=str)]}
with (AREA / "delivery-validation.json").open("xb") as stream:
    stream.write((json.dumps(value, sort_keys=True, separators=(",", ":")) + "\n").encode())
print(json.dumps({key: value[key] for key in (
    "historicalEvidenceFilesUnchangedAgainstGit", "r3ExternalArchivePairsUnchanged",
    "cumulativeNativeStarts", "cumulativeReservedSessionSeconds",
    "sourceDerivedWirePairsVerified", "pinnedHostSourceBlobsVerified")}))
