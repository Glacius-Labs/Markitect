"""Freeze the inputs of one Model assessment series before any provider work, and verify them before each attempt.

Usage:
  freeze.py freeze SPEC FROZEN_DIR
  freeze.py verify FROZEN_DIR CONTROL
  freeze.py ceilings FROZEN_DIR

SPEC (JSON) names the built candidate, the experiments directory of the clean
clone it was built from, the ledger of an earlier freeze of the same series if
there is one (its attempts keep counting), and the controls:
  {"build": BUILD_DIR, "kit": SOURCE_EXPERIMENTS_DIR, "ledgerFrom": EARLIER_LEDGER (optional),
   "controls": [{"id": ..., "packet": {"materialize": [ARGV...]} | {"repo": PATH, "head": SHA},
                 "expectation": {"expected": ..., "issue": {"definitions": [...], "terms": [REGEX...]},
                                 "freedom": {...}, "source": {...}}}]}

`materialize` runs with the frozen kit as working directory and the packet
directory appended as its last argument. Every packet must pass `markitect
check` (the Model check and the strict File check) with the build's CLI, or the
freeze is refused. The expectations go to FROZEN_DIR/controls, never into a
packet; the manifest binds every hash, and `verify` recomputes them. Nothing
here starts an Agent.
"""
import hashlib
import json
import re
import shutil
import subprocess
import sys
from pathlib import Path

EXPECTED = {"accepted", "model_gap", "model_error"}
CEILINGS = {"attempts": 12, "summedSeconds": 90 * 60, "attemptSeconds": 10 * 60, "model": "claude-sonnet-5-5", "effort": "high"}


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def git(repo, *args):
    return subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, text=True).stdout.strip()


def definitions(packet):
    """Definition identities of the packet's local model and its dependencies: package path and id per file."""
    markitect = packet / ".markitect"
    roots = [markitect / "model", *sorted((markitect / "dependencies").glob("*/model"))]
    out = []
    for root in roots:
        for path in sorted(root.rglob("*.yaml")):
            match = re.search(r"^id:\s*(\S+)\s*$", path.read_text(encoding="utf-8"), re.MULTILINE)
            if match:
                out.append(path.parent.relative_to(root).as_posix() + "/" + match.group(1))
    return out


def body_of(text):
    """The body of one Definition file: a literal or folded block, or a plain or quoted one-line value.

    A body this simple reader cannot take is refused rather than compared wrongly.
    """
    lines = text.splitlines()
    for i, line in enumerate(lines):
        match = re.match(r"^body:\s*(.*)$", line)
        if not match:
            continue
        value = match.group(1).strip()
        if value.startswith('"'):
            try:
                return json.loads(value)  # a one-line YAML double-quoted scalar uses JSON's escapes
            except json.JSONDecodeError:
                raise SystemExit(f"a double-quoted body this reader cannot take: {value[:60]}")
        if value.startswith("'"):
            if len(value) < 2 or not value.endswith("'"):
                raise SystemExit(f"a single-quoted body this reader cannot take: {value[:60]}")
            return value[1:-1].replace("''", "'")
        if value and value[0] not in "|>":
            return value
        block = []
        for next_line in lines[i + 1:]:
            if next_line.strip() and not next_line.startswith((" ", "\t")):
                break
            block.append(next_line.strip())
        return "\n".join(block).strip()
    return ""


def definition_texts(packet):
    """The package and body of every local Definition, for the delivery check."""
    root = packet / ".markitect" / "model"
    out = {}
    for path in sorted(root.rglob("*.yaml")):
        text = path.read_text(encoding="utf-8")
        match = re.search(r"^id:\s*(\S+)\s*$", text, re.MULTILINE)
        if match:
            package = path.parent.relative_to(root).as_posix()
            out[package + "/" + match.group(1)] = {"package": package, "body": body_of(text)}
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


def validate(cid, expectation):
    if expectation.get("expected") not in EXPECTED:
        raise SystemExit(f"{cid}: expected must be one of {sorted(EXPECTED)}")
    for key in ("issue", "freedom"):
        for term in (expectation.get(key) or {}).get("terms", []):
            try:
                re.compile(term)
            except re.error as e:
                raise SystemExit(f"{cid}: {key} term {term!r} is no regular expression: {e}")


def kit_hashes(kit):
    return {p.relative_to(kit).as_posix(): sha(p) for p in sorted(kit.rglob("*")) if p.is_file()}


def tracked(repo):
    return [p for p in subprocess.run(["git", "-C", str(repo), "ls-files", "-z"], check=True, capture_output=True, text=True).stdout.split("\0") if p]


def images(build_text):
    """The container images the build recorded: (reference, image ID)."""
    out = re.findall(r"^image (\S+) (sha256:[0-9a-f]{64})$", build_text, re.MULTILINE)
    out += [(i, i) for i in re.findall(r"^python_image=(sha256:[0-9a-f]{64})$", build_text, re.MULTILINE)]
    return out


def image_id(reference):
    done = subprocess.run(["docker", "image", "inspect", "--format", "{{.Id}}", reference], capture_output=True, text=True)
    return done.stdout.strip() if done.returncode == 0 else ""


