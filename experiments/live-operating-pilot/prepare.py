#!/usr/bin/env python3
"""Clone the frozen source bundle into a new external directory and select one revision."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

BASELINE = "9dc9e68c182bccd408690f9f20300a75c68bd35a"
CHANGED = "bd06f90c9afcc80f53f5c70d355b39266b6c4331"


def run(args):
    subprocess.run(args, check=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--destination", required=True, help="Absolute path to a destination directory that does not exist")
    parser.add_argument("--revision", choices=("baseline", "changed"), default="baseline")
    args = parser.parse_args()

    package = Path(__file__).resolve().parent
    repo = package.parent.parent
    destination = Path(args.destination).expanduser()
    if not destination.is_absolute():
        raise SystemExit("--destination must be an absolute external path")
    destination = destination.resolve(strict=False)
    if destination.exists():
        raise SystemExit(f"refusing existing destination: {destination}")
    if not destination.parent.is_dir():
        raise SystemExit(f"destination parent must already exist: {destination.parent}")
    if destination == repo or destination.is_relative_to(repo) or repo.is_relative_to(destination):
        raise SystemExit("destination must be outside the Markitect checkout")
    revision = BASELINE if args.revision == "baseline" else CHANGED
    bundle = package / "source.bundle"

    # The bundle is a local input. Clone without checkout, set local line-ending
    # behavior, then materialize the exact frozen commit bytes.
    run(["git", "clone", "--no-checkout", str(bundle), str(destination)])
    run(["git", "-C", str(destination), "config", "core.autocrlf", "false"])
    run(["git", "-C", str(destination), "switch", "--create", "codex/pilot-" + args.revision, revision])
    actual = subprocess.check_output(
        ["git", "-C", str(destination), "rev-parse", "HEAD"], text=True
    ).strip()
    if actual != revision:
        raise SystemExit(f"checkout mismatch: expected {revision}, got {actual}")
    print(json.dumps({
        "destination": str(destination),
        "revision": args.revision,
        "commit": actual,
        "core.autocrlf": subprocess.check_output(
            ["git", "-C", str(destination), "config", "--local", "core.autocrlf"], text=True
        ).strip(),
    }, indent=2))


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as exc:
        print(f"command failed with exit code {exc.returncode}: {exc.cmd}", file=sys.stderr)
        raise SystemExit(exc.returncode)