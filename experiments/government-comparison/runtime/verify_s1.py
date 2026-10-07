"""Finite deterministic verification only. Never imports a native launch entrypoint."""
import argparse
import hashlib
import json
from pathlib import Path
import platform
import subprocess
import sys
import tempfile
import unittest
import zipfile

from context_smoke import prepare, probe_current_identity
from dispatch import digest, encoded, runtime_pins
import test_dispatch


def verify(destination):
    destination = Path(destination).resolve()
    destination.mkdir(parents=True, exist_ok=False)
    checkout = Path(__file__).resolve().parents[3]
    historical = [checkout / "experiments/government-comparison/.study-data" / name for name in
                  ("actual-runner-authorized-starts.sqlite", "runner-01601-diagnostic.sqlite")]
    before = {str(path): digest(path.read_bytes()) for path in historical if path.exists()}
    if len(before) != len(historical):
        raise ValueError("both historical ledgers must exist to verify preservation")
    modules = ("test_dispatch", "test_context_smoke", "test_runtime")
    suite = unittest.TestSuite(unittest.defaultTestLoader.loadTestsFromName(name) for name in modules)
    with (destination / "targeted-tests.log").open("w", encoding="utf-8") as log:
        log.write("Deterministic local mechanics only; no Actor/model/inference invocation.\n")
        log.write("Retained dispatch evidence: " + str(test_dispatch.ROOT) + "\n")
        result = unittest.TextTestRunner(stream=log, verbosity=2).run(suite)
    context_root = Path(tempfile.mkdtemp(prefix="markitect-s1-context-retained-")) / "synthetic"
    manifest = prepare(context_root)
    access = probe_current_identity(context_root)
    records = []
    with zipfile.ZipFile(destination / "mechanical-records.zip", "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for label, root in (("dispatch", test_dispatch.ROOT), ("context", context_root)):
            for path in sorted(root.rglob("*")):
                if path.is_file():
                    name = label + "/" + path.relative_to(root).as_posix()
                    raw = path.read_bytes()
                    archive.writestr(name, raw)
                    records.append({"path": name, "sha256": digest(raw), "bytes": len(raw)})
    (destination / "mechanical-records-manifest.json").write_bytes(encoded(records) + b"\n")
    after = {str(path): digest(path.read_bytes()) for path in historical if path.exists()}
    observation = {"schemaVersion": 1, "kind": "s1-infrastructure-mechanics-only", "actorStarts": 0,
                   "inferenceCalls": 0, "nativeRunnerStarts": 0, "studyCellsExecuted": 0,
                   "testsRun": result.testsRun, "failures": len(result.failures), "errors": len(result.errors),
                   "successful": result.wasSuccessful(), "testModules": list(modules),
                   "runtimeSourceSha256": runtime_pins(), "testSourceSha256": {name: digest(
                       Path(__file__).with_name(name + ".py").read_bytes()) for name in (*modules, "verify_s1", "context_smoke")},
                   "pythonVersion": platform.python_version(), "pythonExecutable": str(Path(sys.executable).resolve()),
                   "pythonSha256": digest(Path(sys.executable).read_bytes()),
                   "sourceHeadAtVerification": subprocess.check_output(["git", "-C", str(checkout), "rev-parse", "HEAD"], text=True).strip(),
                   "mechanicalEvidenceRoot": str(test_dispatch.ROOT), "contextEvidenceRoot": str(context_root),
                   "contextManifestSha256": manifest["manifestSha256"], "sameUserHelperReadGranted": access["controlReadGranted"],
                   "historicalLedgerHashesBefore": before, "historicalLedgerHashesAfter": after,
                   "historicalLedgersUnchanged": before == after,
                   "mechanicalArchiveSha256": digest((destination / "mechanical-records.zip").read_bytes()),
                   "limits": "unchanged commonLimits; mechanical synthetic counters are not native provider measurements",
                   "s1": "open; context/access Actor probes, native product smokes and profile decision remain outstanding"}
    (destination / "observation.json").write_bytes(encoded(observation) + b"\n")
    print(json.dumps(observation, indent=2))
    return result.wasSuccessful() and before == after


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", required=True)
    raise SystemExit(0 if verify(parser.parse_args().evidence) else 1)
