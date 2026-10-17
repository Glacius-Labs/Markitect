"""Runs inside the assessment container: write the manifest, then the oracle.

Mounts: /out (built program, writable), /assess/realization (read-only),
/evaluator (playground files at a fixed commit, read-only).
"""
import argparse
import hashlib
import json
import subprocess
import sys

p = argparse.ArgumentParser()
p.add_argument("--label", required=True)
p.add_argument("--profile", required=True)
p.add_argument("--source", required=True)
p.add_argument("--commit", required=True)
p.add_argument("--limits", required=True)
a = p.parse_args()

with open("/out/library", "rb") as f:
    program = hashlib.sha256(f.read()).hexdigest()
protected = json.loads(subprocess.run([sys.executable, "-B", "/evaluator/assess.py", "--hashes"],
                                      check=True, capture_output=True, text=True).stdout)
manifest = {
    "experiment_id": "durchstich-" + a.label,
    "artifact_id": a.label,
    "model_digest": "not verified by this harness (library M1 at evaluator commit " + a.commit + ")",
    "source_tree": a.source,
    "profile": a.profile,
    "command": ["/out/library"],
    "cwd": "/assess/realization",
    "program_sha256": {"/out/library": program},
    "protected_sha256": protected,
    "limits": json.loads(a.limits),
}
with open("/out/manifest.json", "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2)
sys.exit(subprocess.run([sys.executable, "-B", "/evaluator/assess.py", "--manifest", "/out/manifest.json",
                         "--output", "/out/assessment.json"]).returncode)
