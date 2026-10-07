"""Finite profile/admission verification; deterministic subprocesses only."""
import argparse
import json
from pathlib import Path
import platform
import subprocess
import sys
import unittest
import zipfile

from diagnostic_history import read_history
from dispatch import CHECKOUT, digest, encoded, external, runtime_pins
import test_dispatch


def verify(destination):
    root = external(destination)
    root.mkdir(parents=True, exist_ok=False)
    before = read_history()
    modules = ("test_measurement_profile", "test_dispatch", "test_runtime")
    suite = unittest.TestSuite(unittest.defaultTestLoader.loadTestsFromName(name) for name in modules)
    with (root / "targeted-tests.log").open("w", encoding="utf-8") as log:
        log.write("Deterministic profile/admission tests only; no native Actor/model/metadata call.\n")
        log.write("Retained dispatch evidence: " + str(test_dispatch.ROOT) + "\n")
        result = unittest.TextTestRunner(stream=log, verbosity=2).run(suite)
    entries = []
    with zipfile.ZipFile(root / "mechanical-records.zip", "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path in sorted(test_dispatch.ROOT.rglob("*")):
            if path.is_file():
                raw, name = path.read_bytes(), path.relative_to(test_dispatch.ROOT).as_posix()
                archive.writestr(name, raw)
                entries.append({"path": name, "sha256": digest(raw), "bytes": len(raw)})
    (root / "mechanical-records-manifest.json").write_bytes(encoded(entries) + b"\n")
    after = read_history()
    (root / "diagnostic-history.json").write_bytes(encoded(after) + b"\n")
    observation = {"schemaVersion": 1, "kind": "measurement-profile-v2-mechanics-only",
                   "actorStarts": 0, "inferenceCalls": 0, "nativeRunnerStarts": 0, "studyCellsExecuted": 0,
                   "metadataRpcCalls": 0, "testsRun": result.testsRun, "failures": len(result.failures),
                   "errors": len(result.errors), "successful": result.wasSuccessful(), "testModules": list(modules),
                   "runtimeSourceSha256": runtime_pins(), "testSourceSha256": {name: digest(
                       Path(__file__).with_name(name + ".py").read_bytes()) for name in (*modules, "verify_profile")},
                   "pythonVersion": platform.python_version(), "pythonExecutable": str(Path(sys.executable).resolve()),
                   "pythonSha256": digest(Path(sys.executable).read_bytes()),
                   "sourceHeadAtVerification": subprocess.check_output(["git", "-C", str(CHECKOUT), "rev-parse", "HEAD"], text=True).strip(),
                   "historicalLedgersUnchanged": before == after,
                   "mechanicalArchiveSha256": digest((root / "mechanical-records.zip").read_bytes()),
                   "mechanicalEvidenceRoot": str(test_dispatch.ROOT), "s1": "open; context DRAFTS only; no live grant"}
    (root / "observation.json").write_bytes(encoded(observation) + b"\n")
    print(json.dumps(observation, indent=2))
    return result.wasSuccessful() and before == after


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True)
    raise SystemExit(0 if verify(parser.parse_args().evidence) else 1)
