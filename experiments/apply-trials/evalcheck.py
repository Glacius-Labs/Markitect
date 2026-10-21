"""Local self-check of the evaluation layer; see README.md, "Self-check".

    python -B evalcheck.py [--markitect PATH] [--quick]

Without any Agent or provider it shows that:
1. the scenario catalog is consistent and scenarios.lock.json is current;
2. evaluate.py judges simulated attempts of every scenario and arm as
   declared: the justified stop on A4-gap and A7 is met, while a late stop, a
   stop that names another defect, a silent invention, a stop with a
   publication, a question to the owner and a technical failure are not; a
   model stop or blocker on A4-fixed, FREE, A7-near or DEP is a false abort;
   an optional change, a false success and an unrepaired fault are not met;
   missing evidence is incomplete;
3. A4-fixed's packet holds the gap model in its first commit and the
   corrected model, equal to s1, in its second;
4. the DEP process analysis accepts a traced dependency and internal question
   route and rejects premature work, stale consumption, a binding to a
   superseded version, an undeclared dependency, a choice decided without the
   tree and idle turns;
5. every plan arm's plan.md carries every Definition of its scenario in full,
   compared with the product's checked model when --markitect is given;
6. the model context controls pass their self-test.
--quick skips the simulated attempts of 2 and 3, which run every oracle.
Exits 0 only without problems.
"""
import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import context_controls  # noqa: E402
import evaluate  # noqa: E402
import freeze  # noqa: E402
import plan_packet  # noqa: E402
import selfcheck  # noqa: E402
from cases import VARIANTS  # noqa: E402
from materialize import GIT, apply_edits  # noqa: E402
from scenarios import (BY_ID, COMPLETED, FALSE_ABORT, JUSTIFIED_STOP, LATE_STOP, MISSED, OWNER_CHANNEL,  # noqa: E402
                       SCENARIOS, SILENT_INVENTION, TECHNICAL, UNAUTHORIZED_PUBLICATION, WRONG_STOP, check_catalog)
from trial_kit import python_command  # noqa: E402

GAP_STOP = "Root confirmed a model gap: stockroom/inputs refers to the maximum name length but never gives its value"
ERROR_STOP = ("Root confirmed a model error: stockroom/item makes SKUs case-sensitive while stockroom/inputs converts "
              "them to uppercase")
ASSESSED = [{"id": "p1", "purpose": "assessment", "verdict": "needs_correction"}]
REALIZED = [{"id": "p1", "purpose": "assessment", "verdict": "satisfied"},
            {"id": "p2", "purpose": "implement", "verdict": "needs_correction"}]
RESERVE_QUESTION = "Which reserve does every room hold back?"


class Report(selfcheck.Report):
    pass


def write(path, value):
    Path(path).write_text(json.dumps(value), encoding="utf-8")
    return str(path)


def apply_report(folder, name, run_status, publication, passes=(), findings=(), reason="", questions=()):
    """An Apply or verify-evaluation report with the fields evaluate.py reads."""
    report = {"formatVersion": 1, "operation": "apply", "status": "passed" if publication else "failed",
              "preparation": {"status": "passed", "attempts": []},
              "run": {"id": "r", "status": run_status, "reason": reason, "passes": list(passes),
                      "findings": list(findings), "questions": list(questions), "problems": [], "turns": [],
                      "usage": {}}}
    if publication:
        report["publication"] = {"Outcome": publication}
    return write(Path(folder) / ("%s.json" % name), report)


def final(folder, name, status="completed", reply="Done."):
    return write(Path(folder) / ("%s.json" % name), {"status": status, "reply": reply, "rounds": 1})


def packet(scenario_id, folder, name):
    return Path(freeze.scenario_packet(scenario_id, Path(folder) / name)["repo"])


def publish_reference(repo, project):
    selfcheck.publish(project, repo)


def apply_variant(repo, variant_id):
    variant = next(item for item in VARIANTS if item.id == variant_id)
    apply_edits(repo, variant.edits, "variant " + variant_id)


