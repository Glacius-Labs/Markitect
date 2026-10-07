"""Preserve public preparation evidence/digests without exporting private cases."""
import argparse
import json
from pathlib import Path
import shutil

from prepare import ROOT, digest


def freeze(smoke_path):
    source = Path(smoke_path).resolve()
    result = json.loads(source.read_text(encoding="utf-8"))
    if result.get("kind") != "fixture-smoke" or result.get("status") != "passed" or result.get("liveTrials") != 0:
        raise ValueError("Expected passing fixture-only smoke")
    evidence = ROOT / "evidence"
    evidence.mkdir(exist_ok=True)
    target = evidence / "fixture-smoke.json"
    if target.exists():
        raise ValueError("Evidence already frozen; create a new version, do not overwrite")
    shutil.copyfile(source, target)
    for name in ("build.log", "api.log"):
        shutil.copyfile(source.parent / name, evidence / name)
    for cell in result["cellProbes"]:
        origin = source.parent / "probe" / cell["trialId"]
        destination = evidence / "probes" / cell["trialId"]
        destination.mkdir(parents=True)
        for name in ("request.json", "result.json", "events.jsonl", "stdout.log", "stderr.log"):
            shutil.copyfile(origin / name, destination / name)
    files = {}
    for p in sorted(ROOT.rglob("*")):
        relative = p.relative_to(ROOT)
        if not p.is_file() or any(part in (".study-data", "__pycache__", "bin", "obj", "evidence") for part in relative.parts):
            continue
        if p.name == "preparation-freeze.json":
            continue
        files[relative.as_posix()] = digest(p)
    private = ROOT / ".study-data" / "private"
    frozen = {"schemaVersion": 1, "kind": "preparation-freeze", "date": "2026-10-07", "liveProtocolFrozen": False,
              "sourceBaseline": "1ea5c76f55526fc4d721e865885436153f48b497",
              "designTransferCommit": "b9e17c9896f14b4b1268e4bcb005395f5d324384",
              "designOwnerCommit": "1600eebd4cf686faa0d4317e0efe29e1c1f08bce",
              "adapterContractCommit": "775338555d2f2743128462af13d998c7d179c7ac",
              "toolchain": {"python": "3.13.3", "git": "2.52.0.windows.1", "go": "1.27.1",
                            "dotnetSdk": "10.0.103", "dotnetRuntime": "10.0.3", "aspnetRuntime": "10.0.3",
                            "sqliteProvider": "10.0.3", "sqlitePclRaw": "3.0.5", "nativeSqlite": "3.53.4"},
              "starts": result["starts"], "sources": files,
              "publicEvidence": {p.relative_to(ROOT).as_posix(): digest(p) for p in sorted(evidence.rglob("*")) if p.is_file()},
              "privateAssessmentDigests": {p.name: digest(p) for p in sorted(private.glob("*")) if p.is_file()},
              "privateHoldoutExecuted": False, "task6EquivalenceMappings": None,
              "studyClassicVersion": None, "studyGovernmentVersion": None,
              "actualRunnerProfile": None, "liveTrials": 0,
              "readiness": "fixture preparation passed; actual product arms and live measurement remain unready"}
    (ROOT / "preparation-freeze.json").write_text(json.dumps(frozen, indent=2) + "\n", encoding="utf-8")
    return {"status": "frozen-preparation-only", "sources": len(files), "privateAssessmentDigests": len(frozen["privateAssessmentDigests"]),
            "liveTrials": 0, "freezeSha256": digest(ROOT / "preparation-freeze.json")}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--smoke-result", required=True)
    args = parser.parse_args()
    print(json.dumps(freeze(args.smoke_result), indent=2))
