"""CLI and MCP composition root for the optional Conventional controller."""
import argparse
import json
from pathlib import Path
import sys

from conventional.service import Service


def main():
    # MCP stdio is UTF-8 regardless of the Windows console code page.
    for stream in (sys.stdin, sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(encoding="utf-8")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", required=True, type=Path)
    commands = parser.add_subparsers(dest="action", required=True)
    for action in ("start", "resume"):
        command = commands.add_parser(action)
        if action == "resume":
            command.add_argument("run_id")
        command.add_argument("--prompt-file", required=True, type=Path)
    for action in ("status", "cancel"):
        commands.add_parser(action).add_argument("run_id")
    commands.add_parser("mcp")
    args = parser.parse_args()
    service = None
    try:
        service = Service(args.config)
        if args.action == "mcp":
            from conventional.mcp import serve
            serve(service)
        else:
            if args.action in {"start", "resume"}:
                prompt = args.prompt_file.read_text(encoding="utf-8")
                result = service.start(prompt) if args.action == "start" else service.resume(args.run_id, prompt)
                # Publish durable handle before waiting; Ctrl+C cancels owned dispatch.
                print(json.dumps(result, ensure_ascii=False), flush=True)
                result = service.wait(result["runId"])
            else:
                result = getattr(service, args.action)(args.run_id)
            print(json.dumps(result, ensure_ascii=False), flush=True)
            return 0 if result["state"] == "completed" else 1
    except (ValueError, OSError, KeyError) as exc:
        print(str(exc), file=sys.stderr)
        return 2
    except KeyboardInterrupt:
        print("Controller interrupted; inspect durable status before any continuation.", file=sys.stderr)
        return 130
    finally:
        if service:
            service.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
