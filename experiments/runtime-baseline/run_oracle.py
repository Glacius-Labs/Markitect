"""Runs inside the assessment container: bind the manifest, run the oracle.

Mounts: /out holds the built program and is read-only, /assess/realization
is read-only, /evaluator holds the playground files at a fixed revision,
read-only. The report goes to standard output, never to a file the program
under test could reach.
"""
import argparse
import hashlib
import json
import sys

sys.path.insert(0, "/evaluator")
import assess as oracle  # noqa: E402  (the evaluator from experiments/library-playground)

p = argparse.ArgumentParser()
p.add_argument("--label", required=True)
p.add_argument("--profile", required=True)
p.add_argument("--source", required=True)
p.add_argument("--commit", required=True)
p.add_argument("--limits", required=True)
p.add_argument("--protected", required=True, help="evaluator file hashes computed on the Host")
a = p.parse_args()

with open("/out/library", "rb") as f:
    program = hashlib.sha256(f.read()).hexdigest()
manifest = {
    "experiment_id": "durchstich-" + a.label,
    "artifact_id": a.label,
    "model_digest": "not verified by this harness (library M1 at evaluator commit " + a.commit + ")",
    "source_tree": a.source,
    "profile": a.profile,
    "command": ["/out/library"],
    "cwd": "/assess/realization",
    "program_sha256": {"/out/library": program},
    "protected_sha256": json.loads(a.protected),
    "limits": json.loads(a.limits),
}
raw = json.dumps(manifest, indent=2).encode()
report = oracle.assess(json.loads(raw), raw)
json.dump(report, sys.stdout)
sys.exit(0 if report.get("passed") else 1)
