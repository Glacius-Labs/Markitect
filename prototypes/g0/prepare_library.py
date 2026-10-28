#!/usr/bin/env python3
"""Assemble the Library M1 fixture exactly as the host variant's Library Apply does.

Copies testdata/library/m1, the vocabulary and Core into .markitect/, writes the
execution settings with the same ownership and project-tests check as
scripts/run-library-apply.ps1 (d4bc68ed), an empty registry, and commits on
dev/library-m1. Prints the base commit.

  python prepare_library.py MARKITECT_REPO TARGET_DIR
"""
import os
import shutil
import subprocess
import sys

EXECUTION = """apiVersion: markitect.execution/v2
ownership:
  - {path: ./, owner: .}
  - {path: realization/catalog/, owner: catalog}
  - {path: realization/lending/, owner: lending}
  - {path: realization/lending/reminders/, owner: lending/reminders}
  - {path: realization/application/, owner: library}
  - {path: realization/members/, owner: members}
  - {path: realization/cmd/library/, owner: library}
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
"""


def main(src, target):
    data = os.path.join(src, "testdata")
    mk = os.path.join(target, ".markitect")
    shutil.copytree(os.path.join(data, "library", "m1"), mk)
    shutil.copytree(os.path.join(data, "library", "vocabulary"), os.path.join(mk, "dependencies", "software"))
    shutil.copytree(os.path.join(data, "core"), os.path.join(mk, "dependencies", "core"))
    with open(os.path.join(mk, "execution.yaml"), "w", newline="\n") as f:
        f.write(EXECUTION)
    os.makedirs(os.path.join(mk, "registry"))
    with open(os.path.join(mk, "registry", "registry.yaml"), "w", newline="\n") as f:
        f.write("apiVersion: markitect.registry/v1\nartifacts: []\n")
    g = lambda *a: subprocess.run(["git", "-C", target, *a], check=True, capture_output=True, text=True).stdout.strip()
    g("init", "-q", "-b", "dev/library-m1")
    g("config", "core.autocrlf", "false")
    g("config", "user.name", "Markitect Library Fixture")
    g("config", "user.email", "fixture@markitect.invalid")
    g("add", "--all")
    g("commit", "-q", "-m", "Canonical Library M1 requirement baseline")
    print(g("rev-parse", "HEAD"))


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
