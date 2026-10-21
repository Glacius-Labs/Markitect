"""Freeze the evaluation kit: scenario packets, plan workspaces and their hashes; see README.md, "Freeze".

OPERATOR MATERIAL.

    python -B freeze.py lock                         # rewrite scenarios.lock.json from the committed sources
    python -B freeze.py check                        # exit 1 when scenarios.lock.json is not current
    python -B freeze.py out DIR [--markitect EXE]    # write every packet into DIR with manifest.json

scenarios.lock.json binds every scenario to its exact inputs: per Markitect
arm the packet's commit and tree and the digest of each file outside the
shared dependencies (the arms differ only in their strategy line), the plan
arm's workspace, plan.md and prompt, the policy, the budget, and the digest
of the kit that judges it; likewise every assessment control's model-only
packet with its version, expected conclusion and budget. Packets are built
from fixed authors, dates and bytes, so the same sources give the same
commits on every platform. A fixture change is a new lock, never a repair of
an attempt that ran on an earlier one. `out` materializes the packets for a
series; with --markitect it records each packet's structural check, which
depends on the product build and therefore stays out of the lock.
"""
import argparse
import hashlib
import json
import subprocess
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import context_controls  # noqa: E402
import plan_packet  # noqa: E402
from materialize import git, materialize, packet_files  # noqa: E402
from scenarios import ASSESSMENT_CONTROLS, MARKITECT, POLICY, SCENARIOS, BY_ID, check_catalog  # noqa: E402

LOCK = HERE / "scenarios.lock.json"
KIT_SUFFIXES = (".py", ".yaml", ".json", ".sh")
SHARED = HERE.parent / "test-projects" / "oracle_kit.py"


