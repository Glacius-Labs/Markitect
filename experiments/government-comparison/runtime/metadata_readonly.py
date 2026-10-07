"""Pinned account/read + model/list only; no Actor, login, thread or turn RPC."""
import json
import os
from pathlib import Path
import sys
import time

from dispatch import digest, encoded, external, validate_listing
from identity_probe import events
from process import bounded
import runner


def collect(destination, executable):
    root = external(destination)
    root.mkdir(parents=True, exist_ok=False)
    pin = runner.inspect(executable)  # Hash file bytes; does not invoke a model.
    request = {"argv": runner.metadata_argv(executable),
               "initialize": {"method": "initialize", "id": 0, "params": {
                   "clientInfo": {"name": "scientist_metadata_only", "version": "2.0"}}},
               "allowedRpc": ["initialize", "initialized", "account/read(refreshToken=false)", "model/list"],
               "actorStarts": 0, "inferenceCalls": 0, "runnerPin": pin}
    request_path = root / "request.json"
    request_path.write_bytes(encoded(request) + b"\n")
    env = {key: val for key, val in os.environ.items() if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
    receipt = bounded([str(Path(sys.executable).resolve()), str(Path(__file__).with_name("selected_metadata.py")), str(request_path)],
                      str(root), root / "process", 45, env=env, stop_path=root / "STOP")
    replies = {v["id"]: v for v in events(root / "process/stdout.log") if "id" in v}
    account = replies.get(1, {}).get("result", {}).get("account")
    errors = [v["error"] for v in replies.values() if "error" in v]
    records = [r for key, value in replies.items() if isinstance(key, int) and key >= 2
               for r in value.get("result", {}).get("data", [])]
    matches = [r for r in records if r.get("model") == runner.MODEL and r.get("id") == runner.MODEL]
    high = any(e.get("reasoningEffort") == runner.REASONING for r in matches for e in r.get("supportedReasoningEfforts", []))
    complete = any(isinstance(key, int) and key >= 2 and value.get("result", {}).get("nextCursor") is None
                   and "result" in value for key, value in replies.items())
    result = {"status": "advertised" if matches and high and account and complete and not errors else "unknown",
              "accountType": account.get("type") if isinstance(account, dict) else None,
              "targetEntries": matches, "requestedHighAdvertised": high, "rpcErrors": errors,
              "processReturnCode": receipt["returnCode"], "processStopReason": receipt["stopReason"],
              "requestSha256": digest(request_path.read_bytes()), "observedAtUnix": time.time(),
              "actorStarts": 0, "inferenceCalls": 0, "resolvedServingModel": None,
              "scope": "Authenticated read-only metadata; catalog may be cached; no serving identity or inference proof",
              "sourceSha256": {name: digest(Path(__file__).with_name(name).read_bytes())
                               for name in ("metadata_readonly.py", "selected_metadata.py", "process.py", "runner.py")}}
    path = root / "result.json"
    path.write_bytes(encoded(result) + b"\n")
    validate_listing({"path": str(path), "sha256": digest(path.read_bytes())})
    return {"path": str(path), "sha256": digest(path.read_bytes())}
