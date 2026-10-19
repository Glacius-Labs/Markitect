"""Approximate write attribution from Execute transcripts: Write/Edit file paths and
paths under /work that Bash commands write, move or delete. Session/tool evidence
only, not complete attribution. Usage: writes.py STATE_DIR..."""
import json
import re
import sys
from collections import defaultdict
from pathlib import Path

WRITE_OPS = re.compile(r"(cat\s*>|>\s*|tee\s|mv\s|cp\s|rm\s|mkdir\s|sed\s+-i|gofmt\s+-w|touch\s)")
PATH = re.compile(r"(/work/[A-Za-z0-9_./-]+|(?<![\w/])(?:realization|\.markitect/registry)/[A-Za-z0-9_./-]+)")


def commands(transcript):
    for line in transcript.read_bytes().split(b"\n"):
        if not line.strip():
            continue
        d = json.loads(line)
        if d.get("kind") != "provider":
            continue
        m = d["data"] if isinstance(d["data"], dict) else json.loads(d["data"])
        if m.get("type") != "assistant":
            continue
        for b in m["message"].get("content", []):
            if b.get("type") == "tool_use":
                yield b.get("name"), b.get("input") or {}


for state in sys.argv[1:]:
    run = sorted((Path(state) / "runs").iterdir())[-1]
    print("#####", state)
    touched = defaultdict(set)
    for t in sorted((run / "transcripts").glob("*-x*.jsonl")):
        manager = re.sub(r"^.*-r\d+-(.*)-x\d+\.jsonl$", r"\1", t.name)
        for name, inp in commands(t):
            if name in ("Write", "Edit", "MultiEdit") and inp.get("file_path"):
                touched[manager].add(inp["file_path"].replace("/work/", ""))
            elif name == "Bash":
                cmd = inp.get("command", "")
                for segment in re.split(r"&&|;|\n|\|", cmd):
                    if WRITE_OPS.search(segment) and "/tmp" not in segment.split(">")[-1][:6]:
                        for p in PATH.findall(segment):
                            p = p.replace("/work/", "").rstrip(".")
                            if p and not p.startswith("tmp"):
                                touched[manager].add(p)
    for manager in sorted(touched):
        dirs = sorted({p if p.count("/") < 2 else "/".join(p.split("/")[:3]) for p in touched[manager]})
        print(f"{manager}: {', '.join(dirs)}")