def freeze(spec_path, frozen):
    spec = json.loads(Path(spec_path).read_text(encoding="utf-8"))
    build, kit = Path(spec["build"]).resolve(), Path(spec["kit"]).resolve()
    frozen = frozen.resolve()
    if frozen.exists():
        raise SystemExit(f"{frozen} exists: a freeze is never repeated in place")
    ids = [c["id"] for c in spec["controls"]]
    if len(ids) != len(set(ids)) or not ids:
        raise SystemExit("control ids must be distinct and present")
    for control in spec["controls"]:
        validate(control["id"], control["expectation"])
    build_text = (build / "build.txt").read_text(encoding="utf-8")
    built = re.search(r"^sha=(\w+)$", build_text, re.MULTILINE).group(1)
    if git(kit, "rev-parse", "HEAD") != built or git(kit, "status", "--porcelain"):
        raise SystemExit(f"{kit} is no clean checkout of the built {built}")
    (frozen / "packets").mkdir(parents=True)
    (frozen / "controls").mkdir()
    # The whole experiments directory: packet materialization reads shared inputs such as test-projects.
    shutil.copytree(kit, frozen / "kit", ignore=shutil.ignore_patterns("__pycache__"))
    manifest = {"build": str(build), "buildText": build_text, "kitSource": {"path": str(kit), "sha": built}, "ceilings": CEILINGS,
                "assessmentPass": sha(frozen / "kit" / "model-assessment" / "assessment-pass.json"),
                "harness": sha(build / "assessment.test"), "cli": sha(build / "markitect"),
                "kit": kit_hashes(frozen / "kit"), "controls": {}}
    for control in spec["controls"]:
        cid, expectation = control["id"], control["expectation"]
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
        hidden = dict(expectation, id=cid, definitions=definitions(packet), definitionTexts=definition_texts(packet))
        (frozen / "controls" / f"{cid}.json").write_text(json.dumps(hidden, indent=1), encoding="utf-8")
        manifest["controls"][cid] = {"packet": str(packet), "head": git(packet, "rev-parse", "HEAD"), "tree": git(packet, "rev-parse", "HEAD^{tree}"),
                                     "files": {p: sha(packet / p) for p in tracked(packet)},
                                     "check": status, "expectation": sha(frozen / "controls" / f"{cid}.json"), "source": source}
    # The ceilings hold for the series, not for one freeze: a later freeze carries the earlier ledger.
    ledger = Path(spec["ledgerFrom"]).read_text(encoding="utf-8") if spec.get("ledgerFrom") else ""
    if " status=running " in ledger:
        raise SystemExit("the earlier ledger has a running attempt: mark it abandoned first")
    manifest["ledgerFrom"] = {"path": spec.get("ledgerFrom", ""), "sha256": hashlib.sha256(ledger.encode()).hexdigest()}
    (frozen / "manifest.json").write_text(json.dumps(manifest, indent=1), encoding="utf-8")
    (frozen / "ledger.txt").write_text(ledger, encoding="utf-8")
    print(json.dumps({"frozen": str(frozen), "manifest": sha(frozen / "manifest.json"), "controls": ids}))
    return 0


def verify(frozen, control):
    """The differences between a frozen series and its manifest for one control; empty when intact."""
    manifest = json.loads((frozen / "manifest.json").read_text(encoding="utf-8"))
    build = Path(manifest["build"])
    problems = []
    if control not in manifest["controls"]:
        return [f"unknown control {control}"]
    entry = manifest["controls"][control]
    packet = frozen / "packets" / control
    expected = {"assessment Pass": (frozen / "kit" / "model-assessment" / "assessment-pass.json", manifest["assessmentPass"]),
                "harness": (build / "assessment.test", manifest["harness"]), "CLI": (build / "markitect", manifest["cli"]),
                "expectation": (frozen / "controls" / f"{control}.json", entry["expectation"])}
    for name, (path, digest) in expected.items():
        if not path.is_file() or sha(path) != digest:
            problems.append(f"the {name} {path} is not the frozen one")
    actual = kit_hashes(frozen / "kit")
    problems += [f"kit file {p} changed or missing" for p, digest in manifest["kit"].items() if actual.get(p) != digest]
    problems += [f"the image {ref} is not the built {iid}" for ref, iid in images(manifest["buildText"]) if image_id(ref) != iid]
    if git(packet, "rev-parse", "HEAD") != entry["head"] or git(packet, "rev-parse", "HEAD^{tree}") != entry["tree"] or git(packet, "status", "--porcelain"):
        problems.append(f"the packet {packet} is not the frozen commit")
    problems += [f"packet file {p} changed" for p, digest in entry["files"].items() if not (packet / p).is_file() or sha(packet / p) != digest]
    return problems


def ceilings(frozen):
    """The series ceilings of a freeze as shell words: attempts, summed seconds, seconds per attempt."""
    c = json.loads((frozen / "manifest.json").read_text(encoding="utf-8"))["ceilings"]
    return f"{c['attempts']} {c['summedSeconds']} {c['attemptSeconds']}"


def main(argv):
    if len(argv) == 4 and argv[1] == "freeze":
        return freeze(argv[2], Path(argv[3]))
    if len(argv) == 4 and argv[1] == "verify":
        problems = verify(Path(argv[2]).resolve(), argv[3])
        for p in problems:
            print(p, file=sys.stderr)
        return 1 if problems else 0
    if len(argv) == 3 and argv[1] == "ceilings":
        print(ceilings(Path(argv[2]).resolve()))
        return 0
    raise SystemExit(__doc__)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
