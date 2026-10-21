"""Freeze the inputs of one Model assessment series before any provider work.

Usage: freeze.py SPEC FROZEN_DIR

SPEC (JSON) names the built candidate and the four controls:
  {"build": BUILD_DIR, "kit": SOURCE_EXPERIMENTS_DIR,
   "controls": [{"id": ..., "packet": {"materialize": [ARGV...]} | {"repo": PATH, "head": SHA},
                 "expectation": {"expected": ..., "issue": {...}, "freedom": {...}, "source": {...}}}]}

`materialize` runs with the frozen kit as working directory and the packet
directory appended as its last argument. Every packet must pass `markitect
check` (the Model check and the strict File check) with the build's CLI, or the
freeze is refused. The expectations go to FROZEN_DIR/controls, never into a
packet; the manifest binds every hash. Nothing here starts an Agent.
"""
import hashlib
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

EXPECTED = {"accepted", "needs_clarification", "model_error"}
CEILINGS = {"attempts": 12, "summedSeconds": 90 * 60, "attemptSeconds": 10 * 60, "model": "claude-sonnet-5-5", "effort": "high"}


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def git(repo, *args):
    return subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, text=True).stdout.strip()


def definitions(packet):
    """Definition identities of the packet's local model: package path and id per file."""
    root = packet / ".markitect" / "model"
    out = []
    for path in sorted(root.rglob("*.yaml")):
        match = re.search(r"^id:\s*(\S+)\s*$", path.read_text(encoding="utf-8"), re.MULTILINE)
        if match:
            package = path.parent.relative_to(root).as_posix()
            out.append((package if package != "." else ".") + "/" + match.group(1))
    return out


def check(build, packet):
    args = [str(build / "markitect"), "check", "--repo", str(packet), "--model", ".markitect/model.yaml"]
    for dep in sorted((packet / ".markitect" / "dependencies").glob("*/model.yaml")):
        args += ["--dependency", dep.relative_to(packet).as_posix()]
    done = subprocess.run(args, capture_output=True, text=True)
    try:
        status = json.loads(done.stdout).get("status")
    except json.JSONDecodeError:
        status = None
    return done.returncode, status, done.stdout


def main(argv):
    spec, frozen = json.loads(Path(argv[1]).read_text(encoding="utf-8")), Path(argv[2])
    build, kit = Path(spec["build"]), Path(spec["kit"])
    if frozen.exists():
        raise SystemExit(f"{frozen} exists: a freeze is never repeated in place")
    ids = [c["id"] for c in spec["controls"]]
    if len(ids) != len(set(ids)) or not ids:
        raise SystemExit("control ids must be distinct and present")
    (frozen / "packets").mkdir(parents=True)
    (frozen / "controls").mkdir()
    for part in ("model-assessment", "apply-trials", "recursive-apply-demo"):
        shutil.copytree(kit / part, frozen / "kit" / part, ignore=shutil.ignore_patterns("__pycache__"))
    build_text = (build / "build.txt").read_text(encoding="utf-8")
    manifest = {"build": str(build), "buildText": build_text, "ceilings": CEILINGS,
                "assessmentPass": sha(frozen / "kit" / "model-assessment" / "assessment-pass.json"),
                "harness": sha(build / "assessment.test"), "cli": sha(build / "markitect"),
                "kit": {p.relative_to(frozen / "kit").as_posix(): sha(p) for p in sorted((frozen / "kit").rglob("*")) if p.is_file()},
                "controls": {}}
    for control in spec["controls"]:
        cid, expectation = control["id"], control["expectation"]
        if expectation.get("expected") not in EXPECTED:
            raise SystemExit(f"{cid}: expected must be one of {sorted(EXPECTED)}")
        packet = frozen / "packets" / cid
        source = control["packet"]
        if "materialize" in source:
            subprocess.run([*source["materialize"], str(packet)], check=True, cwd=frozen / "kit", capture_output=True, text=True)
        else:
            subprocess.run(["git", "clone", "--quiet", "--no-hardlinks", source["repo"], str(packet)], check=True)
            if git(packet, "rev-parse", "HEAD") != source["head"]:
                raise SystemExit(f"{cid}: packet head is not {source['head']}")
            git(packet, "remote", "remove", "origin")
        if git(packet, "status", "--porcelain"):
            raise SystemExit(f"{cid}: packet is not one clean commit")
        code, status, output = check(build, packet)
        (frozen / "packets" / f"{cid}.check.json").write_text(output, encoding="utf-8")
        if code != 0 or status != "passed":
            raise SystemExit(f"{cid}: markitect check {status} (exit {code}); a control must pass the Model check and the File check")
        hidden = dict(expectation, id=cid, definitions=definitions(packet))
        (frozen / "controls" / f"{cid}.json").write_text(json.dumps(hidden, indent=1), encoding="utf-8")
        manifest["controls"][cid] = {"packet": str(packet), "head": git(packet, "rev-parse", "HEAD"), "tree": git(packet, "rev-parse", "HEAD^{tree}"),
                                     "files": {p: sha(packet / p) for p in git(packet, "ls-files").splitlines()},
                                     "check": status, "expectation": sha(frozen / "controls" / f"{cid}.json"), "source": source}
    (frozen / "manifest.json").write_text(json.dumps(manifest, indent=1), encoding="utf-8")
    (frozen / "ledger.txt").write_text("", encoding="utf-8")
    print(json.dumps({"frozen": str(frozen), "manifest": sha(frozen / "manifest.json"), "controls": ids}))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
