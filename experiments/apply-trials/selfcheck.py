"""Local self-check of the Apply trial fixtures and their independent checks; see README.md.

    python -B selfcheck.py [--project NAME ...] [--markitect PATH] [--python CMD]

Shows, without any Agent or provider:
1. every Definition a check cites exists in its profile's model;
2. the reference passes its base profile and its own tests, and each
   validation overlay passes its profile;
3. degenerate command lines fail every profile;
4. each fault alone fails exactly its declared oracle and artifact checks and
   breaks the reference tests exactly when declared, and a case's faults
   together fail exactly the union;
5. the reference fails exactly the declared checks of each changed profile;
6. every case materializes and passes the packet audit, and with --markitect
   the packet passes `markitect check` and `context` at HEAD;
7. the assessor judges simulated results: unrepaired, partly and fully
   repaired starts, an unchanged stand and a model-defect report;
8. each oracle variant fails exactly its declared checks: accepted
   alternatives fail nothing and wrong versions are rejected;
9. the depot hyphen fault reproduces the faulty bytes that the hardening
   series froze, and a model change on the reference is judged as met only
   when its documentation follows;
10. every conforming but unattractive stand (#561) passes its profile and its
   own tests, and its edits touch exactly the paths it declares tempting.
Exits 0 only without problems.
"""
import argparse
import hashlib
import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
sys.path.insert(0, str(HERE.parent / "test-projects"))

import assess  # noqa: E402
import oracle  # noqa: E402
from cases import (APPLIED, CASES, DEFECT, GAP, N1_HYPHEN, N1_HYPHEN_SHA256, PROJECTS, REFUSED, SHIFTS,  # noqa: E402
                   UNCHANGED, VARIANTS, Edit, markdown)
from materialize import apply_edits, materialize, project_root  # noqa: E402
from oracle_kit import DEGENERATE  # noqa: E402
from trial_kit import python_command  # noqa: E402

VALIDATION = {
    "stockroom": {"s2": ("validation/s2", ["limits.json"]), "o": ("validation/o", ["limits.json"])},
}
MODEL_ARGS = ["--model", ".markitect/model.yaml", "--dependency", ".markitect/dependencies/core/model.yaml",
              "--dependency", ".markitect/dependencies/software/model.yaml"]


class Report:
    def __init__(self):
        self.problems, self.facts = [], {}

    def expect(self, condition, problem):
        if not condition:
            self.problems.append(problem)
        return condition


def build(project, root, faults=(), overlay=None, remove=(), stand=()):
    """A packet-shaped folder ROOT holding the reference realization and registry with overlays, a case's stand
    edits and faults."""
    root = Path(root)
    shutil.copytree(project_root(project) / "reference" / "realization", root / "realization",
                    ignore=shutil.ignore_patterns("__pycache__"))
    shutil.copytree(project_root(project) / "reference" / "registry", root / ".markitect" / "registry")
    if overlay:
        for path in (project_root(project) / overlay).iterdir():
            shutil.copyfile(path, root / "realization" / path.name)
    for name in remove:
        (root / "realization" / name).unlink()
    apply_edits(root, stand, "stand")
    for fault in faults:
        apply_edits(root, fault.edits, "fault " + fault.id)
    return root / "realization"


def reference_file(project, path):
    """The reference source of a packet path: realization/... or .markitect/registry/..."""
    reference = project_root(project) / "reference"
    if path.startswith(".markitect/registry/"):
        return reference / "registry" / path[len(".markitect/registry/"):]
    return reference / path


def failing(project, profile, realization, python, previous=None):
    outcome = oracle.assess(project, profile, realization, python, previous)
    return {item["id"] for item in outcome["checks"] if item["status"] != "pass"}, outcome


def check_citations(project, report):
    checks = oracle.load_checks(project)
    for profile, parameters in checks.profiles.items():
        model = project_root(project) / "snapshots" / parameters["snapshot"] / "model"
        known = {path.relative_to(model).with_suffix("").as_posix() for path in model.rglob("*.yaml")}
        for item in checks.selected(profile):
            for definition in item["definitions"]:
                report.expect(definition in known, "%s cites %s, absent from profile %s" % (item["id"], definition, profile))
    report.facts["citations"] = "%d checks" % len(checks.checks)


