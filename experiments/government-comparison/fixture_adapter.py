"""Mechanical interchange fixture: reports missing capabilities, calls no actor."""
import argparse
import hashlib
import json
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--request", required=True)
parser.add_argument("--result", required=True)
args = parser.parse_args()
request = json.loads(Path(args.request).read_text(encoding="utf-8"))
result = {"schemaVersion": 1, "trialId": request["trialId"], "operation": request["operation"],
          "requestSha256": hashlib.sha256(Path(args.request).read_bytes()).hexdigest(),
          "mode": "fixture", "status": "readiness_gap", "candidateCommit": None,
          "capabilities": ["fixture-interchange-only"],
          "gaps": ["No actual actor runner configured", "Product adapter not supplied/frozen"],
          "receipts": [], "usage": None}
Path(args.result).write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
