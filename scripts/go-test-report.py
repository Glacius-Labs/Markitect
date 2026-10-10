#!/usr/bin/env python3
"""Summarize a `go test -json` event stream for CI logs and timing artifacts.

The script prints the captured output of failed tests and packages, then the
package and test timings. With --summary it also appends a Markdown table to
that file, for example $GITHUB_STEP_SUMMARY. It exits 1 when any package or
test failed, a package failed to build, or the stream holds no package result.
"""

import argparse
from dataclasses import dataclass, field
import json
from pathlib import Path
import sys


@dataclass
class Result:
    status: str = "running"
    elapsed: float = 0.0
    output: list[str] = field(default_factory=list)


@dataclass
class Report:
    packages: dict[str, Result] = field(default_factory=dict)
    tests: dict[tuple[str, str], Result] = field(default_factory=dict)
    build_output: dict[str, list[str]] = field(default_factory=dict)
    malformed: list[str] = field(default_factory=list)

    def failed_packages(self) -> list[str]:
        return sorted(name for name, result in self.packages.items() if result.status != "pass" and result.status != "skip")

    def failed_tests(self) -> list[tuple[str, str]]:
        # A test still running when its package stopped, for example on a timeout panic, never got a result.
        return sorted(key for key, result in self.tests.items() if result.status in ("fail", "running"))

    def tested_packages(self) -> list[tuple[str, Result]]:
        tested = [(name, result) for name, result in self.packages.items() if result.status != "skip"]
        return sorted(tested, key=lambda item: (-item[1].elapsed, item[0]))

    def passed(self) -> bool:
        return bool(self.packages) and not self.failed_packages() and not self.failed_tests()


def parse(lines) -> Report:
    report = Report()
    for line in lines:
        text = line.strip()
        if not text:
            continue
        try:
            event = json.loads(text)
        except json.JSONDecodeError:
            report.malformed.append(text)
            continue
        action = event.get("Action", "")
        if action in ("build-output", "build-fail"):
            target = report.build_output.setdefault(event.get("ImportPath", ""), [])
            if action == "build-output":
                target.append(event.get("Output", ""))
            continue
        package = event.get("Package", "")
        test = event.get("Test", "")
        result = report.tests.setdefault((package, test), Result()) if test else report.packages.setdefault(package, Result())
        if action == "output":
            result.output.append(event.get("Output", ""))
        elif action in ("pass", "fail", "skip"):
            result.status = action
            result.elapsed = float(event.get("Elapsed", 0.0))
    return report


def write_log(report: Report, out, slowest: int) -> None:
    for name, lines in sorted(report.build_output.items()):
        out.write(f"=== build output {name}\n{''.join(lines)}")
    for package, test in report.failed_tests():
        result = report.tests[(package, test)]
        label = "FAIL" if result.status == "fail" else "INTERRUPTED"
        out.write(f"=== {label} {package} {test}\n{''.join(result.output)}")
    for package in report.failed_packages():
        out.write(f"=== FAIL package {package}\n{''.join(report.packages[package].output)}")
    out.write("\nPackages by elapsed time:\n")
    for name, result in report.tested_packages():
        out.write(f"{result.status:>7} {result.elapsed:9.2f}s  {name}\n")
    out.write(f"\nSlowest {slowest} tests:\n")
    for (package, test), result in slowest_tests(report, slowest):
        out.write(f"{result.status:>7} {result.elapsed:9.2f}s  {package} {test}\n")
    passed = sum(1 for result in report.tests.values() if result.status == "pass")
    skipped = sum(1 for result in report.tests.values() if result.status == "skip")
    out.write(
        f"\n{len(report.packages)} packages, {len(report.failed_packages())} failed; "
        f"{passed} tests passed, {len(report.failed_tests())} failed, {skipped} skipped\n"
    )
    if report.malformed:
        out.write(f"{len(report.malformed)} lines were not JSON test events; first: {report.malformed[0]}\n")


def write_summary(report: Report, path: Path, title: str, slowest: int) -> None:
    lines = [f"### {title}", ""]
    lines.append("Passed." if report.passed() else "**Failed.**")
    failed = [f"`{package}` `{test}`" for package, test in report.failed_tests()] + [f"`{package}` (package)" for package in report.failed_packages()]
    if failed:
        lines += ["", "Failures: " + ", ".join(failed)]
    lines += ["", "| Package | Result | Seconds |", "|---|---|---:|"]
    for name, result in report.tested_packages():
        lines.append(f"| `{name}` | {result.status} | {result.elapsed:.1f} |")
    lines += ["", f"Slowest {slowest} tests:", "", "| Test | Package | Seconds |", "|---|---|---:|"]
    for (package, test), result in slowest_tests(report, slowest):
        lines.append(f"| `{test}` | `{package}` | {result.elapsed:.1f} |")
    with path.open("a", encoding="utf-8") as summary:
        summary.write("\n".join(lines) + "\n\n")


def slowest_tests(report: Report, count: int):
    top_level = [(key, result) for key, result in report.tests.items() if "/" not in key[1] and result.status != "running"]
    return sorted(top_level, key=lambda item: (-item[1].elapsed, item[0]))[:count]


def main(argv: list[str]) -> int:
    # Test output may hold any character; a redirected stream on Windows
    # would otherwise use the console code page and fail on it.
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(encoding="utf-8", errors="replace")
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("input", type=Path, help="go test -json output")
    parser.add_argument("--summary", type=Path, help="append a Markdown summary to this file")
    parser.add_argument("--title", default="Go tests", help="heading of the Markdown summary")
    parser.add_argument("--slowest", type=int, default=20, help="number of slowest tests to list")
    args = parser.parse_args(argv)
    with args.input.open(encoding="utf-8", errors="replace") as stream:
        report = parse(stream)
    write_log(report, sys.stdout, args.slowest)
    if args.summary is not None:
        write_summary(report, args.summary, args.title, args.slowest)
    if not report.packages:
        print("no package results found in the test event stream", file=sys.stderr)
    return 0 if report.passed() else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
