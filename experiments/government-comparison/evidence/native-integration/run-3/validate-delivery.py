"""Verify immutable and additive R3 delivery evidence without any execution."""
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[3]
REPO = ROOT.parents[1]
spec = importlib.util.spec_from_file_location("r3_driver", ROOT / "run-native-integration-r3.py")
d = importlib.util.module_from_spec(spec)
spec.loader.exec_module(d)
baseline = "f5cd8871b84f45541044110057b59f0bb1ca0804"
freeze = json.loads((d.EVIDENCE / "preflight-freeze.json").read_bytes())
for item in freeze["inputs"]:
    assert d.binding(item["path"]) == item
historical = subprocess.check_output(
    ["git", "ls-tree", "-r", "--name-only", baseline, "--",
     "experiments/government-comparison/evidence/native-integration/"], cwd=REPO, text=True).splitlines()
queries = [(baseline + ":" + name).encode() for name in historical]
raw = subprocess.check_output(["git", "cat-file", "--batch"], cwd=REPO, input=b"\n".join(queries) + b"\n")
offset = 0
for name in historical:
    end = raw.index(b"\n", offset)
    size = int(raw[offset:end].split()[-1])
    committed = raw[end + 1:end + 1 + size]
    assert (REPO / name).read_bytes() == committed, name
    offset = end + 1 + size + 1
old_pairs = json.loads((ROOT / "evidence/native-integration/run-2/external-snapshots.json").read_bytes())
old_unchanged, old_budget_appended = 0, 0
for item in old_pairs:
    assert d.binding(item["snapshotPath"])["sha256"] == item["sha256"]
    if Path(item["originalPath"]).resolve() == (d.LEGACY / "native-starts.sqlite").resolve():
        old_budget_appended += 1
    else:
        assert d.binding(item["originalPath"])["sha256"] == item["sha256"]
        old_unchanged += 1
pairs = json.loads((d.EVIDENCE / "external-snapshot-manifest.json").read_bytes())
for item in pairs:
    assert d.binding(item["original"]["path"]) == item["original"]
    assert d.binding(item["snapshot"]["path"]) == item["snapshot"]
    assert item["original"]["sha256"] == item["snapshot"]["sha256"]
summary = json.loads((d.EVIDENCE / "quota-and-ledger-summary.json").read_bytes())
assert summary["newConsumed"] == {"nativeStarts": 6, "wrapperAttempts": 4,
                                  "delegates": 4, "reservedSessionSeconds": 900}
assert summary["cumulative"] == {"nativeStarts": 11, "reservedSessionSeconds": 1650}
assert summary["historyPreserved"]
gov = json.loads((d.EVIDENCE / "government/queue-result.json").read_bytes())
classic = json.loads((d.EVIDENCE / "classic/flow-result.json").read_bytes())
assert gov["status"] == "incomplete" and classic["status"] == "completed"
assert not (d.EVIDENCE / "government/resume-result.json").exists()
assert classic["classic"]["errors"] == []
review = json.loads((d.EXTERNAL / "classic/execute-review.json").read_bytes())
report = d.EXTERNAL / "classic/outer-controller-evidence/classic-native/execute.json"
assert review["executeReportSha256"] == d.binding(report)["sha256"]
assert review["executeDigest"] == json.loads(report.read_bytes())["digest"]
for arm in summary["arms"].values():
    outer = arm["outerControllerProcessReceipt"]
    processes = outer.get("actions", [outer.get("process")])
    assert processes and all(p["wallSeconds"] <= 38 and p.get("stopReason") is None for p in processes)
assert summary["arms"]["classic"]["controllerElapsedSeconds"] <= 180
assert summary["arms"]["government"]["controllerElapsedSeconds"] <= 38
paths = {p for p in d.EVIDENCE.rglob("*") if p.is_file()}
paths.update(ROOT / name for name in (
    "native-integration-handoff.md", "run-native-integration-r3.py",
    "runtime/government_roles.py", "runtime/native_controller.py", "runtime/native_fixture_budget.py",
    "runtime/test_native_fixture_budget.py", "runtime/test_native_r3_diagnostics.py",
    "runtime/test_r3_classic_preflight.py", "runtime/test_native_r3_checkpoints.py",
    "public/native-integration-r3-native-preflight-review.md",
    "public/native-integration-r3-post-run-review.md"))
value = {
    "kind": "R3 terminal deterministic native integration, qualified mixed outcome",
    "acceptedOfflinePredecessor": baseline,
    "nativeFreezeCommit": "5795069128586977d1be66af6564846e2915c166",
    "grantKey": d.R3_KEY, "amendmentKey": "r3-a1",
    "unchangedFrozenInputs": len(freeze["inputs"]),
    "historicalGitFilesUnchanged": len(historical),
    "r2ExternalSnapshotPairsUnchanged": len(old_pairs),
    "r2OriginalFilesUnchanged": old_unchanged,
    "originalBudgetAppendedWithPriorRowsPreserved": old_budget_appended == 1,
    "newExternalCopyPairsVerified": len(pairs),
    "outcomes": {"government": "incomplete; first failed Queue; no Resume or retry",
                 "classic": "completed bounded mechanics; original narrative rendering qualified by separate erratum"},
    "consumption": summary["newConsumed"], "cumulative": summary["cumulative"],
    "selectedOfflineTests": 38, "initialCommandErrorPreserved": True,
    "newRealStudyActors": 0, "newProviderRuns": 0, "newMetadataSessions": 0, "newStudyCells": 0,
    "s1Complete": False, "semanticAcceptance": False, "humanAcceptance": False,
    "files": [d.binding(p) for p in sorted(paths, key=str)]}
d.write_new(d.EVIDENCE / "delivery-validation.json", value)
print(json.dumps({key: value[key] for key in (
    "unchangedFrozenInputs", "historicalGitFilesUnchanged", "r2OriginalFilesUnchanged",
    "originalBudgetAppendedWithPriorRowsPreserved", "newExternalCopyPairsVerified", "consumption")}))