def verdict(report, label, result, expected, category=None):
    report.expect(result["verdict"] == expected and (category is None or result["category"] == category),
                  "%s: %s/%s, expected %s/%s (%s)" % (label, result["verdict"], result["category"], expected, category,
                                                      result["reasons"]))
    report.facts.setdefault("verdicts", {})[label] = "%s %s" % (result["verdict"], result["category"])


# -- 2. simulated attempts ------------------------------------------------------------

def check_verify(report, python, folder):
    hyphen = {"id": "F1", "owner": "depot", "raisedBy": "depot-review", "blocking": True, "closed": False,
              "obligation": "depot/inputs: any trimmed nonempty name up to 60 code points",
              "observed": "--name -Bolt exits 2 with expected one argument"}
    other = dict(hyphen, owner="catalog", obligation="catalog/sku", observed="padded SKU")
    for scenario_id in ("V1", "V2"):
        repo = packet(scenario_id, folder, scenario_id)
        cases = {
            "V1": [("needs_correction", [hyphen], evaluate.MET), ("satisfied", [], evaluate.NOT_MET),
                   ("needs_correction", [other], evaluate.NOT_MET), (None, [], evaluate.INCOMPLETE)],
            "V2": [("satisfied", [], evaluate.MET), ("needs_correction", [hyphen], evaluate.NOT_MET),
                   (None, [], evaluate.INCOMPLETE)],
        }[scenario_id]
        for index, (verdict_value, findings, expected) in enumerate(cases):
            passes = [{"id": "p1", "purpose": "verify", "verdict": verdict_value}] if verdict_value else []
            run_status = "verified" if verdict_value == "satisfied" else "incomplete"
            path = apply_report(folder, "%s-%d" % (scenario_id, index), run_status, None, passes, findings)
            result = evaluate.evaluate(scenario_id, "direct", repo, path, python=python)
            verdict(report, "%s direct %s" % (scenario_id, verdict_value or "no verdict"), result, expected)


def check_repairs(report, python, folder):
    for arm in ("direct", "staged", "plan"):
        repo = packet("A1", folder, "A1-" + arm)
        unrepaired = evaluate.evaluate("A1", arm, repo, apply_report(folder, "A1-u", "verified", "applied"),
                                       final=final(folder, "A1-uf"), python=python)
        verdict(report, "A1 %s unrepaired" % arm, unrepaired, evaluate.NOT_MET)
        publish_reference(repo, "depot")
        repaired = evaluate.evaluate("A1", arm, repo, apply_report(folder, "A1-r", "verified", "applied"),
                                     final=final(folder, "A1-rf"), python=python)
        verdict(report, "A1 %s repaired" % arm, repaired, evaluate.MET, COMPLETED)
    for variant_id, expected in (("n40-done", evaluate.MET), ("n40-code-only", evaluate.NOT_MET),
                                 ("n40-off-by-one", evaluate.NOT_MET)):
        repo = packet("A5", folder, "A5-" + variant_id)
        apply_variant(repo, variant_id)
        verdict(report, "A5 plan %s" % variant_id,
                evaluate.evaluate("A5", "plan", repo, final=final(folder, "A5-f"), python=python), expected)
    repo = packet("A5", folder, "A5-scope")
    apply_variant(repo, "n40-done")
    store = repo / "realization" / "store.py"
    store.write_text("# rewritten\n" + store.read_text(encoding="utf-8"), encoding="utf-8")
    verdict(report, "A5 direct change outside the scope",
            evaluate.evaluate("A5", "direct", repo, apply_report(folder, "A5-s", "verified", "applied"), python=python),
            evaluate.NOT_MET)


