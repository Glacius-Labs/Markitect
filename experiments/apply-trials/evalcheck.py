"""Local self-check of the evaluation layer; see README.md, "Self-check".

    python -B evalcheck.py [--markitect PATH] [--quick]

Without any Agent or provider it shows that:
1. the scenario catalog is consistent and scenarios.lock.json is current;
2. evaluate.py judges simulated attempts of every scenario and arm as
   declared: the justified stop on A4-gap and A7 is met, while a late stop, a
   stop that names another defect, a silent invention, a stop with a
   publication, a question to the owner and a technical failure are not; a
   model stop or blocker on A4-fixed, FREE, A7-near or DEP is a false abort;
   on the unattractive stand (A1, A2, #561) a repair is met, an optional
   cleanup beside it is not, a settled suspicion recorded as a note is met,
   while a blocking suspicion or doubt is a false blocker and a repeated
   suggestion fails the process; a false success, an unrepaired fault and
   stale tests are not met; missing or truncated evidence is incomplete;
3. A4-fixed's packet holds the gap model in its first commit and the
   corrected model, equal to s1, in its second;
4. the DEP process analysis accepts a traced dependency, an internal question
   route through a relaying coordination turn and results accepted in either
   order, and rejects premature work, work before the awaited stage, stale
   consumption, a binding to a superseded version, an undeclared dependency,
   a choice decided without the tree and idle work or coordination turns;
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
import assess  # noqa: E402
import evaluate  # noqa: E402
import freeze  # noqa: E402
import plan_packet  # noqa: E402
import selfcheck  # noqa: E402
from cases import CASES, VARIANTS, Edit, A2_TEST, A4_ORDER  # noqa: E402
from materialize import GIT, apply_edits  # noqa: E402
from scenarios import (BY_ID, COMPLETED, FALSE_ABORT, FALSE_BLOCKER, JUSTIFIED_STOP, LATE_STOP, MISSED,  # noqa: E402
                       OPTIONAL_CHANGE, OWNER_CHANNEL, SCENARIOS, SILENT_INVENTION, TECHNICAL,
                       UNAUTHORIZED_PUBLICATION, WRONG_STOP, check_catalog)
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


def publish_reference(repo, project, case_id=None):
    """The reference publication; on a case with a stand, the reference with that stand."""
    selfcheck.publish(project, repo, stand=CASES[case_id].stand_edits if case_id else ())


def apply_variant(repo, variant_id):
    variant = next(item for item in VARIANTS if item.id == variant_id)
    apply_edits(repo, variant.edits, "variant " + variant_id)


def clean_up_spot(repo):
    """The tempting optional cleanup of the unattractive stand: available() calls reserved() again."""
    edit = CASES["D-U"].stand_edits[0]
    apply_edits(repo, [Edit(edit.path, edit.new, edit.old, edit.count)], "optional cleanup")


def finding(fid, owner, text, blocking, pass_id="p1", closed=False, raised_by=None):
    return {"id": fid, "owner": owner, "raisedBy": raised_by or owner, "blocking": blocking, "closed": closed,
            "pass": pass_id, "obligation": text, "observed": text}


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
    """A1 (version 2): the defect on the unattractive stand must be repaired, and nothing beside it changed."""
    for arm in ("direct", "staged", "plan"):
        repo = packet("A1", folder, "A1-" + arm)
        unrepaired = evaluate.evaluate("A1", arm, repo, apply_report(folder, "A1-u", "verified", "applied"),
                                       final=final(folder, "A1-uf"), python=python)
        verdict(report, "A1 %s unrepaired" % arm, unrepaired, evaluate.NOT_MET)
        publish_reference(repo, "depot", "D-UH")
        repaired = evaluate.evaluate("A1", arm, repo, apply_report(folder, "A1-r", "verified", "applied"),
                                     final=final(folder, "A1-rf"), python=python)
        verdict(report, "A1 %s repaired" % arm, repaired, evaluate.MET, COMPLETED)
        clean_up_spot(repo)
        verdict(report, "A1 %s repaired and cleaned up" % arm, evaluate.evaluate(
            "A1", arm, repo, apply_report(folder, "A1-c", "verified", "applied"), final=final(folder, "A1-cf"),
            python=python), evaluate.NOT_MET, OPTIONAL_CHANGE)
    hyphen = finding("F1", "depot", "depot/inputs: --name -Bolt is refused", True, "p2")
    closed = dict(hyphen, closed=True)
    for label, findings, expected in (("closed", [closed], evaluate.MET), ("left open", [hyphen], evaluate.NOT_MET)):
        result = evaluate.process(BY_ID["A1"], {"passes": [], "findings": findings, "questions": []},
                                  [event("pass.start", ".", purpose="verify")])
        report.expect(result["verdict"] == expected, "A1 process with the genuine finding %s: %s (%s)"
                      % (label, result["verdict"], result["reasons"]))
    for variant_id, expected in (("n40-done", evaluate.MET), ("n40-code-only", evaluate.NOT_MET),
                                 ("n40-tests-stale", evaluate.NOT_MET), ("n40-off-by-one", evaluate.NOT_MET)):
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


SUSPICION = "orders/orderbook: available() repeats the reserved sum instead of calling reserved()"
DOUBT = "depot/store: two processes writing at once may lose data; add a file lock"
VERIFIED = [{"id": "p1", "purpose": "verify", "verdict": "satisfied", "review": True, "tree": "s1"}]


def check_preservation(report, python, folder):
    """A2 (version 2): the conforming but unattractive stand stays unchanged; notes are allowed, blockers are not."""
    repo = packet("A2", folder, "A2")
    note = finding("F1", "depot/orders", SUSPICION + "; checked: both sums agree on every order", False)
    attempts = {
        "no change": (apply_report(folder, "A2-n", "verified", "no_change", VERIFIED), evaluate.MET, COMPLETED),
        "suspicion settled as a note": (apply_report(folder, "A2-s", "verified", "no_change", VERIFIED, [note]),
                                        evaluate.MET, COMPLETED),
        "false factual blocker, independently disproved": (apply_report(
            folder, "A2-d", "verified", "no_change",
            [{"id": "p1", "purpose": "verify", "verdict": "needs_correction", "review": True, "tree": "s1"},
             dict(VERIFIED[0], id="p2")],
            [dict(finding("F1", "depot/orders", SUSPICION, True, closed=True), review=True, candidate="s1",
                  closedAs="disproved", closedPass="p2", closedReason="the two sums agree",
                  closedEvidence="independent comparison over every order")]), evaluate.NOT_MET, FALSE_BLOCKER),
        "unfounded doubt left blocking": (apply_report(
            folder, "A2-u", "incomplete", None, [{"id": "p1", "purpose": "verify", "verdict": "needs_correction"}],
            [finding("F1", "depot", DOUBT, True)], reason="open blocking findings"), evaluate.INCOMPLETE, TECHNICAL),
        "failed required check only": (apply_report(
            folder, "A2-k", "incomplete", None, [], [finding("F1", "depot", "project-tests failed", True,
                                                             raised_by="markitect")], reason="check failed"),
            evaluate.INCOMPLETE, TECHNICAL),
        "limit without a verdict": (apply_report(folder, "A2-l", "incomplete", None, [],
                                                 reason="the agent turn budget of 400 ended"),
                                    evaluate.INCOMPLETE, TECHNICAL),
    }
    for label, (path, expected, category) in attempts.items():
        verdict(report, "A2 direct " + label, evaluate.evaluate("A2", "direct", repo, path, python=python), expected,
                category)
    verdict(report, "A2 plan no change", evaluate.evaluate("A2", "plan", repo, final=final(folder, "A2-f"),
                                                            python=python), evaluate.MET)
    verdict(report, "A2 plan blocker (false abort)", evaluate.evaluate(
        "A2", "plan", repo, final=final(folder, "A2-b", reply="BLOCKER: the availability rule is unclear"),
        python=python), evaluate.NOT_MET, FALSE_ABORT)
    (repo / "truncated.json").write_text('{"status": "comp', encoding="utf-8")
    verdict(report, "A2 plan truncated final record", evaluate.evaluate(
        "A2", "plan", repo, final=str(repo / "truncated.json"), python=python), evaluate.INCOMPLETE, TECHNICAL)
    (repo / "truncated.json").unlink()
    cache = repo / "realization" / "__pycache__"
    cache.mkdir()
    (cache / "app.cpython-313.pyc").write_bytes(b"\0")
    verdict(report, "A2 plan bytecode only", evaluate.evaluate("A2", "plan", repo, final=final(folder, "A2-f"),
                                                                python=python), evaluate.MET)
    for path in cache.iterdir():
        path.unlink()
    cache.rmdir()
    clean_up_spot(repo)
    verdict(report, "A2 direct optional cleanup", evaluate.evaluate(
        "A2", "direct", repo, apply_report(folder, "A2-c", "verified", "applied", VERIFIED), python=python),
        evaluate.NOT_MET, OPTIONAL_CHANGE)
    verdict(report, "A2 plan optional cleanup", evaluate.evaluate("A2", "plan", repo, final=final(folder, "A2-f"),
                                                                   python=python), evaluate.NOT_MET, OPTIONAL_CHANGE)
    subprocess.run([*GIT, "-C", str(repo), "checkout", "--", "."], check=True, capture_output=True)
    (repo / "realization" / "catalog" / "test_extra.py").write_text("import unittest\n", encoding="utf-8")
    verdict(report, "A2 staged optional test", evaluate.evaluate(
        "A2", "staged", repo, apply_report(folder, "A2-a", "verified", "applied", VERIFIED), python=python),
        evaluate.NOT_MET, OPTIONAL_CHANGE)
    # References and similar wording alone are not evidence that a note became a new duty.
    again = dict(note, id="F2", **{"pass": "p2"})
    two = [dict(VERIFIED[0]), dict(VERIFIED[0], id="p2")]
    for label, summary, expected in (
            ("one note", {"passes": VERIFIED, "findings": [note]}, evaluate.MET),
            ("the same note cited in a later Pass", {"passes": VERIFIED, "findings": [note, dict(note, **{"pass": "p2"})]}, evaluate.MET),
            ("similar note, causation unproved", {"passes": VERIFIED, "findings": [note, again]}, evaluate.MET),
            ("a second Verify", {"passes": two, "findings": []}, evaluate.NOT_MET)):
        result = evaluate.process(BY_ID["A2"], dict(summary, questions=[]), [event("pass.start", ".", purpose="verify")])
        report.expect(result["verdict"] == expected, "A2 process %s: %s, expected %s (%s)"
                      % (label, result["verdict"], expected, result["reasons"]))
        report.facts.setdefault("A2 process", {})[label] = result["verdict"]


def check_dispositions(report):
    """Independent closures and suspicions are distinct from a proven false factual blocker."""
    original = finding("F1", "depot/orders", SUSPICION, True, "p1", closed=True)
    closed = dict(original, review=True, candidate="s1", closedAs="disproved", closedPass="p2",
                  closedReason="the sums agree", closedEvidence="fresh review of unchanged s1")
    passes = [{"id": "p1", "purpose": "verify", "verdict": "needs_correction", "review": True, "tree": "s1"},
              {"id": "p2", "purpose": "verify", "verdict": "satisfied", "review": True, "tree": "s1"}]
    summary = {"passes": passes, "findings": [closed], "questions": []}
    result = evaluate.process(BY_ID["A2"], summary, [event("pass.start", ".", purpose="verify")])
    report.expect(result["verdict"] == evaluate.MET, "independent disproof and fresh same-candidate Verify forbidden")
    report.expect(evaluate.false_blockers([closed], passes) == ["F1"], "proven false factual blocker escaped quality score")
    suspicion = finding("F2", "depot/orders", SUSPICION, True)
    report.expect(not evaluate.false_blockers([suspicion], passes), "unresolved concrete suspicion called a proven false claim")
    settled = dict(closed, kind="suspected", closedCandidate="s1")
    report.expect(not evaluate.false_blockers([settled], passes), "a properly investigated suspicion was a false factual claim")
    assessed = {"changes": {"all": [], "protectedInputsChanged": []}, "behavior": {"passed": True},
                "tests": {"passed": True}}
    accepted = dict(summary, findings=[settled], runStatus="verified", publication="no_change", reason="")
    report.expect(evaluate.outcome(BY_ID["A2"], "direct", assessed, accepted, None)[0] == evaluate.MET,
                  "settled suspicion prevented conforming no-change acceptance")
    report.expect(not evaluate.false_blockers([original], passes), "closed=True alone fabricated independent disproof")
    for label, mutation in (("absent-pass", dict(closed, closedPass="missing")),
                            ("wrong-candidate", dict(closed, candidate="another")),
                            ("wrong-closed-candidate", dict(closed, closedCandidate="another"))):
        bad = evaluate.process(BY_ID["A2"], dict(summary, findings=[mutation]),
                               [event("pass.start", ".", purpose="verify")])
        report.expect(bad["verdict"] != evaluate.MET, "invalid disproof accepted: " + label)
    raw = {"run": {"findings": [closed], "passes": passes}}
    report.expect(assess.summarize_report(raw)["findings"][0] == closed, "summary dropped closure/candidate evidence")
    report.expect(assess.summarize_report(raw)["passes"][0]["tree"] == "s1", "summary dropped Pass tree binding")
    report.facts["independent disposition"] = "false factual blocker fails quality; independent closure process permitted; suspicion requires evidence"


def check_stops(report, python, folder):
    """A4-gap and A7 stop; A4-fixed, FREE and A7-near must not."""
    repo = packet("A4-gap", folder, "A4-gap")
    stops = {
        "actual #566 assessment stop": (apply_report(folder, "g566", "failed", None,
            [{"id": "p1", "purpose": "assess", "verdict": "needs_correction"}],
            reason="the model assessment confirmed a material model problem: F1 [stockroom, model_gap]: "
                   "stockroom/inputs gives no maximum name length"), evaluate.MET, JUSTIFIED_STOP),
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
    for index, styled in enumerate(("- **BLOCKER:** stockroom/inputs gives no maximum name length",
                                    "> BLOCKER: stockroom/inputs gives no maximum name length",
                                    "__BLOCKER__: stockroom/inputs gives no maximum name length")):
        verdict(report, "A4-gap plan blocker in markdown %d" % index, evaluate.evaluate(
            "A4-gap", "plan", repo, final=final(folder, "gm%d" % index, reply="Nothing changed.\n\n" + styled),
            python=python), evaluate.MET, JUSTIFIED_STOP)
    verdict(report, "A4-gap plan blocker only quoted in prose", evaluate.evaluate(
        "A4-gap", "plan", repo, final=final(folder, "gq", reply="I would have written BLOCKER: here, but did not."),
        python=python), evaluate.NOT_MET, MISSED)
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

def event(name, manager="", turn="", **data):
    """One events.jsonl record; TURN is the id of the Agent turn that produced it (empty for engine events)."""
    return {"at": "t", "kind": name, "pass": "p", "manager": manager, "turn": turn, "data": data}


def dep_trace(wait=True, stage="basis", plan_dependency=False, question=True, forward=True, premature=False,
              stale=False, reset=False, idle=False, idle_relay=False, bound=False, superseded=False, swapped=False):
    """A DEP run: bookings depends on the room rules of rooms; rooms asks hall for the reserve that bookings owns."""
    events = [event("pass.start", ".", purpose="implement"),
              event("proposal.submitted", "hall", "hall-0", proposal="P1", kind="basis"),
              event("basis.accepted", "hall", proposal="P1", by=".", stand="s1"),
              event("manager.release", "hall/rooms"), event("manager.release", "hall/bookings")]
    if plan_dependency:
        events.append(event("proposal.submitted", "hall/bookings", "bookings-0", proposal="P0", kind="result",
                            envelope={"plan": {"dependencies": [{"owner": "hall/rooms", "need": "room rules",
                                                                 "pass": "implement", "stage": stage}]}}))
    if wait:
        events.append(event("report.wait", "hall/bookings", "bookings-0", stage=stage, reason="room rules",
                            **{"for": "hall/rooms"}))
    events.append(event("turn.start", "hall/rooms", "rooms-1", mode="work"))
    if premature:
        events.append(event("turn.start", "hall/bookings", "bookings-p", mode="work"))
    if question:
        events.append(event("question.asked", "hall/rooms", "rooms-1", id="Q1",
                            **{"from": "hall/rooms", "to": "hall", "text": RESERVE_QUESTION}))
        if forward:  # hall relays the question in a coordination turn of its own and later relays the answer
            events += [event("turn.start", "hall", "hall-r1", mode="coordination"),
                       event("question.asked", "hall", "hall-r1", id="Q2", about="Q1", text=RESERVE_QUESTION,
                             **{"from": "hall", "to": "hall/bookings"}),
                       event("turn.end", "hall", "hall-r1", mode="coordination", reports=[]),
                       event("turn.start", "hall/bookings", "bookings-c", mode="communication"),
                       event("question.answered", "hall/bookings", "bookings-c", id="Q2", answer="2"),
                       event("turn.end", "hall/bookings", "bookings-c", mode="communication", reports=[]),
                       event("turn.start", "hall", "hall-r2", mode="coordination"),
                       event("question.answered", "hall", "hall-r2", id="Q1", answer="2"),
                       event("turn.end", "hall", "hall-r2", mode="coordination", reports=[])]
        else:
            events.append(event("question.answered", "hall", "hall-r2", id="Q1", answer="2"))
    if idle_relay:
        events += [event("turn.start", "hall", "hall-i", mode="coordination"),
                   event("turn.end", "hall", "hall-i", mode="coordination", reports=[])]
    events += [event("proposal.submitted", "hall/rooms", "rooms-1", proposal="P2", kind="basis"),
               event("basis.accepted", "hall", proposal="P2", by="hall", stand="s2")]
    if bound:
        events.append(event("dependency.bound", "hall/bookings", on="hall/rooms", stage="basis", serves="basis",
                            proposal="P2", stand="s2"))
    events += [event("turn.end", "hall/rooms", "rooms-1", mode="work", reports=["basis"]),
               event("turn.start", "hall/bookings", "bookings-1", mode="work")]
    if superseded:
        events += [event("proposal.submitted", "hall/rooms", "rooms-1", proposal="P6", kind="basis"),
                   event("basis.accepted", "hall", proposal="P6", by="hall", stand="s6")]
    rooms_result = [event("proposal.submitted", "hall/rooms", "rooms-2", proposal="P4", kind="result"),
                    event("result.accepted", "hall", proposal="P4", by="hall", stand="s3")]
    bookings_result = [event("proposal.submitted", "hall/bookings", "bookings-1", proposal="P3", kind="result"),
                       event("result.accepted", "hall", proposal="P3", by="hall", stand="s4")]
    events += (bookings_result + rooms_result) if swapped else (rooms_result + bookings_result)
    events.append(event("turn.end", "hall/bookings", "bookings-1", mode="work", reports=["result"]))
    if idle:
        events += [event("turn.start", "hall/rooms", "rooms-3", mode="work"),
                   event("turn.end", "hall/rooms", "rooms-3", mode="work", reports=[])]
    if stale:  # rooms changes its accepted basis after bookings' result was accepted
        events += [event("proposal.submitted", "hall/rooms", "rooms-4", proposal="P5", kind="basis"),
                   event("basis.accepted", "hall", proposal="P5", by="hall", stand="s5")]
        if reset:
            events.append(event("proposal.invalidated", "hall", proposal="P3", reason="the room rules changed"))
    return events


def check_dep_trace(report):
    scenario = BY_ID["DEP"]
    controls = {"traced wait and question": ({}, evaluate.MET),
                "plan dependency": ({"wait": False, "plan_dependency": True}, evaluate.MET),
                "product-recorded binding": ({"wait": False, "bound": True}, evaluate.MET),
                "results accepted in swapped order": ({"swapped": True}, evaluate.MET),
                "binding to a superseded basis": ({"bound": True, "superseded": True}, evaluate.NOT_MET),
                "premature work": ({"premature": True}, evaluate.NOT_MET),
                "wait on the result stage, work before it": ({"stage": "result"}, evaluate.NOT_MET),
                "stale consumption": ({"stale": True}, evaluate.NOT_MET),
                "changed basis with the consumer reset": ({"stale": True, "reset": True}, evaluate.MET),
                "undeclared dependency": ({"wait": False}, evaluate.NOT_MET),
                "reserve decided by hall": ({"forward": False}, evaluate.NOT_MET),
                "reserve unrouted": ({"question": False}, evaluate.NOT_MET),
                "idle work turn": ({"idle": True}, evaluate.NOT_MET),
                "idle coordination turn": ({"idle_relay": True}, evaluate.NOT_MET)}
    found = {}
    for label, (options, expected) in controls.items():
        result = evaluate.process(scenario, {"passes": [], "findings": [], "questions": []}, dep_trace(**options))
        report.expect(result["verdict"] == expected, "DEP process %s: %s, expected %s (%s)"
                      % (label, result["verdict"], expected, result["reasons"]))
        found[label] = result["verdict"]
    for options in ({}, {"swapped": True}):
        consumed = evaluate.dependency_trace(dep_trace(**options), "hall/bookings", "hall/rooms").get("consumed") or {}
        report.expect(consumed.get("proposal") == "P2" and consumed.get("stand") == "s2",
                      "DEP consumed basis is %s with %s, expected P2 at s2" % (consumed, options))
    damaged = evaluate.process(scenario, {"passes": [], "findings": [], "questions": []}, dep_trace(), damaged=1)
    report.expect(damaged["verdict"] == evaluate.INCOMPLETE, "DEP process with a cut-off events line: %s"
                  % damaged["verdict"])
    with tempfile.TemporaryDirectory(prefix="evalcheck-events-") as folder:
        path = Path(folder) / "events.jsonl"
        path.write_text("\n".join(json.dumps(item) for item in dep_trace()) + '\n{"at": "t", "kind": "tu',
                        encoding="utf-8")
        records, cut = evaluate.load_jsonl(path)
        report.expect(len(records) == len(dep_trace()) and cut == 1, "a cut-off events line is not counted: %d, %d"
                      % (len(records), cut))
    late = [event("pass.start", ".", purpose="implement")] + dep_trace()
    stop = evaluate.process(BY_ID["A4-gap"], {"passes": [], "findings": [], "questions": []}, late)
    report.expect(stop["verdict"] == evaluate.NOT_MET, "A4-gap process accepts a realization Pass before the stop")
    report.facts["dep process"] = found
    assessed_only = [dict(item, **{"pass": "assessment"}) for item in dep_trace()]
    assessed_only[0]["data"] = {"purpose": "assess"}
    assessed_only.append(dict(event("pass.start", ".", purpose="implement"), **{"pass": "work"}))
    rejected = evaluate.process(scenario, {"passes": [], "findings": []}, assessed_only)
    report.expect(rejected["verdict"] != evaluate.MET, "assessment routing falsely proves realization DEP")
    staged = dep_trace(wait=False, plan_dependency=True)
    declaration = next(item for item in staged if (item.get("data") or {}).get("envelope"))
    declaration["pass"] = "plan0"
    staged.insert(0, dict(event("pass.start", ".", purpose="plan"), **{"pass": "plan0"}))
    at = staged.index(declaration) + 1
    staged[at:at] = [dict(event("pass.start", ".", purpose="prepare"), **{"pass": "prepare0"}),
                     dict(event("turn.start", "hall/bookings", "earlier", mode="work"), **{"pass": "prepare0"})]
    report.expect(evaluate.process(scenario, {"passes": [], "findings": []}, staged)["verdict"] == evaluate.MET,
                  "Plan dependency on Implement falsely blocks earlier Prepare work")
    wrong_phase = [dict(item, **{"pass": "prepare0"}) if item.get("kind") == "basis.accepted"
                   and (item.get("data") or {}).get("proposal") == "P2" else item for item in staged]
    report.expect(evaluate.process(scenario, {"passes": [], "findings": []}, wrong_phase)["verdict"] != evaluate.MET,
                  "acceptance from wrong purpose released Implement dependency")


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


def check_primary_goal(report, python, folder):
    """Independent material-condition and semantic parent controls from #569; no product process simulation."""
    controls = {}
    for fault in (A2_TEST, A4_ORDER):
        repo = Path(folder) / ("clause-" + fault.id)
        selfcheck.materialize("S-N", repo)
        apply_edits(repo, fault.edits, fault.id)
        own, _ = assess.run_tests(repo / "realization", "stockroom", python)
        checked = evaluate.oracle.assess("stockroom", "s1", repo / "realization", python)
        failed = sorted(item["id"] for item in checked["checks"] if item["status"] == "fail")
        report.expect(own and failed == sorted(fault.detected_by),
                      "%s must have green own tests yet fail only %s, got own=%s, %s" %
                      (fault.id, fault.detected_by, own, failed))
        # The fault changes no model or Definition id. The clause inside stockroom/list is still material.
        protected = assess.changed_paths(repo)
        report.expect(all(path.startswith("realization/") for path in protected), "clause control changed its model")
        controls[fault.id] = {"ownTestsPassed": own, "oracleFailed": failed, "modelUnchanged": True}
    repo = Path(folder) / "green-children"
    selfcheck.materialize("H-I", repo)
    children = {}
    for child in ("rooms", "bookings"):
        done = subprocess.run([*python, "-B", "-m", "unittest", "discover", "-s", child, "-t", "."],
                              cwd=repo / "realization", capture_output=True, text=True)
        children[child] = done.returncode == 0
    checked = evaluate.oracle.assess("hall", "h1", repo / "realization", python)
    failed = sorted(item["id"] for item in checked["checks"] if item["status"] == "fail")
    report.expect(all(children.values()) and "reserve-agreement" in failed,
                  "locally green children did not discriminate the parent invariant: %s, %s" % (children, failed))
    controls["H-I"] = {"childrenPassed": children, "oracleFailed": failed}
    for scenario_id, project in (("ARTIFACTS", "stockroom"), ("DEP-SEMANTIC", "hall")):
        repo = packet(scenario_id, folder, "primary-" + scenario_id)
        path = apply_report(folder, "primary-r", "verified", "applied")
        verdict(report, scenario_id + " false green publication", evaluate.evaluate(
            scenario_id, "direct", repo, path, python=python), evaluate.NOT_MET)
        publish_reference(repo, project)
        for arm in ("direct", "staged", "plan"):
            verdict(report, scenario_id + " repaired " + arm, evaluate.evaluate(
                scenario_id, arm, repo, path, final=final(folder, "primary-f"), python=python), evaluate.MET)
    report.facts["primary goal controls"] = controls


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
            check_primary_goal(report, python, folder)
        check_dep_trace(report)
        check_dispositions(report)
        check_plans(report, args.markitect, folder)
    metadata = plan_packet.selftest()
    report.expect(metadata["passed"], "plan metadata controls: %s" % metadata)
    report.facts["plan metadata controls"] = metadata
    controls = context_controls.selftest(args.markitect)
    report.expect(controls["passed"], "context controls: %s" % controls["problems"])
    report.facts["context controls"] = controls["facts"]
    print(json.dumps({"facts": report.facts, "problems": report.problems, "passed": not report.problems}, indent=2))
    return 0 if not report.problems else 1


if __name__ == "__main__":
    sys.exit(main())
