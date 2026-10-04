#!/usr/bin/env python3
"""Literal Relay operations checks; no provider execution or source analysis."""
from __future__ import annotations
import hashlib
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXPECTED_RUNS = {
    "Operations policy": "python scripts/check_operations.py",
    "Operations checker tests": "python -m unittest discover -s scripts",
    "Root module tests": "go test ./...",
    "Nested module tests": "go -C tools/process-sentinel test ./...",
}

def fail(message: str) -> None:
    raise SystemExit(f"operations-policy: {message}")

def inventory():
    config = (ROOT / "docs/engineering/managed-files.yaml").read_text(encoding="utf-8")
    owner_section = config.split("owners:\n", 1)[1].split("excluded:\n", 1)[0]
    owners = {}
    for line in owner_section.splitlines():
        match = re.fullmatch(r"  ([^:]+): (.+)", line)
        if match:
            owners[match.group(1)] = match.group(2)
    roots_section = config.split("roots:\n", 1)[1].split("owners:\n", 1)[0]
    roots = [line[4:].strip() for line in roots_section.splitlines() if line.startswith("  - ")]
    exclusions_section = config.split("excluded:\n", 1)[1]
    exclusion_records = re.findall(r"(?m)^  - path: ([^\n]+)\n    reason: ([^\n]+)$", exclusions_section)
    exclusions = {path.strip(): reason.strip() for path, reason in exclusion_records}
    return roots, owners, exclusions

def run() -> None:
    source = (ROOT / "docs/engineering/agent-rules.md").read_text(encoding="utf-8")
    expected = hashlib.sha256(source.encode()).hexdigest()
    for rel in ("AGENTS.md", ".claude/CLAUDE.md"):
        text = (ROOT / rel).read_text(encoding="utf-8")
        if f"rules-sha256: {expected}" not in text or source not in text:
            fail(f"{rel} is stale relative to docs/engineering/agent-rules.md")
    hook = (ROOT / ".githooks/pre-commit").read_bytes()
    declared = (ROOT / ".githooks/pre-commit.sha256").read_text(encoding="ascii").strip()
    if hashlib.sha256(hook).hexdigest() != declared:
        fail(".githooks/pre-commit digest is stale")
    workflow = (ROOT / ".github/workflows/ci.yaml").read_text(encoding="utf-8")
    for name, command in EXPECTED_RUNS.items():
        block = re.search(rf"- name: {re.escape(name)}\n\s+run: (.+)", workflow)
        if not block or block.group(1).strip() != command:
            fail(f"CI literal for {name!r} must equal {command!r}")
        if command.encode() not in hook:
            fail(f"pre-commit is missing {command!r}")
    roots, owners, exclusions = inventory()
    if not exclusions or any(not reason for reason in exclusions.values()):
        fail("each excluded path needs an explicit reason")
    if set(owners) & set(exclusions):
        fail("a path cannot be both owned and excluded")
    discovered = set()
    for root in roots:
        path = ROOT / root
        if path.is_file():
            discovered.add(root.replace("\\", "/"))
        elif path.is_dir():
            discovered.update(p.relative_to(ROOT).as_posix() for p in path.rglob("*") if p.is_file())
        else:
            fail(f"managed root is missing: {root}")
    discovered = {path for path in discovered if not any(path == prefix or path.startswith(prefix + "/") for prefix in exclusions)}
    missing = sorted(discovered - owners.keys())
    stale = sorted(owners.keys() - discovered)
    if missing or stale:
        fail(f"managed-path inventory mismatch; unowned={missing}, stale={stale}")

if __name__ == "__main__":
    run()
