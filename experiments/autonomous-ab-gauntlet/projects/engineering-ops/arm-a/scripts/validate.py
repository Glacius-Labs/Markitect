#!/usr/bin/env python3
"""Run Relay's public project-owned validation commands."""
import subprocess
import sys

COMMANDS = [
    [sys.executable, "scripts/check_operations.py"],
    [sys.executable, "-m", "unittest", "discover", "-s", "scripts"],
    ["go", "test", "./..."],
    ["go", "-C", "tools/process-sentinel", "test", "./..."],
]

for command in COMMANDS:
    completed = subprocess.run(command, check=False)
    if completed.returncode:
        raise SystemExit(completed.returncode)