def sha256(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def kit_files():
    """Every file that builds or judges a packet; guides and generated pages excluded."""
    files = {}
    for path in sorted(HERE.rglob("*")):
        relative = path.relative_to(HERE).as_posix()
        if (path.is_file() and path.suffix in KIT_SUFFIXES and "__pycache__" not in path.parts
                and path != LOCK):
            files["experiments/apply-trials/" + relative] = sha256(path)
    files["experiments/test-projects/oracle_kit.py"] = sha256(SHARED)
    return files


def digest(mapping):
    return hashlib.sha256(json.dumps(mapping, sort_keys=True).encode("utf-8")).hexdigest()


def budget_of(item):
    """The packet budget of a scenario or an assessment control."""
    budget = item.budget
    return {"executionTime": budget.execution, "turnTimeout": budget.turn, "agentTurns": budget.turns,
            "passes": budget.passes}


def control_packet(control, target):
    """The model-only packet of one assessment control, with its budget and the direct strategy."""
    materialize(control.case, target, budget=budget_of(control), strategy="direct")
    return {"repo": str(Path(target).resolve()), **describe(target)}


def describe(repo):
    files = {path: sha256(Path(repo) / path) for path in packet_files(repo)
             if not path.startswith(".markitect/dependencies/")}
    dependencies = {path: sha256(Path(repo) / path) for path in packet_files(repo)
                    if path.startswith(".markitect/dependencies/")}
    return {"head": git(repo, "rev-parse", "HEAD").strip(), "tree": git(repo, "rev-parse", "HEAD^{tree}").strip(),
            "commits": len(git(repo, "rev-list", "HEAD").split()), "files": files,
            "dependenciesDigest": digest(dependencies)}


def scenario_packet(scenario_id, target, arm="direct"):
    """The Markitect packet of one scenario and arm: its case's packet with the scenario's budget and the arm's
    strategy. The plan arm's workspace is derived from the direct packet; its .markitect/ does not enter it."""
    scenario = BY_ID[scenario_id]
    strategy = arm if arm in MARKITECT else "direct"
    materialize(scenario.case, target, budget=budget_of(scenario), strategy=strategy)
    return {"repo": str(Path(target).resolve()), **describe(target)}


def lock_data():
    check_catalog()
    kit = kit_files()
    data = {"formatVersion": 1, "kitDigest": digest(kit), "kit": kit,
            "policy": POLICY, "planPrompt": plan_packet.PROMPT, "scenarios": {}, "assessmentControls": {},
            "contextControls": {}}
    with tempfile.TemporaryDirectory(prefix="freeze-") as folder:
        folder = Path(folder)
        for scenario in SCENARIOS:
            packets = {arm: scenario_packet(scenario.id, folder / scenario.id / arm, arm) for arm in MARKITECT}
            entry = {"case": scenario.case, "version": scenario.version, "expect": scenario.expect,
                     "arms": list(scenario.arms),
                     "preflight": scenario.preflight, "internalRoute": list(scenario.internal_route),
                     "budget": budget_of(scenario), "sessions": list(scenario.sessions),
                     "markitect": {arm: {k: v for k, v in packet.items() if k != "repo"} for arm, packet in packets.items()}}
            if "plan" in scenario.arms:
                plan = plan_packet.build_from_packet(packets["direct"]["repo"], folder / scenario.id / "plan")
                entry["plan"] = {key: plan[key] for key in ("head", "tree", "commits", "planSha256", "promptSha256", "files")}
            data["scenarios"][scenario.id] = entry
        for control in ASSESSMENT_CONTROLS:
            packet = control_packet(control, folder / control.id)
            data["assessmentControls"][control.id] = {
                "case": control.case, "version": control.version, "conclusion": control.conclusion,
                "budget": budget_of(control), **{k: v for k, v in packet.items() if k != "repo"}}
        for snapshot in ("c1", "c1-stale"):
            made = context_controls.materialize(snapshot, folder / snapshot)
            data["contextControls"][snapshot] = describe(made["target"])
        data["contextControls"]["expected.json"] = sha256(context_controls.EXPECTED)
    data["assessmentKey"] = sha256(HERE / "assessment-key.md")
    return data


def write_lock():
    text = json.dumps(lock_data(), indent=1, sort_keys=True) + "\n"
    LOCK.write_text(text, encoding="utf-8", newline="\n")
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def check_lock():
    current = json.dumps(lock_data(), indent=1, sort_keys=True) + "\n"
    recorded = LOCK.read_text(encoding="utf-8") if LOCK.is_file() else ""
    if current == recorded:
        return {"current": True, "sha256": hashlib.sha256(current.encode("utf-8")).hexdigest()}
    old, new = json.loads(recorded or "{}"), json.loads(current)
    changed = sorted(key for key in set(old.get("scenarios", {})) | set(new["scenarios"])
                     if old.get("scenarios", {}).get(key) != new["scenarios"].get(key))
    return {"current": False, "changedScenarios": changed, "kitChanged": old.get("kitDigest") != new["kitDigest"]}


def structural_check(markitect, repo):
    done = subprocess.run([markitect, "check", "--repo", repo, "--revision", "HEAD", *context_controls.MODEL_ARGS],
                          capture_output=True, text=True, encoding="utf-8")
    try:
        return json.loads(done.stdout).get("status")
    except ValueError:
        return "unreadable"


def write_out(directory, markitect=None):
    """Every packet of the lock under DIRECTORY, verified against the lock, with a manifest."""
    directory = Path(directory).resolve()
    if directory.exists():
        raise SystemExit("%s already exists" % directory)
    lock = json.loads(LOCK.read_text(encoding="utf-8"))
    manifest = {"lockSha256": sha256(LOCK), "kitDigest": lock["kitDigest"], "scenarios": {}}
    problems = []
    for scenario in SCENARIOS:
        expected = lock["scenarios"][scenario.id]
        entry = {}
        for arm in (arm for arm in scenario.arms if arm in MARKITECT):
            packet = scenario_packet(scenario.id, directory / scenario.id / arm, arm)
            if packet["head"] != expected["markitect"][arm]["head"]:
                problems.append("%s %s: packet head %s, locked %s" % (scenario.id, arm, packet["head"],
                                                                     expected["markitect"][arm]["head"]))
            entry[arm] = {"repo": packet["repo"], "head": packet["head"]}
            if markitect:
                entry[arm]["structuralCheck"] = structural_check(markitect, packet["repo"])
                if entry[arm]["structuralCheck"] != "passed":
                    problems.append("%s %s: markitect check %s" % (scenario.id, arm, entry[arm]["structuralCheck"]))
        if "plan" in scenario.arms:
            with tempfile.TemporaryDirectory(prefix="freeze-plan-") as folder:
                source = scenario_packet(scenario.id, Path(folder) / "direct", "direct")
                plan = plan_packet.build_from_packet(source["repo"], directory / scenario.id / "plan")
            if plan["head"] != expected["plan"]["head"]:
                problems.append("%s: plan head %s, locked %s" % (scenario.id, plan["head"], expected["plan"]["head"]))
            entry["plan"] = {"repo": plan["target"], "head": plan["head"]}
            (directory / scenario.id / "plan-prompt.md").write_text(plan_packet.PROMPT, encoding="utf-8", newline="\n")
        manifest["scenarios"][scenario.id] = entry
    manifest["assessmentControls"] = {}
    for control in ASSESSMENT_CONTROLS:
        packet = control_packet(control, directory / "assessment" / control.id)
        locked = lock["assessmentControls"][control.id]["head"]
        if packet["head"] != locked:
            problems.append("%s: packet head %s, locked %s" % (control.id, packet["head"], locked))
        entry = {"repo": packet["repo"], "head": packet["head"], "version": control.version}
        if markitect:
            entry["structuralCheck"] = structural_check(markitect, packet["repo"])
            if entry["structuralCheck"] != "passed":
                problems.append("%s: markitect check %s" % (control.id, entry["structuralCheck"]))
        manifest["assessmentControls"][control.id] = entry
    if markitect:
        version = subprocess.run([markitect, "version"], capture_output=True, text=True).stdout.strip()
        manifest["markitect"] = {"path": str(markitect), "version": version, "sha256": sha256(markitect)}
    manifest["problems"] = problems
    (directory / "manifest.json").write_text(json.dumps(manifest, indent=1) + "\n", encoding="utf-8")
    return manifest


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("lock")
    sub.add_parser("check")
    out = sub.add_parser("out")
    out.add_argument("directory")
    out.add_argument("--markitect")
    args = parser.parse_args(argv)
    if args.command == "lock":
        print(json.dumps({"written": str(LOCK), "sha256": write_lock()}))
        return 0
    if args.command == "check":
        result = check_lock()
        print(json.dumps(result, indent=1))
        return 0 if result["current"] else 1
    manifest = write_out(args.directory, args.markitect)
    print(json.dumps({key: manifest[key] for key in ("lockSha256", "kitDigest", "problems")}, indent=1))
    return 0 if not manifest["problems"] else 1


if __name__ == "__main__":
    sys.exit(main())