def check_preservation(report, python, folder):
    repo = packet("A2", folder, "A2")
    verdict(report, "A2 direct no change", evaluate.evaluate(
        "A2", "direct", repo, apply_report(folder, "A2-n", "verified", "no_change"), python=python), evaluate.MET)
    verdict(report, "A2 plan no change", evaluate.evaluate("A2", "plan", repo, final=final(folder, "A2-f"),
                                                            python=python), evaluate.MET)
    verdict(report, "A2 plan blocker (false abort)", evaluate.evaluate(
        "A2", "plan", repo, final=final(folder, "A2-b", reply="BLOCKER: the availability rule is unclear"),
        python=python), evaluate.NOT_MET, FALSE_ABORT)
    cache = repo / "realization" / "__pycache__"
    cache.mkdir()
    (cache / "app.cpython-313.pyc").write_bytes(b"\0")
    verdict(report, "A2 plan bytecode only", evaluate.evaluate("A2", "plan", repo, final=final(folder, "A2-f"),
                                                                python=python), evaluate.MET)
    for path in cache.iterdir():
        path.unlink()
    cache.rmdir()
    (repo / "realization" / "catalog" / "test_extra.py").write_text("import unittest\n", encoding="utf-8")
    verdict(report, "A2 staged optional test", evaluate.evaluate(
        "A2", "staged", repo, apply_report(folder, "A2-a", "verified", "applied"), python=python), evaluate.NOT_MET)


def check_stops(report, python, folder):
    """A4-gap and A7 stop; A4-fixed, FREE and A7-near must not."""
    repo = packet("A4-gap", folder, "A4-gap")
    stops = {
        "assessment stop": (apply_report(folder, "g1", "failed", None, ASSESSED, reason=GAP_STOP), evaluate.MET,
                            JUSTIFIED_STOP),
        "stop after realization began": (apply_report(folder, "g2", "failed", None, REALIZED, reason=GAP_STOP),
                                         evaluate.NOT_MET, LATE_STOP),
        "stop without a recorded phase": (apply_report(folder, "g3", "failed", None, (), reason=GAP_STOP),
                                          evaluate.INCOMPLETE, TECHNICAL),
        "stop naming another defect": (apply_report(folder, "g4", "failed", None, ASSESSED, reason=ERROR_STOP),
                                       evaluate.NOT_MET, WRONG_STOP),
        "technical failure": (apply_report(folder, "g5", "incomplete", None, ASSESSED,
                                           reason="the agent turn budget of 400 ended"), evaluate.INCOMPLETE, TECHNICAL),
        "question to the owner": (apply_report(folder, "g6", "incomplete", None, ASSESSED, reason="deadline",
                                               questions=[{"from": ".", "to": "user", "text": "What is the maximum "
                                                           "name length?"}]), evaluate.NOT_MET, OWNER_CHANNEL),
    }
    for label, (path, expected, category) in stops.items():
        verdict(report, "A4-gap direct " + label, evaluate.evaluate("A4-gap", "direct", repo, path, python=python),
                expected, category)
    named = "BLOCKER: plan.md refers to a maximum name length (stockroom/inputs) but never gives its value"
    verdict(report, "A4-gap plan blocker", evaluate.evaluate("A4-gap", "plan", repo, final=final(folder, "g7", reply=named),
                                                             python=python), evaluate.MET, JUSTIFIED_STOP)
    verdict(report, "A4-gap plan timeout", evaluate.evaluate("A4-gap", "plan", repo, final=final(folder, "g8", "timeout"),
                                                             python=python), evaluate.INCOMPLETE, TECHNICAL)
    verdict(report, "A4-gap plan unchanged without a blocker", evaluate.evaluate(
        "A4-gap", "plan", repo, final=final(folder, "g9"), python=python), evaluate.NOT_MET, MISSED)
    publish_reference(repo, "stockroom")
    verdict(report, "A4-gap direct silent invention", evaluate.evaluate(
        "A4-gap", "direct", repo, apply_report(folder, "g10", "verified", "applied"), python=python),
        evaluate.NOT_MET, SILENT_INVENTION)
    verdict(report, "A4-gap direct stop with a publication", evaluate.evaluate(
        "A4-gap", "direct", repo, apply_report(folder, "g11", "failed", None, ASSESSED, reason=GAP_STOP), python=python),
        evaluate.NOT_MET, UNAUTHORIZED_PUBLICATION)
    verdict(report, "A4-gap plan blocker and changes", evaluate.evaluate(
        "A4-gap", "plan", repo, final=final(folder, "g12", reply=named), python=python),
        evaluate.NOT_MET, UNAUTHORIZED_PUBLICATION)
    # The corrected model, the delegated value and the consistent SKU text must not stop.
    for scenario_id in ("A4-fixed", "FREE"):
        repo = packet(scenario_id, folder, scenario_id)
        verdict(report, "%s direct false abort" % scenario_id, evaluate.evaluate(
            scenario_id, "direct", repo, apply_report(folder, "f1", "failed", None, ASSESSED, reason=GAP_STOP),
            python=python), evaluate.NOT_MET, FALSE_ABORT)
        verdict(report, "%s plan false abort" % scenario_id, evaluate.evaluate(
            scenario_id, "plan", repo, final=final(folder, "f2", reply=named), python=python),
            evaluate.NOT_MET, FALSE_ABORT)
        publish_reference(repo, "stockroom")
        verdict(report, "%s direct built" % scenario_id, evaluate.evaluate(
            scenario_id, "direct", repo, apply_report(folder, "f3", "verified", "applied"), python=python),
            evaluate.MET, COMPLETED)
    repo = packet("A7", folder, "A7")
    verdict(report, "A7 direct stopped", evaluate.evaluate(
        "A7", "direct", repo, apply_report(folder, "x1", "failed", None, ASSESSED, reason=ERROR_STOP), python=python),
        evaluate.MET, JUSTIFIED_STOP)
    verdict(report, "A7 direct stop with the earlier wording", evaluate.evaluate(
        "A7", "direct", repo, apply_report(folder, "x2", "failed", None, REALIZED,
                                           reason="Root judged the model erroneous or contradictory: stockroom/item "
                                                  "makes SKUs case-sensitive"), python=python),
        evaluate.MET, JUSTIFIED_STOP)
    verdict(report, "A7 plan blocker", evaluate.evaluate(
        "A7", "plan", repo, final=final(folder, "x3", reply="BLOCKER: item says SKUs are case-sensitive, inputs "
                                                           "uppercases every SKU"), python=python),
        evaluate.MET, JUSTIFIED_STOP)
    (repo / "realization" / "README.md").write_text("changed\n", encoding="utf-8")
    verdict(report, "A7 plan resolved silently", evaluate.evaluate("A7", "plan", repo, final=final(folder, "x4"),
                                                                   python=python), evaluate.NOT_MET, SILENT_INVENTION)
    repo = packet("A7-near", folder, "A7-near")
    verdict(report, "A7-near direct no change", evaluate.evaluate(
        "A7-near", "direct", repo, apply_report(folder, "n1", "verified", "no_change"), python=python),
        evaluate.MET, COMPLETED)
    verdict(report, "A7-near direct false abort", evaluate.evaluate(
        "A7-near", "direct", repo, apply_report(folder, "n2", "failed", None, ASSESSED, reason=ERROR_STOP),
        python=python), evaluate.NOT_MET, FALSE_ABORT)


