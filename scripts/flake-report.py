#!/usr/bin/env python3
"""Report flaky and failing Go tests across CI runs from their test results.

Input is the list of CI runs (`gh run list --json ...`) and a directory with
the downloaded `go-test-<os>-<sha>[-<attempt>]` artifacts of those runs, one
folder per run as `gh run download --dir <dir>/<run id>` creates them. A test
is flaky when it both passed and failed on the same commit and runner OS. The
report never retries anything; it only reads what the runs recorded.
"""

import argparse
from collections import defaultdict
from dataclasses import dataclass, field
import importlib.util
import json
from pathlib import Path
import re
import sys

_SPEC = importlib.util.spec_from_file_location("go_test_report", Path(__file__).with_name("go-test-report.py"))
assert _SPEC is not None and _SPEC.loader is not None
go_test_report = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(go_test_report)

ARTIFACT = re.compile(r"^go-test-(?P<os>.+)-(?P<sha>[0-9a-f]{40})(?:-(?P<attempt>\d+))?$")
PACKAGE = "(package)"


@dataclass
class Outcomes:
    """Outcomes of one test, keyed by (commit, OS)."""

    runs: dict[tuple[str, str], list[str]] = field(default_factory=lambda: defaultdict(list))
    last_failure: str = ""

    def record(self, sha: str, os_name: str, status: str, run: str) -> None:
        self.runs[(sha, os_name)].append(status)
        if status == "fail":
            self.last_failure = max(self.last_failure, run)

    def failures(self) -> int:
        return sum(statuses.count("fail") for statuses in self.runs.values())

    def observations(self) -> int:
        return sum(len(statuses) for statuses in self.runs.values())

    def flaky_on(self) -> list[tuple[str, str]]:
        return sorted(key for key, statuses in self.runs.items() if "pass" in statuses and "fail" in statuses)


@dataclass
class Collection:
    tests: dict[tuple[str, str, str], Outcomes] = field(default_factory=lambda: defaultdict(Outcomes))
    files: int = 0
    runs: set[str] = field(default_factory=set)
    skipped: list[str] = field(default_factory=list)


def collect(runs: list[dict], results: Path) -> Collection:
    known = {str(run["databaseId"]): run for run in runs}
    collection = Collection()
    for run_dir in sorted(path for path in results.iterdir() if path.is_dir()):
        if run_dir.name not in known:
            collection.skipped.append(f"{run_dir.name}: not in the run list")
            continue
        for artifact in sorted(path for path in run_dir.iterdir() if path.is_dir()):
            match = ARTIFACT.match(artifact.name)
            stream = artifact / "go-test.json"
            if not match or not stream.is_file():
                collection.skipped.append(f"{run_dir.name}/{artifact.name}: no go-test.json")
                continue
            with stream.open(encoding="utf-8", errors="replace") as lines:
                report = go_test_report.parse(lines)
            collection.files += 1
            collection.runs.add(run_dir.name)
            run_label = f"{run_dir.name}#{match['attempt'] or '1'}"
            for (package, test), result in report.tests.items():
                status = "fail" if result.status in ("fail", "running") else result.status
                if status in ("pass", "fail"):
                    collection.tests[(package, test, match["os"])].record(match["sha"], match["os"], status, run_label)
            for package, result in report.packages.items():
                if result.status == "skip":
                    continue
                status = "pass" if result.status == "pass" else "fail"
                collection.tests[(package, PACKAGE, match["os"])].record(match["sha"], match["os"], status, run_label)
    return collection


def render(collection: Collection, runs: list[dict], limit: int) -> str:
    dates = sorted(run.get("createdAt", "") for run in runs if str(run["databaseId"]) in collection.runs)
    lines = ["# Flake report", ""]
    span = f"{dates[0][:10]} to {dates[-1][:10]}" if dates else "no dates"
    lines.append(f"{len(collection.runs)} CI runs with test results ({span}), {collection.files} result files. "
                 "A test is flaky when it passed and failed on the same commit and OS.")
    flaky = [(key, outcomes) for key, outcomes in collection.tests.items() if outcomes.flaky_on()]
    flaky.sort(key=lambda item: (-len(item[1].flaky_on()), -item[1].failures(), item[0]))
    lines += ["", f"## Flaky tests ({len(flaky)})", ""]
    if flaky:
        lines += ["| Test | Package | OS | Commits | Failures / runs | Last failure |", "|---|---|---|---:|---:|---|"]
        for (package, test, os_name), outcomes in flaky[:limit]:
            lines.append(f"| `{test}` | `{package}` | {os_name} | {len(outcomes.flaky_on())} | "
                         f"{outcomes.failures()} / {outcomes.observations()} | {outcomes.last_failure} |")
    else:
        lines.append("None.")
    failing = [(key, outcomes) for key, outcomes in collection.tests.items() if outcomes.failures()]
    failing.sort(key=lambda item: (-item[1].failures(), item[0]))
    lines += ["", f"## All failing tests ({len(failing)})", ""]
    if failing:
        lines += ["| Test | Package | OS | Failures / runs | Last failure |", "|---|---|---|---:|---|"]
        for (package, test, os_name), outcomes in failing[:limit]:
            lines.append(f"| `{test}` | `{package}` | {os_name} | {outcomes.failures()} / {outcomes.observations()} | "
                         f"{outcomes.last_failure} |")
    else:
        lines.append("None.")
    if len(flaky) > limit or len(failing) > limit:
        lines += ["", f"Tables show the first {limit} rows."]
    if collection.skipped:
        lines += ["", f"Skipped {len(collection.skipped)} downloads without readable results, for example: "
                  + "; ".join(collection.skipped[:3])]
    return "\n".join(lines) + "\n"


def main(argv: list[str]) -> int:
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(encoding="utf-8", errors="replace")
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--runs", type=Path, required=True, help="JSON list from gh run list (databaseId, createdAt)")
    parser.add_argument("--results", type=Path, required=True, help="directory with one folder of artifacts per run")
    parser.add_argument("--summary", type=Path, help="also append the report to this file")
    parser.add_argument("--limit", type=int, default=30, help="rows per table")
    args = parser.parse_args(argv)
    runs = json.loads(args.runs.read_text(encoding="utf-8"))
    text = render(collect(runs, args.results), runs, args.limit)
    sys.stdout.write(text)
    if args.summary is not None:
        with args.summary.open("a", encoding="utf-8") as summary:
            summary.write(text + "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
