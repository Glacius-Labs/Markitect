#!/usr/bin/env python3
"""Run Relay's public project-owned validation commands."""
import subprocess
import sys

COMMANDS = [
    ["markitect", "check", "--repo", "."],
    ["markitect", "render", "--repo", ".", "--check"],
    ["markitect-check-modules", "--repo", ".", "--hooks", ".markitect/modules/githooks.config", "--pipelines", ".markitect/modules/pipelines.config"],
    ["markitect-check-artifacts", "--repo", ".", "--config", "markitect-artifacts.yaml"],
    ["go", "test", "./..."],
    ["go", "-C", "tools/process-sentinel", "test", "./..."],
]

for command in COMMANDS:
    completed = subprocess.run(command, check=False)
    if completed.returncode:
        raise SystemExit(completed.returncode)