def check_references(project, python, report):
    base = PROJECTS[project]["profile"]
    with tempfile.TemporaryDirectory(prefix="trial-self-") as folder:
        realization = build(project, folder)
        bad, outcome = failing(project, base, realization, python)
        report.expect(not bad, "the reference fails %s: %s" % (base, sorted(bad)))
        passed, output = assess.run_tests(realization, project, python)
        report.expect(passed, "the reference tests fail: " + output)
        report.facts["reference"] = "%s %d/%d, own tests %s" % (base, outcome["summary"]["passed"],
                                                                 outcome["summary"]["total"], "pass" if passed else "fail")
    for profile, (overlay, remove) in VALIDATION.get(project, {}).items():
        with tempfile.TemporaryDirectory(prefix="trial-self-") as folder:
            realization = build(project, Path(folder) / "new", overlay=overlay, remove=remove)
            previous = build(project, Path(folder) / "old") if profile == "s2" else None
            bad, outcome = failing(project, profile, realization, python, previous)
            report.expect(not bad, "validation overlay %s fails %s" % (profile, sorted(bad)))
            report.facts["validation " + profile] = "%d/%d" % (outcome["summary"]["passed"], outcome["summary"]["total"])


def check_degenerate(project, python, report):
    checks = oracle.load_checks(project)
    for name, source in DEGENERATE.items():
        with tempfile.TemporaryDirectory(prefix="trial-degenerate-") as folder:
            (Path(folder) / "app.py").write_text(source, encoding="utf-8")
            for profile in checks.profiles:
                outcome = oracle.assess(project, profile, folder, python)
                summary = outcome["summary"]
                report.expect(not outcome["passed"] and summary["passed"] * 2 <= summary["total"],
                              "degenerate %s passes %d of %d %s checks" % (name, summary["passed"], summary["total"], profile))


def check_faults(project, python, report):
    found = {}
    for case in CASES.values():
        if case.project != project or not case.faults:
            continue
        union = set()
        for fault in case.faults:
            with tempfile.TemporaryDirectory(prefix="trial-fault-") as folder:
                realization = build(project, folder, faults=[fault], stand=case.stand_edits)
                bad, _ = failing(project, case.profile or PROJECTS[project]["profile"], realization, python)
                report.expect(bad == set(fault.detected_by), "%s %s fails %s, declared %s"
                              % (case.id, fault.id, sorted(bad), sorted(fault.detected_by)))
                relevant = sorted(set(case.static) | {name for other in case.faults for name in other.static})
                static = {name for name, problem in assess.run_static(relevant, realization, case).items() if problem}
                report.expect(static == set(fault.static), "%s %s fails artifact checks %s, declared %s"
                              % (case.id, fault.id, sorted(static), sorted(fault.static)))
                passed, _ = assess.run_tests(realization, project, python)
                report.expect(passed != fault.test_signal, "%s %s: reference tests %s, declared test signal %s"
                              % (case.id, fault.id, "pass" if passed else "fail", fault.test_signal))
                union |= set(fault.detected_by)
                found["%s %s" % (case.id, fault.id)] = sorted(bad) + sorted(static) + (["tests"] if not passed else [])
        with tempfile.TemporaryDirectory(prefix="trial-fault-") as folder:
            realization = build(project, folder, faults=case.faults, stand=case.stand_edits)
            bad, _ = failing(project, case.profile or PROJECTS[project]["profile"], realization, python)
            report.expect(bad == union, "%s all faults fail %s, declared %s" % (case.id, sorted(bad), sorted(union)))
    report.facts["faults"] = found


def check_shifts(project, python, report):
    for (owner, profile), declared in SHIFTS.items():
        if owner != project:
            continue
        with tempfile.TemporaryDirectory(prefix="trial-shift-") as folder:
            realization = build(project, folder)
            bad, _ = failing(project, profile, realization, python, realization if profile == "s2" else None)
            report.expect(bad == set(declared), "the reference under %s fails %s, declared %s"
                          % (profile, sorted(bad), sorted(declared)))
            report.facts["shift " + profile] = sorted(bad)


