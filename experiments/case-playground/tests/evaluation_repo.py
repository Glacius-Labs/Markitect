"""Test helper: a temporary Git repository whose `evaluation/` folder is committed, and a
real git runner for registration.py that the Docker fakes (which replace subprocess.run)
do not intercept."""
from __future__ import annotations

import json
import os
import shutil
import subprocess
from pathlib import Path
from unittest import mock

from playground import registration

PLAYGROUND = Path(__file__).resolve().parents[1]
REAL_POPEN = subprocess.Popen  # captured before any test replaces subprocess.run and Popen
ENV = {**os.environ, "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1", "GIT_AUTHOR_NAME": "Test",
       "GIT_AUTHOR_EMAIL": "test@example.invalid", "GIT_COMMITTER_NAME": "Test",
       "GIT_COMMITTER_EMAIL": "test@example.invalid"}


def _real_git(repo: Path, args) -> subprocess.CompletedProcess:
    command = ["git", "-C", str(repo), *args]
    with REAL_POPEN(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, stdin=subprocess.DEVNULL,
                    env=ENV) as process:
        out, err = process.communicate(timeout=120)
    return subprocess.CompletedProcess(command, process.returncode, out, err)


def git(repo: Path, *args: str) -> str:
    done = _real_git(repo, args)
    if done.returncode:
        raise AssertionError(f"git {' '.join(args)}: {done.stderr.decode('utf-8', errors='replace')}")
    return done.stdout.decode("utf-8").strip()


def runner(repo: Path) -> registration.Runner:
    """Real git in `repo`, without the user's Git configuration."""
    return lambda args: _real_git(repo, args)


def use(repo: Path | None = None):
    """Patch registration.git_in: real git in `repo` (default: the root it is asked for)."""
    return mock.patch.object(registration, "git_in", lambda root: runner(repo or root))


def committed(root: Path, *, config: dict | None = None, case: str | None = None) -> Path:
    """`root` as a Git repository whose evaluation/ holds the playground's common/ and
    config.json (or `config`) and, with `case`, a holdout and ground truth; committed."""
    evaluation = root / "evaluation"
    shutil.copytree(PLAYGROUND / "evaluation" / "common", evaluation / "common",
                    ignore=shutil.ignore_patterns("__pycache__"))
    if config is None:
        config = json.loads((PLAYGROUND / "evaluation" / "config.json").read_text(encoding="utf-8"))
    (evaluation / "config.json").write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8", newline="\n")
    if case:
        (evaluation / case / "reference").mkdir(parents=True)
        (evaluation / case / "holdout.py").write_text("print('holdout')\n", encoding="utf-8", newline="\n")
        (evaluation / case / "ground-truth.json").write_text('{"case": "%s"}\n' % case, encoding="utf-8",
                                                             newline="\n")
        (evaluation / case / "validate.py").write_text("", encoding="utf-8", newline="\n")
        (evaluation / case / "reference" / "SECRET.md").write_text("reference\n", encoding="utf-8", newline="\n")
    git(root, "init", "-q")
    git(root, "add", "-A")
    git(root, "commit", "-q", "-m", "Register the evaluation files")
    return root
