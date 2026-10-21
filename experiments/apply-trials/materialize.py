"""Build the target repository (the Agents' packet) of one Apply trial case; see README.md.

    python -B materialize.py CASE TARGET [--from-repo REPO]

TARGET must not exist. Without --from-repo the packet is the case's model
snapshot with its model edits, the shared dependencies, the project's
execution settings and, for a reference start, the reference realization and
registry with the case's faults, committed once. With --from-repo the packet
is a clone of REPO's committed history plus its uncommitted publication
(committed as accepted), then the case's model. Operator material (this
directory's guides, cases, checks and validation overlays) never enters a
packet; the packet is audited for leaking terms before it is committed.
"""
import argparse
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

from cases import CASES  # noqa: E402

DEPENDENCIES = HERE.parent / "test-projects" / "library-r2" / "project" / ".markitect" / "dependencies"
PACKET_ROOTS = (".markitect", "realization")
LEAK = re.compile(r"\b(fault\w*|inject\w*|deliberate\w*|drift\w*|trial\w*|oracle\w*|answer key|mutant\w*|"
                  r"seeded|planted|[SDH]-[A-Z])\b", re.IGNORECASE)
IDENTITY = {"GIT_AUTHOR_NAME": "Project owner", "GIT_AUTHOR_EMAIL": "owner@example.invalid",
            "GIT_COMMITTER_NAME": "Project owner", "GIT_COMMITTER_EMAIL": "owner@example.invalid",
            "GIT_AUTHOR_DATE": "2026-10-20T08:00:00+00:00", "GIT_COMMITTER_DATE": "2026-10-20T08:00:00+00:00"}
GIT = ["git", "-c", "core.autocrlf=false", "-c", "core.safecrlf=false", "-c", "core.hooksPath=", "-c", "core.filemode=false",
       "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"]


class PacketError(Exception):
    pass


def git(repo, *args, check=True):
    done = subprocess.run([*GIT, "-C", str(repo), *args], capture_output=True, text=True,
                          env=dict(os.environ, **IDENTITY))
    if check and done.returncode != 0:
        raise PacketError("git %s failed: %s" % (" ".join(args), done.stderr.strip()))
    return done.stdout


def project_root(project):
    """A hand-written project directory, or the generated directory of a fleet project."""
    if project.startswith("fleet-"):
        import fleet
        return fleet.project_dir(project)
    return HERE / project


def project_dir(case):
    return project_root(case.project)


def apply_edits(root, edits, label):
    for edit in edits:
        path = Path(root) / edit.path
        if not path.is_file():
            raise PacketError("%s: %s does not exist" % (label, edit.path))
        text = path.read_text(encoding="utf-8")
        found = text.count(edit.old)
        if found != edit.count:
            raise PacketError("%s: %s holds the edited text %d times, expected %d" % (label, edit.path, found, edit.count))
        path.write_text(text.replace(edit.old, edit.new), encoding="utf-8", newline="\n")


def write_model(root, case, snapshot, edits=None):
    """Replace the packet's manifest, lock and Definitions by the snapshot plus EDITS (the case's model edits)."""
    source = project_dir(case) / "snapshots" / snapshot
    target = Path(root) / ".markitect"
    target.mkdir(parents=True, exist_ok=True)
    if (target / "model").exists():
        shutil.rmtree(target / "model")
    shutil.copytree(source / "model", target / "model")
    for name in ("model.yaml", "model-lock.yaml"):
        shutil.copyfile(source / name, target / name)
    apply_edits(root, case.model_edits if edits is None else edits, case.id + " model")


def write_base(root, case):
    """The first commit's content; a case with prior edits starts from the earlier model version it corrects."""
    target = Path(root) / ".markitect"
    shutil.copytree(DEPENDENCIES, target / "dependencies")
    shutil.copyfile(project_dir(case) / "execution.yaml", target / "execution.yaml")
    write_model(root, case, case.snapshot, case.prior_edits or None)
    if case.start == "reference":
        reference = project_dir(case) / "reference"
        shutil.copytree(reference / "realization", Path(root) / "realization",
                        ignore=shutil.ignore_patterns("__pycache__"))
        shutil.copytree(reference / "registry", target / "registry")
        apply_edits(root, case.stand_edits, "%s stand" % case.id)
        for fault in case.faults:
            apply_edits(root, fault.edits, "%s fault %s" % (case.id, fault.id))


def packet_files(root):
    root = Path(root)
    files = []
    for path in sorted(root.rglob("*")):
        relative = path.relative_to(root).as_posix()
        if relative == ".git" or relative.startswith(".git/") or not path.is_file():
            continue
        files.append(relative)
    return files


def audit(root, strict=lambda relative: True):
    """Refuse a packet that holds operator material or names the trial's answers.

    Files for which `strict` is false (an earlier Apply's own artifacts) and
    their history only produce warnings: the materializer did not write them.
    """
    problems, warnings = [], []
    for relative in packet_files(root):
        if relative.split("/", 1)[0] not in PACKET_ROOTS:
            problems.append("unexpected top-level path " + relative)
            continue
        if relative.startswith(".markitect/dependencies/"):
            continue
        text = (Path(root) / relative).read_bytes().decode("utf-8", errors="replace")
        for match in LEAK.finditer(relative + "\n" + text):
            (problems if strict(relative) else warnings).append("%s names %r" % (relative, match.group(0)))
    for match in LEAK.finditer(git(root, "log", "--format=%B")):
        (problems if strict("") else warnings).append("a commit message names %r" % match.group(0))
    if problems:
        raise PacketError("packet audit failed:\n  " + "\n  ".join(problems))
    return warnings