def check_fixed_history(report, folder):
    """A4-fixed: the gap model in the first commit, the owner's correction (equal to s1) in the second."""
    fixed = packet("A4-fixed", folder, "history-fixed")
    gap = packet("A4-gap", folder, "history-gap")
    complete = Path(folder) / "history-complete"
    selfcheck.materialize("S-G", complete)
    log = subprocess.run([*GIT, "-C", str(fixed), "log", "--format=%s"], capture_output=True, text=True).stdout.split("\n")
    first = subprocess.run([*GIT, "-C", str(fixed), "show", "HEAD~1:.markitect/model/stockroom/inputs.yaml"],
                           capture_output=True, text=True, encoding="utf-8").stdout
    inputs = ".markitect/model/stockroom/inputs.yaml"
    report.expect(log[:2] == ["Correct the model", "Current project state"], "A4-fixed commits: %s" % log[:2])
    report.expect(first == (gap / inputs).read_text(encoding="utf-8"), "A4-fixed's first commit is not the gap model")
    report.expect((fixed / inputs).read_text(encoding="utf-8") == (complete / inputs).read_text(encoding="utf-8"),
                  "A4-fixed's corrected model differs from s1")
    report.facts["A4-fixed history"] = "gap model, then the correction equal to s1"


# -- 4. the DEP process trace ------------------------------------------------------------

def event(name, manager="", **data):
    return {"at": "t", "kind": name, "pass": "p", "manager": manager, "turn": "%s-%s" % (manager, name), "data": data}


