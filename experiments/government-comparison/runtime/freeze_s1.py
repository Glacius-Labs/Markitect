"""Freeze only explicit public S1 infrastructure sources and mechanical evidence."""
import argparse
import json
from pathlib import Path
import subprocess
import zipfile

from dispatch import digest, encoded, runtime_pins


def freeze(evidence_path, output):
    root = Path(__file__).parents[1]
    evidence = Path(evidence_path).resolve()
    observation = json.loads((evidence / "observation.json").read_bytes())
    if (not observation["successful"] or not observation["historicalLedgersUnchanged"]
            or observation["actorStarts"] != 0 or observation["inferenceCalls"] != 0
            or observation["runtimeSourceSha256"] != runtime_pins()):
        raise ValueError("passing exact-source mechanics-only verification required")
    with zipfile.ZipFile(evidence / "mechanical-records.zip") as archive:
        manifest = json.loads((evidence / "mechanical-records-manifest.json").read_bytes())
        if sorted(archive.namelist()) != sorted(item["path"] for item in manifest):
            raise ValueError("mechanical archive inventory mismatch")
        for item in manifest:
            raw = archive.read(item["path"])
            if len(raw) != item["bytes"] or digest(raw) != item["sha256"]:
                raise ValueError("mechanical archive digest mismatch")
    names = ["README.md", "harness.py", "prepare.py", ".gitattributes"]
    names += ["runtime/" + name for name in runtime_pins() if name not in {"harness.py", "prepare.py"}]
    names += ["runtime/" + name + ".py" for name in
              ("context_smoke", "test_context_smoke", "test_dispatch", "test_runtime", "verify_s1", "freeze_s1")]
    names += ["runtime/runner-pin.json", "runtime/classic-pin.json"]
    names += ["public/" + name for name in ("adapter-contract.md", "runner-readiness.md", "resource-proposal.json",
              "context-access-smoke.md", "dispatch-enforcement.md", "s1-dispatch.md", "s1-next-request.json",
              "s1-profile-correction-proposal.md")]
    sources = {name: digest((root / name).read_bytes()) for name in sorted(set(names))}
    head = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
    if head != observation["sourceHeadAtVerification"]:
        raise ValueError("source HEAD differs from verification")
    for name, sha in sources.items():
        committed = subprocess.check_output(["git", "-C", str(root), "show", head + ":experiments/government-comparison/" + name])
        if digest(committed) != sha:
            raise ValueError("source is not the committed candidate: " + name)
    prior = subprocess.check_output(["git", "-C", str(root), "show",
        "b16d70407016ca25651935c2c9283ed5511212e1:experiments/government-comparison/public/resource-proposal.json"])
    if prior != (root / "public/resource-proposal.json").read_bytes():
        raise ValueError("existing resource proposal changed")
    value = {"schemaVersion": 1, "kind": "s1-infrastructure-mechanics-freeze", "sourceCandidate": head,
             "sources": sources, "evidence": {p.relative_to(root).as_posix(): digest(p.read_bytes())
              for p in sorted(evidence.iterdir()) if p.is_file()},
             "mechanicalArchiveEntries": len(manifest), "actorStarts": 0, "inferenceCalls": 0,
             "commonLimitsUnchanged": True, "historicalDiagnosticGrantsExhausted": True,
             "liveProtocolFrozen": False, "liveRunGrantIssued": False, "s1": "open"}
    output = Path(output)
    with output.open("xb") as out:
        out.write(encoded(value) + b"\n")
    print(json.dumps({"sourceCandidate": head, "sources": len(sources), "evidence": len(value["evidence"]),
                      "mechanicalArchiveEntries": len(manifest), "freezeSha256": digest(output.read_bytes())}, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    freeze(args.evidence, args.output)