def accept_publication(source, target):
    """Copy SOURCE's uncommitted changes (an Apply publication) into the clone TARGET."""
    status = subprocess.run([*GIT, "-C", str(source), "status", "--porcelain=v1", "-z", "-uall"],
                            capture_output=True, check=True).stdout.decode("utf-8").split("\0")
    changed = []
    for entry in filter(None, status):
        relative = entry[3:]
        changed.append(relative)
        origin, destination = Path(source) / relative, Path(target) / relative
        if origin.is_file():
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(origin, destination)
        elif destination.exists():
            destination.unlink()
    return changed


def commit(root, message):
    git(root, "add", "-A")
    if git(root, "status", "--porcelain"):
        git(root, "commit", "-q", "-m", message)


def model_case(root):
    """The first case with an oracle profile whose model the repository holds byte for byte, or None."""
    from tempfile import TemporaryDirectory
    for case in CASES.values():
        if not case.snapshot or not case.profile:
            continue
        with TemporaryDirectory(prefix="trial-model-") as folder:
            write_model(folder, case, case.snapshot)
            if same_tree(Path(folder) / ".markitect", Path(root) / ".markitect", ("model", "model.yaml", "model-lock.yaml")):
                return case
    return None


def same_tree(left, right, names):
    def collect(base):
        found = {}
        for name in names:
            path = base / name
            if path.is_file():
                found[name] = path.read_bytes()
            elif path.is_dir():
                for item in path.rglob("*"):
                    if item.is_file():
                        found[item.relative_to(base).as_posix()] = item.read_bytes()
        return found
    return collect(left) == collect(right)


def set_budget(root, budget):
    """Replace the packet's budget block by BUDGET {executionTime, turnTimeout, agentTurns, passes}."""
    path = Path(root) / ".markitect" / "execution.yaml"
    lines = path.read_text(encoding="utf-8").split("\n")
    start = lines.index("budget:")
    end = start + 1
    while end < len(lines) and lines[end].startswith("  "):
        end += 1
    block = ["budget:"] + ["  %s: %s" % (key, budget[key]) for key in ("executionTime", "turnTimeout", "agentTurns", "passes")]
    path.write_text("\n".join(lines[:start] + block + lines[end:]), encoding="utf-8", newline="\n")


def set_strategy(root, strategy):
    """Select the packet's Apply strategy, the one line in which the direct and staged arms differ."""
    path = Path(root) / ".markitect" / "execution.yaml"
    text = path.read_text(encoding="utf-8")
    changed, count = re.subn(r"(?m)^strategy: .*$", "strategy: " + strategy, text)
    if count != 1:
        raise PacketError("%s has no single strategy line" % path)
    path.write_text(changed, encoding="utf-8", newline="\n")


def configure(root, budget, strategy):
    if budget:
        set_budget(root, budget)
    if strategy:
        set_strategy(root, strategy)


def materialize(case_id, target, from_repo=None, budget=None, strategy=None):
    """BUDGET and STRATEGY, when given, replace the execution budget and strategy in the first commit (not with
    --from-repo)."""
    if case_id not in CASES:
        raise PacketError("unknown case %s; known: %s" % (case_id, ", ".join(CASES)))
    case = CASES[case_id]
    target = Path(target).resolve()
    if target.exists():
        raise PacketError("%s already exists" % target)
    accepted = []
    if from_repo:
        target.parent.mkdir(parents=True, exist_ok=True)
        git(target.parent, "clone", "-q", "--no-hardlinks", str(Path(from_repo).resolve()), str(target))
        git(target, "remote", "remove", "origin")
        accepted = accept_publication(from_repo, target)
        commit(target, "Accept the published realization")
        if case.snapshot:
            write_model(target, case, case.snapshot)
            commit(target, "Update the model")
    elif case.start == "previous" and case.snapshot:
        # Controlled form: the reference stands in for the previous result.
        reference = CASES["S-N"] if case.project == "stockroom" else CASES["D-N"]
        target.mkdir(parents=True)
        write_base(target, reference)
        configure(target, budget, strategy)
        git(target, "init", "-q")
        commit(target, "Current project state")
        write_model(target, case, case.snapshot)
        commit(target, "Update the model")
    elif case.start == "previous":
        raise PacketError("%s needs --from-repo: the committed result of an earlier Apply" % case.id)
    else:
        target.mkdir(parents=True)
        write_base(target, case)
        configure(target, budget, strategy)
        git(target, "init", "-q")
        commit(target, "Current project state")
        if case.prior_edits:
            write_model(target, case, case.snapshot)
            commit(target, "Correct the model")
    warnings = audit(target, (lambda relative: relative.startswith(".markitect/model")) if from_repo else (lambda relative: True))
    return {"case": case.id, "target": str(target), "head": git(target, "rev-parse", "HEAD").strip(),
            "files": len(packet_files(target)), "acceptedFromSource": accepted, "auditWarnings": warnings}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("case")
    parser.add_argument("target")
    parser.add_argument("--from-repo", help="the repository of an earlier Apply, with its publication")
    args = parser.parse_args(argv)
    try:
        print(json.dumps(materialize(args.case, args.target, args.from_repo), indent=2))
    except PacketError as error:
        print("materialize: %s" % error, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
