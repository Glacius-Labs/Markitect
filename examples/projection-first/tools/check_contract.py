from __future__ import annotations

import hashlib
import json
import re
import subprocess
import sys
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
IMPLEMENTATION_PATH = "scripts/list_open_listings.py"
CI_WORKFLOW_PATH = ".github/workflows/ci.yaml"
HOOK_PATH = ".githooks/pre-commit"


def read_json_yaml(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        try:
            label = str(path.relative_to(ROOT))
        except ValueError:
            label = str(path)
        raise ValueError(f"cannot read JSON-subset YAML {label}: {exc}") from exc


def load_fixture(root: Path = ROOT) -> tuple[dict[str, Any], dict[str, Any], dict[str, Any]]:
    view = read_json_yaml(root / "resources" / "list-open-listings.yaml")
    process = read_json_yaml(root / "resources" / "process.yaml")
    fixture_path = view.get("spec", {}).get("testCases")
    if view.get("kind") != "ListingView" or process.get("kind") != "DispatchProcess":
        raise ValueError("canonical intent resource kinds are incorrect")
    if not isinstance(fixture_path, list) or len(fixture_path) != 1:
        raise ValueError("ListingView must name exactly one fixed test fixture")
    fixture = read_json_yaml(root / fixture_path[0])
    if not isinstance(fixture.get("cases"), list) or not fixture["cases"]:
        raise ValueError("fixture must contain a non-empty cases array")
    return view, process, fixture


def expected_rows(view: dict[str, Any], items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    spec = view["spec"]
    if (
        spec.get("filterStatus") != "Open"
        or spec.get("sortFields") != ["openedAt", "id"]
        or spec.get("sortOrder") != "ascending"
        or spec.get("outputFormat") != "json-stdout"
    ):
        raise ValueError("checker supports only this finite ListingView contract")
    selected = []
    for row in items:
        if not isinstance(row, dict):
            raise ValueError("each input item must be a JSON object")
        item_id = row.get("id")
        opened = row.get("openedAt")
        if not isinstance(item_id, str) or not item_id:
            raise ValueError("every row must have a non-empty stable id")
        if not isinstance(opened, str) or not re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", opened):
            raise ValueError(f"row {item_id!r} has a non-canonical UTC timestamp")
        try:
            datetime.strptime(opened, "%Y-%m-%dT%H:%M:%SZ")
        except ValueError as exc:
            raise ValueError(f"row {item_id!r} has an invalid UTC timestamp") from exc
        if row.get("status") == spec["filterStatus"]:
            selected.append(row)
    ids = [row["id"] for row in selected]
    if len(ids) != len(set(ids)):
        raise ValueError("selected rows must have unique stable ids")
    return sorted(selected, key=lambda row: (row["openedAt"], row["id"]))


def _sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check_process_files(root: Path, process: dict[str, Any]) -> None:
    spec = process["spec"]
    targets = [IMPLEMENTATION_PATH, CI_WORKFLOW_PATH, HOOK_PATH]
    missing = [path for path in targets if not (root / path).is_file()]
    if missing:
        raise ValueError("missing materializer-owned paths: " + ", ".join(missing))
    workflow_text = (root / CI_WORKFLOW_PATH).read_text(encoding="utf-8")
    hook_text = (root / HOOK_PATH).read_text(encoding="utf-8")
    for command in spec["verificationCommands"]:
        if command not in workflow_text:
            raise ValueError(f"CI workflow does not contain declared command: {command}")
    if spec["preCommitCommand"] not in hook_text:
        raise ValueError("pre-commit hook does not contain the canonical checker command")


def run_candidate(root: Path, view: dict[str, Any], process: dict[str, Any], fixture: dict[str, Any]) -> None:
    view_path = root / "resources" / "list-open-listings.yaml"
    implementation = root / IMPLEMENTATION_PATH
    bound_paths = {
        "resources/list-open-listings.yaml",
        "resources/process.yaml",
        IMPLEMENTATION_PATH,
        CI_WORKFLOW_PATH,
        HOOK_PATH,
        *view["spec"]["testCases"],
    }
    before = {path: _sha(root / path) for path in sorted(bound_paths)}
    for case in fixture["cases"]:
        name = case.get("name", "<unnamed>")
        if not isinstance(case.get("items"), list):
            raise ValueError(f"case {name!r} must provide an items array")
        expected = {"items": expected_rows(view, case["items"])}
        with tempfile.TemporaryDirectory(prefix="projection-first-check-") as temp:
            work = Path(temp)
            proc = subprocess.run(
                [sys.executable, str(implementation), "--view", str(view_path)],
                input=json.dumps({"items": case["items"]}, separators=(",", ":")),
                text=True,
                encoding="utf-8",
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                cwd=work,
                timeout=10,
                check=False,
            )
            if proc.returncode != 0:
                raise ValueError(f"case {name!r}: implementation exit {proc.returncode}: {proc.stderr.strip()}")
            if proc.stderr:
                raise ValueError(f"case {name!r}: unexpected stderr: {proc.stderr.strip()}")
            try:
                actual = json.loads(proc.stdout)
            except json.JSONDecodeError as exc:
                raise ValueError(f"case {name!r}: stdout is not one JSON value: {exc}") from exc
            if actual != expected:
                raise ValueError(f"case {name!r}: output differs from the independent expected rows")
            if list(work.iterdir()):
                raise ValueError(f"case {name!r}: implementation wrote into its temporary working directory")
    after = {path: _sha(root / path) for path in sorted(bound_paths)}
    if before != after:
        raise ValueError("candidate changed a bound intent, fixture, implementation, workflow, or hook file")


def check(root: Path = ROOT) -> None:
    view, process, fixture = load_fixture(root)
    check_process_files(root, process)
    run_candidate(root, view, process, fixture)


def main() -> int:
    try:
        check()
    except Exception as exc:
        print(f"projection contract: failed: {exc}", file=sys.stderr)
        return 1
    print("projection contract: passed finite fixture")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
