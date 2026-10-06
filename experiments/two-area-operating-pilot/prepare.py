#!/usr/bin/env python3
"""Prepare an exact baseline or changed fixture from a frozen pilot attempt."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

def run(argv):
    subprocess.run(argv, check=True)

def main():
    package = Path(__file__).resolve().parent
    repo = package.parent.parent
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--attempt", choices=("1", "2", "3"), default="3")
    parser.add_argument("--destination", required=True, help="Absolute path to a new external directory")
    parser.add_argument("--revision", choices=("baseline", "changed"), default="baseline")
    args = parser.parse_args()
    manifest_name = {"1": "fixture-manifest.json", "2": "fixture-manifest-attempt-02.json", "3": "fixture-manifest-attempt-03.json"}[args.attempt]
    manifest = json.loads((package / manifest_name).read_text(encoding="utf-8"))
    destination = Path(args.destination).expanduser()
    if not destination.is_absolute():
        raise SystemExit("--destination must be absolute")
    destination = destination.resolve(strict=False)
    if destination.exists():
        raise SystemExit(f"refusing existing destination: {destination}")
    if not destination.parent.is_dir():
        raise SystemExit(f"destination parent must already exist: {destination.parent}")
    if destination == repo or destination.is_relative_to(repo) or repo.is_relative_to(destination):
        raise SystemExit("destination must be outside the Markitect checkout")
    commit = manifest["baseCommit"] if args.revision == "baseline" else manifest["changedCommit"]
    branch = f"codex/prepared-two-area-{args.attempt}-{args.revision}"
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise SystemExit("manifest commit is not a full lowercase Git SHA")
    if not re.fullmatch(r"codex/prepared-two-area-[123]-(baseline|changed)", branch):
        raise SystemExit("refusing a non-feature or unexpected branch name")
    bundle_path = manifest["sourceBundle"]["path"]
    if not isinstance(bundle_path, str) or not bundle_path:
        raise SystemExit("manifest source bundle path must be a nonempty relative path")
    relative_bundle = Path(bundle_path)
    if relative_bundle.is_absolute():
        raise SystemExit("manifest source bundle path must stay inside the package")
    package_root = package.resolve(strict=True)
    try:
        bundle = (package_root / relative_bundle).resolve(strict=True)
        bundle.relative_to(package_root)
    except (OSError, ValueError):
        raise SystemExit("manifest source bundle path escapes the package or does not exist")
    if not bundle.is_file():
        raise SystemExit("manifest source bundle path is not a file")
    expected_digest = manifest["sourceBundle"].get("sha256")
    if not isinstance(expected_digest, str):
        raise SystemExit("manifest source bundle SHA-256 is missing")
    expected_digest = expected_digest.removeprefix("sha256:").lower()
    if not re.fullmatch(r"[0-9a-f]{64}", expected_digest):
        raise SystemExit("manifest source bundle SHA-256 is malformed")
    hasher = hashlib.sha256()
    with bundle.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            hasher.update(block)
    actual_digest = hasher.hexdigest()
    if actual_digest != expected_digest:
        raise SystemExit(f"source bundle digest mismatch: expected {expected_digest}, got {actual_digest}")
    run(["git", "clone", "--no-checkout", str(bundle), str(destination)])
    run(["git", "-C", str(destination), "config", "--local", "core.autocrlf", "false"])
    run(["git", "-C", str(destination), "switch", "--create", branch, commit])
    actual = subprocess.check_output(
        ["git", "-C", str(destination), "rev-parse", "HEAD"], text=True
    ).strip()
    autocrlf = subprocess.check_output(
        ["git", "-C", str(destination), "config", "--local", "core.autocrlf"], text=True
    ).strip()
    if actual != commit or autocrlf.lower() != "false":
        raise SystemExit(f"fixture verification failed: commit={actual}, core.autocrlf={autocrlf!r}")
    print(json.dumps({
        "attempt": int(args.attempt),
        "destination": str(destination),
        "revision": args.revision,
        "branch": branch,
        "commit": actual,
        "core.autocrlf": autocrlf,
    }, indent=2))
    return 0

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except subprocess.CalledProcessError as exc:
        print(f"command failed ({exc.returncode}): {exc.cmd}", file=sys.stderr)
        raise SystemExit(exc.returncode)