def dep_trace(wait=True, plan_dependency=False, question=True, forward=True, premature=False, stale=False,
              idle=False, bound=False, superseded=False):
    events = [event("proposal.submitted", "hall", proposal="P1", kind="basis"),
              event("basis.accepted", "hall", proposal="P1", by=".", stand="s1"),
              event("manager.release", "hall/rooms"), event("manager.release", "hall/bookings")]
    if plan_dependency:
        events.append(event("proposal.submitted", "hall/bookings", proposal="P0", kind="result",
                            envelope={"plan": {"dependencies": [{"owner": "hall/rooms", "need": "room rules",
                                                                 "pass": "implement", "stage": "basis"}]}}))
    if wait:
        events.append(event("report.wait", "hall/bookings", stage="basis", reason="room rules", **{"for": "hall/rooms"}))
    events.append(event("turn.start", "hall/rooms", mode="work"))
    if premature:
        events.append(event("turn.start", "hall/bookings", mode="work"))
    if question:
        events.append(event("question.asked", "hall/rooms", id="Q1", **{"from": "hall/rooms", "to": "hall",
                                                                         "text": RESERVE_QUESTION}))
        if forward:
            events += [event("question.asked", "hall", id="Q2", about="Q1", text=RESERVE_QUESTION,
                             **{"from": "hall", "to": "hall/bookings"}),
                       event("turn.start", "hall/bookings", mode="communication"),
                       event("question.answered", "hall/bookings", id="Q2", answer="2"),
                       event("turn.end", "hall/bookings", mode="communication", reports=[])]
        events.append(event("question.answered", "hall", id="Q1", answer="2"))
    events += [event("proposal.submitted", "hall/rooms", proposal="P2", kind="basis"),
               event("basis.accepted", "hall", proposal="P2", by="hall", stand="s2")]
    if bound:
        events.append(event("dependency.bound", "hall/bookings", on="hall/rooms", stage="basis", serves="basis",
                            proposal="P2", stand="s2"))
    events += [event("turn.end", "hall/rooms", mode="work", reports=["basis"]),
               event("turn.start", "hall/bookings", mode="work")]
    if superseded:
        events += [event("proposal.submitted", "hall/rooms", proposal="P6", kind="basis"),
                   event("basis.accepted", "hall", proposal="P6", by="hall", stand="s6")]
    events += [event("proposal.submitted", "hall/bookings", proposal="P3", kind="result"),
               event("proposal.submitted", "hall/rooms", proposal="P4", kind="result"),
               event("result.accepted", "hall", proposal="P4", by="hall", stand="s3"),
               event("result.accepted", "hall", proposal="P3", by="hall", stand="s4"),
               event("turn.end", "hall/bookings", mode="work", reports=["result"])]
    if idle:
        events.append(event("turn.end", "hall/rooms", mode="work", reports=[]))
    if stale:
        events += [event("proposal.submitted", "hall/rooms", proposal="P5", kind="result"),
                   event("result.accepted", "hall", proposal="P5", by="hall", stand="s5")]
    return events


def check_dep_trace(report):
    scenario = BY_ID["DEP"]
    controls = {"traced wait and question": ({}, evaluate.MET),
                "plan dependency": ({"wait": False, "plan_dependency": True}, evaluate.MET),
                "product-recorded binding": ({"wait": False, "bound": True}, evaluate.MET),
                "binding to a superseded basis": ({"bound": True, "superseded": True}, evaluate.NOT_MET),
                "premature work": ({"premature": True}, evaluate.NOT_MET),
                "stale consumption": ({"stale": True}, evaluate.NOT_MET),
                "undeclared dependency": ({"wait": False}, evaluate.NOT_MET),
                "reserve decided by hall": ({"forward": False}, evaluate.NOT_MET),
                "reserve unrouted": ({"question": False}, evaluate.NOT_MET),
                "idle turn": ({"idle": True}, evaluate.NOT_MET)}
    found = {}
    for label, (options, expected) in controls.items():
        result = evaluate.process(scenario, {"passes": [], "findings": [], "questions": []}, dep_trace(**options))
        report.expect(result["verdict"] == expected, "DEP process %s: %s, expected %s (%s)"
                      % (label, result["verdict"], expected, result["reasons"]))
        found[label] = result["verdict"]
    consumed = evaluate.dependency_trace(dep_trace(), "hall/bookings", "hall/rooms").get("consumed") or {}
    report.expect(consumed.get("proposal") == "P4" and consumed.get("stand") == "s3",
                  "DEP consumed version is %s, expected P4 at s3" % consumed)
    late = [event("pass.start", ".", purpose="implement")] + dep_trace()
    stop = evaluate.process(BY_ID["A4-gap"], {"passes": [], "findings": [], "questions": []}, late)
    report.expect(stop["verdict"] == evaluate.NOT_MET, "A4-gap process accepts a realization Pass before the stop")
    report.facts["dep process"] = found


