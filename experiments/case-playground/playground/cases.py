"""Cases are discovered from their folders; there is no case list in the code.

A case is a folder `cases/<name>/` (not `common`) whose name matches the manifest id
pattern and which holds `README.md`, `BACKLOG.md`, a valid `STATIONS.json` naming the
case, and its public checks `checks/<name>.py`. `evaluation/<name>/` (hidden files) is
optional. An invalid folder is an error that names the folder and the rule; nothing is
skipped silently. Files directly in `cases/` (the task prompt) are not cases.
"""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path

from .lifecycle import LifecycleError, load_station_plan

ROOT = Path(__file__).resolve().parent.parent  # the playground; /in inside the run container
NAME = re.compile(r"[a-z0-9][a-z0-9-]{0,62}")  # the manifest id pattern
# `common` is seeded into every case; `checks/acceptance.py` is the shared driver.
RESERVED = ("common", "acceptance")


class CaseError(ValueError):
    """A folder in cases/ breaks a rule, or a requested case does not exist."""


@dataclass(frozen=True)
class Case:
    name: str
    folder: Path
    stations: int  # waves in STATIONS.json
    checks: Path  # cases/<name>/checks/<name>.py
    evaluation: Path | None  # evaluation/<name>/ when present


def discover(root: Path | None = None) -> dict[str, Case]:
    """Every case under `<root>/cases`, sorted by name; raises CaseError for an invalid folder."""
    root = Path(root or ROOT)
    folder = root / "cases"
    if not folder.is_dir():
        raise CaseError(f"no cases folder: {folder}")
    found: dict[str, Case] = {}
    for path in sorted(folder.iterdir()):
        if path.name == "common" or not path.is_dir():
            continue
        found[path.name] = _case(root, path)
    return found


def get(name: str, root: Path | None = None) -> Case:
    """One case by name; CaseError names the known cases when it does not exist."""
    known = discover(root)
    if name not in known:
        raise CaseError(f"unknown case {name!r}; cases are the folders in {Path(root or ROOT) / 'cases'}: "
                        f"{', '.join(known) or 'none'}")
    return known[name]


def _case(root: Path, path: Path) -> Case:
    name, where = path.name, f"cases/{path.name}"
    if not NAME.fullmatch(name):
        raise CaseError(f"{where}: a case folder name must match {NAME.pattern}")
    if name in RESERVED:
        raise CaseError(f"{where}: the name {name!r} is reserved")
    if name.replace("-", "_") in sys.stdlib_module_names:  # checks/<name>.py would shadow it in the seed
        raise CaseError(f"{where}: the name {name!r} is a Python standard-library module")
    for required in ("README.md", "BACKLOG.md", "STATIONS.json"):
        if not (path / required).is_file():
            raise CaseError(f"{where}: {required} is missing")
    try:
        plan = load_station_plan(path / "STATIONS.json", name)
    except LifecycleError as exc:
        raise CaseError(f"{where}: {exc}") from exc
    checks = path / "checks" / f"{name}.py"
    if not checks.is_file():
        raise CaseError(f"{where}: its public checks checks/{name}.py are missing")
    evaluation = root / "evaluation" / name
    return Case(name=name, folder=path, stations=len(plan), checks=checks,
                evaluation=evaluation if evaluation.is_dir() else None)