def check_packets(project, markitect, report):
    packets = {}
    with tempfile.TemporaryDirectory(prefix="trial-packets-") as folder:
        for case in CASES.values():
            if case.project != project or (case.start == "previous" and not case.snapshot):
                continue
            target = Path(folder) / case.id
            try:
                made = materialize(case.id, target)
            except Exception as error:  # noqa: BLE001 - every failure is a fixture problem
                report.expect(False, "%s does not materialize: %s" % (case.id, error))
                continue
            entry = {"files": made["files"], "head": made["head"][:12]}
            if markitect:
                for command in (["check"], ["context", PROJECTS[project]["exception"]]):
                    done = subprocess.run([markitect, *command, "--repo", str(target), "--revision", "HEAD", *MODEL_ARGS],
                                          capture_output=True, text=True, encoding="utf-8")
                    try:
                        result = json.loads(done.stdout)
                        status = result.get("status")
                        accounting = result["stages"]["structural"]["files"].get("accounting") or {}
                        findings = sorted((item["Code"], item["Path"]) for item in accounting.get("findings") or [])
                    except (ValueError, KeyError, TypeError):
                        status, findings = "unreadable", []
                    expected = STRICT_ACCOUNTING_FAILS.get(case.id, [])
                    report.expect(status == ("failed" if expected else "passed") and findings == expected,
                                  "%s: markitect %s is %s with %s, expected the findings %s"
                                  % (case.id, command[0], status, findings, expected))
                    entry[command[0]] = status if not findings else "%s %s" % (status, findings)
            packets[case.id] = entry
    report.facts["packets"] = packets


# Cases whose start stand fails strict accounting on purpose, with exactly these findings; Apply refuses
# them before any work (DEC-064).
STRICT_ACCOUNTING_FAILS = {"D-R": [("file.unexplained", "realization/reports/views.py"),
                                   ("registry.outside_area", "realization/reports/views.py")]}


def fake_report(path, run_status, publication, reason="", questions=(), status=None, preparation="passed"):
    """A minimal Apply report; RUN_STATUS None leaves out the run, as a failed preparation does."""
    report = {"formatVersion": 1, "operation": "apply", "status": status or ("passed" if publication else "failed"),
              "preparation": {"status": preparation, "attempts": [{"fileDiagnostics": []}]}}
    if run_status:
        report["run"] = {"id": "r", "status": run_status, "reason": reason, "passes": [], "findings": [],
                         "questions": [{"from": ".", "to": "user", "text": text, "answered": False} for text in questions],
                         "problems": [], "turns": [], "usage": {}}
        if reason.startswith("Root judged the model erroneous or contradictory: "):
            report["run"]["problems"] = [{"manager": ".", "category": "model", "abort": True,
                                          "text": reason.removeprefix("Root judged the model erroneous or contradictory: ")}]
    if publication:
        report["publication"] = {"Outcome": publication}
    Path(path).write_text(json.dumps(report), encoding="utf-8")
    return str(path)


