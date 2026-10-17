"""Read-only source inventory; no application or study execution."""

import argparse
import hashlib
import json
from pathlib import PurePosixPath
import re
import subprocess
import sys


ROOT = "experiments/test-projects/comparison/playground/"
HEX40 = re.compile(r"[0-9a-f]{40}")
HEX64 = re.compile(r"[0-9a-f]{64}")


def path(value):
    if (not isinstance(value, str) or not value or "\\" in value
            or any(character in value for character in ("\0", "\n", "\r", ":"))
            or PurePosixPath(value).is_absolute()
            or any(part in ("", ".", "..") for part in value.split("/"))):
        raise ValueError("Expected an exact relative POSIX path")
    return value


def digest(data):
    return hashlib.sha256(data).hexdigest()


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("Duplicate manifest key: " + key)
        result[key] = value
    return result


class GitSources:
    def __init__(self, repo):
        self.repo = repo
        self.cache = {}
        self.commits = set()

    def read(self, revision, source):
        if not isinstance(revision, str) or not HEX40.fullmatch(revision):
            raise ValueError("Source revision must be a full lowercase commit SHA")
        source = path(source)
        key = (revision, source)
        if key not in self.cache:
            def git(*args):
                return subprocess.run(["git", "-C", str(self.repo), *args],
                                      check=True, stdout=subprocess.PIPE,
                                      stderr=subprocess.PIPE).stdout
            if revision not in self.commits:
                if git("cat-file", "-t", revision).strip() != b"commit":
                    raise ValueError("Source SHA must identify a commit")
                self.commits.add(revision)
            tree = git("ls-tree", "-z", revision, "--", source)
            metadata, separator, observed = tree.rstrip(b"\0").partition(b"\t")
            if (not separator or observed.decode("utf-8") != source
                    or not metadata.startswith(b"100644 blob ")):
                raise ValueError("Expected one ordinary Git file: " + source)
            self.cache[key] = git("show", revision + ":" + source)
        return self.cache[key]


def inspect(sources, revision, scenario_id):
    def manifest(name):
        return json.loads(sources.read(revision, ROOT + name),
                          object_pairs_hook=unique_object)

    scenarios = manifest("scenarios.yaml")
    factors = manifest("factors.yaml")
    if scenarios.get("formatVersion") != 1 or factors.get("formatVersion") != 1:
        raise ValueError("Unsupported source manifest version")
    selected = [item for item in scenarios["scenarios"] if item["id"] == scenario_id]
    if len(selected) != 1:
        raise ValueError("Expected exactly one declared scenario: " + scenario_id)
    selected = selected[0]
    sets = scenarios["sourceSets"]

    def resolve(name, ancestors=()):
        if name in ancestors:
            raise ValueError("Cyclic source-set reference")
        item = sets[name]
        files, patches, blockers = [], [], []
        if "base" in item:
            files, patches, blockers = resolve(item["base"], ancestors + (name,))
        return (files + item.get("files", []), patches + item.get("patches", []),
                blockers + item.get("sourceBlockers", []))

    files, patches, blockers = resolve(selected["sourceSet"])
    arms = {"ordinary": {}, "markitect": {}}
    for entry in files:
        target = path(entry["target"])
        expected = entry.get("sha256")
        if expected is None or entry.get("revision") is None:
            blockers.append("UNBOUND source: " + entry["source"])
            continue
        if not HEX64.fullmatch(expected):
            raise ValueError("Expected SHA-256: " + entry["source"])
        source_revision = revision if entry["revision"] == "packet" else entry["revision"]
        content = sources.read(source_revision, entry["source"])
        if digest(content) != expected:
            raise ValueError("Source hash mismatch: " + entry["source"])
        for arm in entry["arms"]:
            if arm not in arms or target in arms[arm]:
                raise ValueError("Invalid arm or duplicate target: " + target)
            arms[arm][target] = content
    for patch in patches:
        target = path(patch["target"])
        before = patch["before"].encode("utf-8")
        after = patch["after"].encode("utf-8")
        for arm in patch["arms"]:
            content = arms[arm][target]
            if digest(content) != patch["beforeSha256"] or not before or content.count(before) != 1:
                raise ValueError("Byte-patch precondition failed: " + target)
            updated = content.replace(before, after, 1)
            if digest(updated) != patch["afterSha256"]:
                raise ValueError("Byte-patch result mismatch: " + target)
            arms[arm][target] = updated
    common = set(arms["ordinary"]) & set(arms["markitect"])
    if any(arms["ordinary"][name] != arms["markitect"][name] for name in common):
        raise ValueError("Shared application bytes differ between arms")
    required = {"ordinary": ["requirements.md", "implementation-prompt.md"],
                "markitect": [".markitect/model.yaml", ".markitect/model-lock.yaml"]}
    for arm, targets in required.items():
        for target in targets + selected.get("applicationTargets", []):
            if target not in arms[arm]:
                blockers.append("UNBOUND required target: " + arm + "/" + target)
    profile = factors["profiles"][selected["profile"]]
    unbound = ["UNBOUND " + name for name, value in profile["actualBindings"].items()
               if value is None]
    unbound.append("Runtime and independent assessment readiness are outside this source preview")
    return {
        "scenario": scenario_id,
        "packetRevision": revision,
        "sourcePrepared": not blockers,
        "sourceStatus": "UNBOUND" if blockers else "SOURCE_PREPARED",
        "sourceBlockers": blockers,
        "runReady": False,
        "runBlockers": unbound,
        "knownBaselineGaps": selected.get("knownBaselineGaps", []),
        "arms": {arm: [{"target": name, "mode": "100644", "sha256": digest(content)}
                       for name, content in sorted(inventory.items())]
                 for arm, inventory in arms.items()},
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".")
    parser.add_argument("--revision", required=True)
    parser.add_argument("--scenario", required=True)
    parser.add_argument("--require-sources", action="store_true")
    parser.add_argument("--require-ready", action="store_true")
    args = parser.parse_args(argv)
    try:
        result = inspect(GitSources(args.repo), args.revision, args.scenario)
        print(json.dumps(result, indent=2, ensure_ascii=False))
        return 3 if args.require_ready or (args.require_sources and not result["sourcePrepared"]) else 0
    except (ValueError, KeyError, TypeError, OSError, subprocess.CalledProcessError) as error:
        print("Source preview failed: " + str(error), file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
