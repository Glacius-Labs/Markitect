"""Launch the separately reserved metadata allocation once, under the existing Job."""
import hashlib
import json
import os
from pathlib import Path
import sys
import time

ROOT = Path(__file__).resolve().parent
STUDY = ROOT.parents[2]
ALLOCATION = "local-policy-read-disabled-status-20261008"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    if sys.flags.optimize:
        raise ValueError("optimized Python is not allowed for frozen source guards")
    request = json.loads((ROOT / "request.json").read_bytes())
    freeze = json.loads((ROOT / "freeze.json").read_bytes())
    reservation = json.loads((ROOT / "reservation.json").read_bytes())
    expected = {"allocationId": ALLOCATION, "maxTrees": 1,
                "requestSha256": sha(ROOT / "request.json"),
                "clientSha256": sha(ROOT / "classified-client.py"),
                "methodEnumsSha256": sha(ROOT / "frozen-method-enums.json")}
    if any(document.get(k) != v for document in (freeze, reservation) for k, v in expected.items()):
        raise ValueError("frozen allocation binding mismatch")
    if reservation.get("freezeSha256") != sha(ROOT / "freeze.json"):
        raise ValueError("reservation freeze mismatch")
    for path, expected_sha in freeze["files"].items():
        if sha(path) != expected_sha:
            raise ValueError("frozen source mismatch")
    if sha(sys.executable) != freeze["pythonSha256"]:
        raise ValueError("Python pin mismatch")
    if request["allocationId"] != ALLOCATION or request["limits"]["wallSeconds"] != 60:
        raise ValueError("request allocation/outer limit mismatch")
    # This durable claim consumes the one outer launch opportunity even if the
    # bounded launcher or native child subsequently fails. Never retry it.
    with (ROOT / "launch-claimed-once.json").open("x", encoding="utf-8") as stream:
        json.dump({**expected, "claimedAtUnix": time.time(),
                   "reservationSha256": sha(ROOT / "reservation.json")}, stream)
        stream.flush()
        os.fsync(stream.fileno())
    sys.path.insert(0, str(STUDY / "runtime"))
    from process import bounded
    environment = {k: v for k, v in os.environ.items()
                   if k not in {"OPENAI_API_KEY", "CODEX_API_KEY", "PYTHONOPTIMIZE"}}
    # A tighter whole-tree watchdog also encloses draining and receipt writing.
    # The allocation allows outer <=60s and inner <=50s; reserve twelve seconds
    # for up to two five-second waits plus polling/final receipt overhead.
    receipt = bounded([str(Path(sys.executable).resolve()), str(ROOT / "classified-client.py")],
                      str(STUDY.parents[1]), ROOT / "process", 38,
                      env=environment, max_log_bytes=65536)
    print(json.dumps({"allocationId": ALLOCATION, "returnCode": receipt["returnCode"],
                      "wallSeconds": receipt["wallSeconds"], "stopReason": receipt["stopReason"]}))
    return receipt["returnCode"]


if __name__ == "__main__":
    raise SystemExit(main())