def check_assessor(project, python, report):
    """Judge simulated results of the drift, artifact, no-change and defect cases."""
    with tempfile.TemporaryDirectory(prefix="trial-assess-") as folder:
        folder = Path(folder)
        for case in CASES.values():
            if case.project != project:
                continue
            if case.expected == REFUSED:
                repo = folder / case.id
                materialize(case.id, repo)
                # The refused report as Apply writes it: failed in preparation, no run, no findings named.
                judged = assess.assess(case.id, repo, fake_report(folder / "refused.json", None, None,
                                                                  preparation="failed"), python)
                report.expect(judged["outcome"]["met"] is True and judged["outcome"]["defectNamed"] is False,
                              "%s: a refused start is not judged as met: %s" % (case.id, judged["outcome"]))
                judged = assess.assess(case.id, repo, fake_report(folder / "stopped.json", None, None,
                                                                  status="not_completed"), python)
                report.expect(judged["outcome"]["met"] is False, "%s: a start that never prepared counts as refused" % case.id)
                judged = assess.assess(case.id, repo, fake_report(folder / "ran.json", "verified", "applied"), python)
                report.expect(judged["outcome"]["met"] is False, "%s: a run that started counts as refused" % case.id)
            elif case.faults:
                repo = folder / case.id
                materialize(case.id, repo)
                start = assess.assess(case.id, repo, None, python)
                report.expect(not any(item["repaired"] for item in start["faults"]),
                              "%s: the unrepaired start counts a fault as repaired" % case.id)
                # Partial repair: only the first fault is undone, by restoring its files from the reference
                # or, where another fault shares a file, by reversing its edits.
                first = case.faults[0]
                shared = {edit.path for fault in case.faults[1:] for edit in fault.edits}
                for edit in first.edits:
                    if edit.path in shared:
                        apply_edits(repo, [Edit(edit.path, edit.new, edit.old, edit.count)], "undo " + first.id)
                    else:
                        shutil.copyfile(reference_file(project, edit.path), repo / edit.path)
                partial = assess.assess(case.id, repo, None, python)
                repaired = [item["id"] for item in partial["faults"] if item["repaired"]]
                report.expect(repaired == [first.id], "%s: partial repair counts %s as repaired" % (case.id, repaired))
                publish(project, repo, stand=case.stand_edits)
                full = assess.assess(case.id, repo, fake_report(folder / "applied.json", "verified", "applied"), python)
                report.expect(all(item["repaired"] for item in full["faults"]), "%s: the reference repair is not "
                              "counted as complete: %s" % (case.id, full["faults"]))
                report.expect(full["outcome"]["met"] is True, "%s: a verified, applied repair does not meet the "
                              "expectation: %s" % (case.id, full["outcome"]))
                report.expect(not full["changes"]["outsideFaultsAndAllowed"], "%s: the reference repair changes %s"
                              % (case.id, full["changes"]["outsideFaultsAndAllowed"]))
                missing = [item["id"] for item in full["faults"]
                           if item.get("regressionTestRequired") and item["regressionTest"]["status"] != "present"]
                report.expect(not missing, "%s: required regression tests not found in the reference: %s" % (case.id, missing))
                report.facts["assess " + case.id] = {
                    "partial": repaired, "regression": {item["id"]: item["regressionTest"]["status"] for item in full["faults"]}}
            elif case.expected in (DEFECT, GAP) and case.start == "reference":
                repo = folder / case.id
                materialize(case.id, repo)
                terms = " ".join(group[0] for group in case.defect_terms)
                for name, status, reason in (("old", "incomplete", "model defect reported by Root: "),
                                             ("new", "failed", "Root judged the model erroneous or contradictory: ")):
                    judged = assess.assess(case.id, repo, fake_report(folder / (name + ".json"), status, None, reason + terms),
                                           python)
                    report.expect(judged["outcome"]["met"] is (name == "new") and judged["outcome"]["defectNamed"],
                                  "%s: the %s model-defect ending is not judged as met: %s" % (case.id, name, judged["outcome"]))
                asked = assess.assess(case.id, repo, fake_report(folder / "asked.json", "incomplete", None, "deadline",
                                                                 questions=[terms]), python)
                report.expect(asked["outcome"]["met"] is False,
                              "%s: a question to the user is judged %s" % (case.id, asked["outcome"]))
                (repo / "realization" / "README.md").write_text("changed\n", encoding="utf-8")
                judged = assess.assess(case.id, repo, fake_report(folder / "defect.json", "failed", None,
                                                                  "Root judged the model erroneous or contradictory: x"), python)
                report.expect(judged["outcome"]["met"] is False, "%s: a changed tree after a defect counts as met" % case.id)
            elif case.expected == UNCHANGED and case.start == "reference":
                repo = folder / case.id
                materialize(case.id, repo)
                judged = assess.assess(case.id, repo, fake_report(folder / "same.json", "verified", "no_change"), python)
                report.expect(judged["outcome"]["met"] and judged["behavior"]["passed"] and judged["tests"]["passed"],
                              "%s: an unchanged conforming stand is not judged as met: %s" % (case.id, judged["outcome"]))


