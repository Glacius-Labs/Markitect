"""Prepare deterministic starts and fresh cells; never copy the study into actors."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parent
ARMS = ("conventional", "classic", "government")
CONDITIONS = ("greenfield", "brownfield")


def git(repo, *args):
    env = dict(os.environ, GIT_AUTHOR_NAME="Study fixture", GIT_AUTHOR_EMAIL="fixture@example.invalid",
               GIT_COMMITTER_NAME="Study fixture", GIT_COMMITTER_EMAIL="fixture@example.invalid",
               GIT_AUTHOR_DATE="2026-10-07T00:00:00Z", GIT_COMMITTER_DATE="2026-10-07T00:00:00Z")
    return subprocess.check_output(["git", "-c", "core.autocrlf=false", "-c", "commit.gpgsign=false",
                                    "-c", "core.hooksPath=", "-C", str(repo), *args], env=env, text=True).strip()


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def prepare(destination):
    destination = Path(destination).resolve()
    destination.mkdir(parents=True, exist_ok=False)
    manifest = {"schemaVersion": 1, "kind": "fixture-preparation", "live": False,
                "starts": {}, "cells": []}
    for condition in CONDITIONS:
        start = destination / "starts" / condition
        start.mkdir(parents=True)
        if condition == "brownfield":
            for src in sorted((ROOT / "fixtures" / "brownfield").rglob("*")):
                relative = src.relative_to(ROOT / "fixtures" / "brownfield")
                if src.is_file() and not any(p in ("bin", "obj", ".git") for p in relative.parts):
                    target = start / relative
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copyfile(src, target)
        git(start, "init", "-q", "-b", "study/start")
        git(start, "config", "core.autocrlf", "false")
        git(start, "add", ".")
        git(start, "commit", "-q", "--allow-empty", "-m", f"Frozen {condition} reference fixture v1")
        sha = git(start, "rev-parse", "HEAD")
        files = git(start, "ls-files").splitlines()
        if condition == "greenfield" and files:
            raise RuntimeError("Greenfield must have an empty Git tree")
        manifest["starts"][condition] = {"commit": sha, "tree": git(start, "rev-parse", "HEAD^{tree}"),
                                         "files": {p: digest(start / p) for p in files}}
        for arm in ARMS:
            cell = destination / "cells" / f"{arm}-{condition}"
            cell.parent.mkdir(parents=True, exist_ok=True)
            subprocess.run(["git", "-c", "core.autocrlf=false", "clone", "-q", "--no-hardlinks", str(start), str(cell)], check=True)
            git(cell, "config", "core.autocrlf", "false")
            git(cell, "checkout", "-q", "-b", f"study/{arm}-{condition}")
            manifest["cells"].append({"arm": arm, "condition": condition, "baseCommit": sha,
                                      "repository": str(cell)})
    (destination / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    return manifest


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True, help="New absolute directory; never reused or deleted")
    args = parser.parse_args()
    if not Path(args.destination).is_absolute():
        parser.error("destination must be absolute")
    print(json.dumps(prepare(args.destination), indent=2))
