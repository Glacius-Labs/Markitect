"""Command line entry point for an explicitly selected Playground adapter."""
from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path

from adapters.contract import AdapterError, validate_result
from adapters.loader import load_adapter


_TERMINAL = {"blocked", "completed", "failed", "cancelled", "uncertain", "needs_input"}


def _json(path, field):
    try:
        value = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise AdapterError(f"cannot read {field}: {exc}") from exc
    return value


def _emit(value):
    sys.stdout.write(json.dumps(value, ensure_ascii=False, sort_keys=True) + "\n")
    sys.stdout.flush()


def _run(adapter, action, args):
    if action == "describe":
        from adapters.contract import validate_descriptor
        return validate_descriptor(adapter.describe())
    if action == "setup":
        context = _json(args.context, "setup context") if args.context else {}
        if not isinstance(context, dict):
            raise AdapterError("setup context must be a JSON object")
        return validate_result(action, adapter.setup(context))
    if action == "ensure-runtime":
        return validate_result("ensure_runtime", adapter.ensure_runtime())
    if action == "start":
        accepted = validate_result(action, adapter.start(args.prompt))
        _emit(accepted)
        if accepted["state"] in _TERMINAL:
            return accepted
        run_id = accepted["runId"]
        while True:
            current = validate_result("status", adapter.status(run_id))
            if current["runId"] != run_id:
                raise AdapterError("status() returned a different runId; preserve evidence and do not replay")
            _emit(current)
            if current["state"] in _TERMINAL:
                return current
            time.sleep(1)
    if action == "resume":
        accepted = validate_result(action, adapter.resume(args.run_id, args.prompt))
        _emit(accepted)
        if accepted["state"] in _TERMINAL:
            return accepted
        while True:
            current = validate_result("status", adapter.status(args.run_id))
            if current["runId"] != args.run_id:
                raise AdapterError("status() returned a different runId; preserve evidence and do not replay")
            _emit(current)
            if current["state"] in _TERMINAL:
                return current
            time.sleep(1)
    if action in {"status", "cancel"}:
        method = getattr(adapter, action)
        result = validate_result(action, method(args.run_id))
        if result.get("runId") and result["runId"] != args.run_id:
            raise AdapterError(f"{action}() returned a different runId")
        return result
    if action == "close":
        return validate_result(action, adapter.close())
    raise AdapterError("unknown adapter command")


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--adapter", required=True, type=Path, help="explicit adapter manifest JSON")
    parser.add_argument("--config", required=True, type=Path, help="adapter configuration JSON")
    commands = parser.add_subparsers(dest="action", required=True)
    commands.add_parser("describe")
    setup = commands.add_parser("setup")
    setup.add_argument("--context", type=Path)
    commands.add_parser("ensure-runtime")
    for action in ("start", "resume"):
        command = commands.add_parser(action)
        if action == "resume":
            command.add_argument("run_id")
        command.add_argument("prompt")
    for action in ("status", "cancel"):
        commands.add_parser(action).add_argument("run_id")
    commands.add_parser("close")
    args = parser.parse_args(argv)
    adapter = None
    try:
        adapter = load_adapter(args.adapter, args.config)
        result = _run(adapter, args.action, args)
        if args.action not in {"start", "resume"}:
            _emit(result)
        return 0 if result.get("state") not in {"failed", "blocked", "uncertain", "needs_input"} else 1
    except (AdapterError, OSError, ValueError, KeyError) as exc:
        sys.stderr.write(str(exc) + "\n")
        return 2
if __name__ == "__main__":
    raise SystemExit(main())