def check_stands(project, python, report):
    """A conforming but unattractive stand: independently conforming, and its tempting spots really differ."""
    found = {}
    for case in CASES.values():
        if case.project != project or not case.stand_edits or case.faults:
            continue
        with tempfile.TemporaryDirectory(prefix="trial-stand-") as folder:
            realization = build(project, folder, stand=case.stand_edits)
            bad, outcome = failing(project, case.profile, realization, python)
            passed, output = assess.run_tests(realization, project, python)
            report.expect(not bad and passed, "the %s stand fails %s, own tests %s" % (case.id, sorted(bad), passed))
            changed = sorted({edit.path for edit in case.stand_edits})
            report.expect(changed == sorted(case.preserve), "the %s stand edits %s, its spots are %s"
                          % (case.id, changed, sorted(case.preserve)))
            found[case.id] = "%s %d/%d, own tests %s, spots %s" % (case.profile, outcome["summary"]["passed"],
                                                                  outcome["summary"]["total"],
                                                                  "pass" if passed else "fail", len(changed))
    if found:
        report.facts["stands"] = found


def variant_static(variant):
    """The artifact checks of the cases that share the variant's project and profile."""
    return sorted({name for case in CASES.values() if case.project == variant.project and case.profile == variant.profile
                   for name in case.static})


def check_variants(project, python, report):
    found = {}
    for variant in VARIANTS:
        if variant.project != project:
            continue
        with tempfile.TemporaryDirectory(prefix="trial-variant-") as folder:
            realization = build(project, folder)
            apply_edits(folder, variant.edits, "variant " + variant.id)
            bad, _ = failing(project, variant.profile, realization, python)
            report.expect(bad == set(variant.fails), "variant %s fails %s, declared %s"
                          % (variant.id, sorted(bad), sorted(variant.fails)))
            case = next(case for case in CASES.values() if case.project == project)
            static = {name for name, problem in assess.run_static(variant_static(variant), realization, case).items()
                      if problem}
            report.expect(static == set(variant.static), "variant %s fails artifact checks %s, declared %s"
                          % (variant.id, sorted(static), sorted(variant.static)))
            passed, _ = assess.run_tests(realization, project, python)
            report.expect(passed is not variant.tests_fail, "variant %s: own tests %s, declared failing %s"
                          % (variant.id, "pass" if passed else "fail", variant.tests_fail))
            found[variant.id] = sorted(bad | static | ({"own tests"} if not passed else set())) or "accepted"
    if found:
        report.facts["variants"] = found


def check_continuity(project, report):
    """The depot hyphen fault is byte for byte the start that the hardening series froze for V1 and A1."""
    if project != "depot":
        return
    with tempfile.TemporaryDirectory(prefix="trial-continuity-") as folder:
        build(project, folder, faults=[N1_HYPHEN])
        digests = {path: hashlib.sha256((Path(folder) / path).read_bytes()).hexdigest() for path in N1_HYPHEN_SHA256}
        report.expect(digests == N1_HYPHEN_SHA256, "the n1 fault does not reproduce the frozen bytes: %s" % digests)
        report.facts["continuity n1"] = "matches the hardening series' V1/A1 bytes" if digests == N1_HYPHEN_SHA256 else digests


def check_model_change(project, python, report):
    """A model change on the reference (start `previous` with a snapshot): complete and code-only publications."""
    for case in CASES.values():
        if case.project != project or case.start != "previous" or not case.snapshot or case.expected != APPLIED:
            continue
        variants = [variant for variant in VARIANTS if variant.project == project and variant.profile == case.profile]
        if not variants:
            continue
        with tempfile.TemporaryDirectory(prefix="trial-change-") as folder:
            folder = Path(folder)
            judged = {}
            for variant in variants:
                repo = folder / variant.id
                materialize(case.id, repo)
                apply_edits(repo, variant.edits, "variant " + variant.id)
                result = assess.assess(case.id, repo, fake_report(folder / "applied.json", "verified", "applied"), python)
                good = not variant.fails and not variant.static and not variant.tests_fail
                met = (result["outcome"]["met"] and result["behavior"]["passed"] and result["tests"]["passed"]
                       and not any(result["static"].values()) and not result["changes"]["outsideFaultsAndAllowed"])
                report.expect(met is good, "%s with variant %s is judged %s, expected %s: %s %s %s"
                              % (case.id, variant.id, met, good, result["behavior"]["failed"], result["static"],
                                 result["changes"]["outsideFaultsAndAllowed"]))
                judged[variant.id] = "met" if met else "not met"
            report.facts["change " + case.id] = judged


