"""Library oracle as a required Apply check.

Runs in the check container with the candidate's realization directory as the
working directory. The evaluator files live in this image, outside the
Agents' reach. The policy profile follows the bound model in the candidate:
maxOpenLoans 5 selects m1, 3 selects m2. Builds ./cmd/library into /tmp and
runs the independent oracle; prints only case IDs and pass/fail, so the
output does not hand the expected values to the Agents.
"""
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

EVALUATOR = Path("/opt/oracle/evaluator")
sys.path.insert(0, str(EVALUATOR))
import assess as oracle  # noqa: E402

LIMITS = {"request_seconds": 2, "process_seconds": 30, "request_bytes": 16384, "output_bytes": 262144}


def main():
    realization = Path.cwd()
    policy = realization.parent / ".markitect/model/lending/loan-policy.yaml"
    match = re.search(r"maxOpenLoans:\s*(\d+)", policy.read_text(encoding="utf-8"))
    profile = {"5": "m1", "3": "m2"}.get(match.group(1) if match else "")
    if profile is None:
        print("library oracle: the bound loan policy selects no known profile")
        return 2
    out = Path("/tmp/oracle")
    out.mkdir(parents=True, exist_ok=True)
    build = subprocess.run(["/usr/local/go/bin/go", "build", "-o", str(out / "library"), "./cmd/library"],
                           capture_output=True, text=True)
    if build.returncode != 0:
        print("library oracle: go build ./cmd/library failed")
        print((build.stdout + build.stderr)[-2000:])
        return 1
    program = hashlib.sha256((out / "library").read_bytes()).hexdigest()
    protected = oracle.protected_hashes()
    manifest = {
        "experiment_id": "apply-check-" + profile,
        "artifact_id": "candidate",
        "model_digest": "bound model in the candidate (" + profile + ")",
        "source_tree": str(realization),
        "profile": profile,
        "command": [str(out / "library")],
        "cwd": str(realization),
        "program_sha256": {str(out / "library"): program},
        "protected_sha256": protected,
        "limits": LIMITS,
    }
    raw = json.dumps(manifest, indent=2).encode()
    report = oracle.assess(json.loads(raw), raw)
    cases = report.get("cases", [])
    passed = sum(1 for c in cases if c.get("passed"))
    discrimination = (report.get("discrimination") or {}).get("expected_mismatch_confirmed")
    print(f"library oracle profile {profile}: {passed}/{len(cases)} cases passed; opposite-profile discrimination confirmed: {discrimination}")
    for case in cases:
        if not case.get("passed"):
            print("failed case:", case.get("id") or case.get("name") or "unnamed")
    for error in report.get("errors") or []:
        print("oracle error:", str(error)[:300])
    return 0 if report.get("passed") else 1


if __name__ == "__main__":
    sys.exit(main())