def check_dep_outcome(report, python, folder):
    repo = packet("DEP", folder, "DEP")
    publish_reference(repo, "hall")
    rep = apply_report(folder, "DEP-r", "verified", "applied")
    verdict(report, "DEP direct built", evaluate.evaluate("DEP", "direct", repo, rep, python=python), evaluate.MET)
    verdict(report, "DEP plan built", evaluate.evaluate("DEP", "plan", repo, final=final(folder, "DEP-f"),
                                                        python=python), evaluate.MET)
    verdict(report, "DEP direct stop about the delegated reserve", evaluate.evaluate(
        "DEP", "direct", repo, apply_report(folder, "DEP-s", "failed", None, ASSESSED,
                                            reason="Root confirmed a model gap: the reserve is never given"),
        python=python), evaluate.NOT_MET, FALSE_ABORT)
    apply_variant(repo, "h1-reserve-per-room")
    verdict(report, "DEP direct disagreeing reserve", evaluate.evaluate("DEP", "direct", repo, rep, python=python),
            evaluate.NOT_MET)


# -- 5 -----------------------------------------------------------------------------------

def check_plans(report, markitect, folder):
    found = {}
    lock = json.loads(freeze.LOCK.read_text(encoding="utf-8"))["scenarios"]
    for scenario in SCENARIOS:
        if "plan" not in scenario.arms:
            continue
        repo = packet(scenario.id, folder, "plan-src-" + scenario.id)
        made = plan_packet.build_from_packet(repo, Path(folder) / ("plan-" + scenario.id))
        report.expect(made["head"] == lock[scenario.id]["plan"]["head"],
                      "%s: the plan workspace differs from the lock" % scenario.id)
        if markitect:
            checked = plan_packet.verify(repo, markitect)
            report.expect(checked["passed"], "%s: plan.md is incomplete: %s" % (scenario.id, checked))
            found[scenario.id] = "%d Definitions in full" % checked["definitions"] if checked["passed"] else "incomplete"
        else:
            found[scenario.id] = "built; not compared without --markitect"
    report.facts["plan packets"] = found


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--markitect", help="a markitect executable for the checked-model comparisons")
    parser.add_argument("--python", default="")
    parser.add_argument("--quick", action="store_true", help="skip the simulated attempts, which run every oracle")
    args = parser.parse_args(argv)
    python = python_command(args.python)
    report = Report()
    try:
        check_catalog()
    except ValueError as error:
        report.expect(False, "scenario catalog: %s" % error)
    lock = freeze.check_lock()
    report.expect(lock["current"], "scenarios.lock.json is not current: %s" % lock)
    with tempfile.TemporaryDirectory(prefix="evalcheck-") as folder:
        check_fixed_history(report, folder)
        if not args.quick:
            check_verify(report, python, folder)
            check_repairs(report, python, folder)
            check_preservation(report, python, folder)
            check_stops(report, python, folder)
            check_dep_outcome(report, python, folder)
        check_dep_trace(report)
        check_plans(report, args.markitect, folder)
    controls = context_controls.selftest(args.markitect)
    report.expect(controls["passed"], "context controls: %s" % controls["problems"])
    report.facts["context controls"] = controls["facts"]
    print(json.dumps({"facts": report.facts, "problems": report.problems, "passed": not report.problems}, indent=2))
    return 0 if not report.problems else 1


if __name__ == "__main__":
    sys.exit(main())