# Model-only start, optional model change with the overlay that realizes it, follow-up case.
LONGITUDINAL = {"stockroom": ("S-G", "S-C", "S-F", "validation/s2", ["limits.json"]),
                "depot": ("D-G", None, "D-F", None, [])}


def publish(project, repo, overlay=None, remove=(), stand=()):
    """Simulate a publication: the reference (with an overlay or a case's stand) and its registry in REPO's tree."""
    for folder in (repo / "realization", repo / ".markitect" / "registry"):
        if folder.exists():
            shutil.rmtree(folder)
    build(project, repo, overlay=overlay, remove=remove, stand=stand)


def check_longitudinal(project, python, report):
    """Model-only start, its accepted publication, the model change on it and the follow-up Apply."""
    if project not in LONGITUDINAL:
        return
    first, change, follow, overlay, remove = LONGITUDINAL[project]
    with tempfile.TemporaryDirectory(prefix="trial-chain-") as folder:
        folder = Path(folder)
        publish(project, materialize_path(first, folder / "first"))
        before, last = assess.changed_paths(folder / "first"), folder / "first"
        if change:
            made = materialize(change, folder / "change", from_repo=folder / "first")
            report.expect(assess.changed_paths(folder / "first") == before, "materializing changed the source repository")
            report.expect(made["acceptedFromSource"] == before, "%s accepted %s, the publication was %s"
                          % (change, made["acceptedFromSource"], before))
            report.expect(not assess.changed_paths(folder / "change"), "%s starts with uncommitted changes" % change)
            publish(project, folder / "change", overlay=overlay, remove=remove)
            judged = assess.assess(change, folder / "change", fake_report(folder / "applied.json", "verified", "applied"),
                                   python)
            report.expect(judged["outcome"]["met"] and judged["behavior"]["passed"] and not judged["behavior"]["skipped"],
                          "%s on the accepted publication is not judged as met: %s %s"
                          % (change, judged["outcome"], judged["behavior"]))
            last = folder / "change"
        materialize(follow, folder / "follow", from_repo=last)
        judged = assess.assess(follow, folder / "follow", fake_report(folder / "same.json", "verified", "no_change"),
                               python, build(project, folder / "previous") if change else None)
        expected = CASES[change or first].profile
        report.expect(judged["profile"] == expected and judged["outcome"]["met"] and judged["behavior"]["passed"]
                      and not judged["behavior"]["skipped"], "%s is not judged as met: profile %s, %s, %s"
                      % (follow, judged["profile"], judged["outcome"], judged["behavior"]))
        report.facts["longitudinal"] = " -> ".join(filter(None, (first, change, follow))) + " judged as met"


def materialize_path(case_id, target):
    materialize(case_id, target)
    return target


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--project", choices=sorted(PROJECTS), action="append")
    parser.add_argument("--markitect", help="a markitect executable for the structural packet checks")
    parser.add_argument("--python", default="", help="interpreter command (default: this interpreter)")
    args = parser.parse_args(argv)
    python = python_command(args.python)
    report, facts = Report(), {}
    key = (HERE / "answer-key.md").read_text(encoding="utf-8")
    report.expect(key == markdown(), "answer-key.md differs from `python -B cases.py`; regenerate it")
    for project in args.project or sorted(PROJECTS):
        report.facts = facts.setdefault(project, {})
        check_citations(project, report)
        check_references(project, python, report)
        check_degenerate(project, python, report)
        check_faults(project, python, report)
        check_shifts(project, python, report)
        check_packets(project, args.markitect, report)
        check_assessor(project, python, report)
        check_longitudinal(project, python, report)
        check_stands(project, python, report)
        check_variants(project, python, report)
        check_continuity(project, report)
        check_model_change(project, python, report)
    print(json.dumps({"facts": facts, "problems": report.problems, "passed": not report.problems},
                     indent=2))
    return 0 if not report.problems else 1


if __name__ == "__main__":
    sys.exit(main())
