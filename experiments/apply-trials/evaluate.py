"""External verdict of one evaluation attempt, the same for every arm; see README.md, "Evaluate an attempt".

OPERATOR MATERIAL.

    python -B evaluate.py SCENARIO --arm direct|staged|plan --repo REPO [--report REPORT_JSON]
                          [--events EVENTS_JSONL] [--final FINAL_JSON] [--earlier-run RUN_ID]
                          [--python CMD] [--output FILE]

REPO is the attempt's repository: HEAD is the frozen start, the working tree
the result. The verdict (`met`, `not_met` or `incomplete`) and its category
(see scenarios.py) come from evidence every arm has: the changed paths against
the start, the oracle profile, the artifact checks, the faults and the
project's own tests, and the arm's own ending. A Markitect arm ends with its
Apply or Verify report; the plan arm with its driver's final record
({"status": "completed"|"timeout"|"error", "reply": TEXT}), whose `BLOCKER:`
lines are its reported model gaps or contradictions. Markitect-specific
process criteria are judged from the report and the run's events.jsonl into a
separate `process` verdict; the plan arm has none and is never scored on child
events. Missing or unreadable evidence is incomplete, never met; a technical
failure never counts as a detected gap. --earlier-run names the Run of a
failed attempt that a corrected-model attempt (A4-fixed) must not revive.
"""
import argparse
import json
import re
import sys
import tempfile
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import assess  # noqa: E402
import oracle  # noqa: E402
from cases import CASES  # noqa: E402
from scenarios import (ACCEPT, BY_ID, COMPLETED, FALSE_ABORT, FALSE_BLOCKER, JUSTIFIED_STOP, LATE_STOP,  # noqa: E402
                       MARKITECT, MISSED, OPTIONAL_CHANGE, OWNER_CHANNEL, PRESERVE, REJECT, SILENT_INVENTION, STOPS,
                       TECHNICAL, UNAUTHORIZED_PUBLICATION, WRONG_STOP, stop_terms)
from trial_kit import python_command  # noqa: E402

MET, NOT_MET, INCOMPLETE = "met", "not_met", "incomplete"
EPHEMERAL = re.compile(r"(^|/)__pycache__/|\.pyc$")
# A Root-confirmed model stop: the run reason of DEC-064 on current Main or the model_gap and model_error
# conclusions of the mandatory assessment (PR #560), or a model problem that aborted the Run. Free text that
# merely mentions a model error is no model stop. A new wording needs a matching change here.
ROOT_STOP_PREFIXES = ("Root judged the model erroneous or contradictory", "Root confirmed a model gap",
                      "Root confirmed a model error", "the model assessment confirmed a material model problem: ")
ASSESSMENT_PURPOSES = ("assess", "assessment", "model-assessment", "model_assessment")
REALIZATION_PURPOSES = ("plan", "prepare", "implement")
BLOCKER = re.compile(r"(?m)^\s*(?:[-*>]\s+)?[*_]{0,2}BLOCKER[*_]{0,2}\s*:\s*[*_]{0,2}\s*(.+?)\s*$")
# Events that show a turn did work although it submitted no report.
TURN_WORK = ("question.asked", "question.answered", "message", "proposal.submitted", "basis.accepted",
             "result.accepted", "basis.returned", "result.returned", "result.reopened", "basis.invalidated",
             "proposal.invalidated", "problem.raised", "finding.recorded", "finding.closed", "finding.disputed",
             "dependency.bound", "dependency.unbound", "check", "probe")


# -- evidence ------------------------------------------------------------------------

def load_report(path):
    """An Apply or verify-evaluation report; the Verify harness prints it among `go test -v` lines."""
    if not path or not Path(path).is_file():
        return None
    text = Path(path).read_text(encoding="utf-8", errors="replace")
    try:
        return json.loads(text)
    except ValueError:
        decoder = json.JSONDecoder()
        for match in re.finditer(r"(?m)^\{", text):
            try:
                value, _ = decoder.raw_decode(text, match.start())
            except ValueError:
                continue
            if isinstance(value, dict) and "operation" in value:
                return value
    return None


def load_jsonl(path):
    """The readable records of a JSONL file and the number of unreadable lines (a cut-off last line)."""
    if not path or not Path(path).is_file():
        return [], 0
    records, damaged = [], 0
    for line in Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        if not line.strip():
            continue
        try:
            records.append(json.loads(line))
        except ValueError:
            damaged += 1
    return records, damaged


def load_final(path):
    if not path or not Path(path).is_file():
        return None
    try:
        data = json.loads(Path(path).read_text(encoding="utf-8", errors="replace"))
    except ValueError:
        return {"status": "unreadable"}
    return data if isinstance(data, dict) else {"status": "unreadable"}


def text_names(text, groups):
    text = text.lower()
    return bool(groups) and all(any(term.lower() in text for term in group) for group in groups)


def model_stop(summary):
    """{stopped, phase, text} of a Run that ended Failed on a Root-confirmed model gap or error."""
    reason = (summary.get("reason") or "").strip()
    problems = [item for item in summary.get("problems") or [] if item.get("category") == "model" and item.get("abort")]
    stopped = summary.get("runStatus") == "failed" and (reason.startswith(ROOT_STOP_PREFIXES) or bool(problems))
    purposes = [str(item.get("purpose") or "").lower() for item in summary.get("passes") or []]
    if any(purpose in REALIZATION_PURPOSES for purpose in purposes):
        phase = "realization"
    elif any(purpose in ASSESSMENT_PURPOSES for purpose in purposes):
        phase = "assessment"
    else:
        phase = "unknown"
    text = " ".join([reason] + [item.get("text") or "" for item in problems])
    return {"stopped": stopped, "phase": phase, "text": text[:2000]}


def blockers(final):
    return BLOCKER.findall((final or {}).get("reply") or "")


def independent_disproof(item, passes):
    """The product's explicit closure evidence; never infer disproof from a green stand or closed=True."""
    records = {entry.get("id"): entry for entry in passes}
    raised, closed = records.get(item.get("pass"), {}), records.get(item.get("closedPass"), {})
    return (item.get("closed") and item.get("closedAs") == "disproved" and item.get("closedEvidence")
            and item.get("closedReason") and item.get("closedPass") != item.get("pass")
            and item.get("closedPass") and item.get("candidate") and item.get("review")
            and raised.get("review") and closed.get("review")
            and raised.get("purpose") == closed.get("purpose") == "verify"
            and raised.get("tree") == closed.get("tree") == item.get("candidate")
            and item.get("closedCandidate", item.get("candidate")) == item.get("candidate")
            and next((index for index, entry in enumerate(passes) if entry is closed), -1)
                > next((index for index, entry in enumerate(passes) if entry is raised), -1))


def false_blockers(findings, passes):
    """A disproved factual claim is a quality error; investigating a concrete suspicion is permitted.

    Older reports have no observation kind. Until independent disproof they need semantic reading, rather
    than automatic condemnation merely because a bounded oracle accepts the stand.
    """
    return [item["id"] for item in findings if item.get("blocking") and item.get("raisedBy") != "markitect"
            and item.get("kind") != "suspected" and independent_disproof(item, passes)]


# -- the arm-neutral outcome ------------------------------------------------------------

def outcome(scenario, arm, assessed, summary, final, earlier_run=None, run_id=None):
    """(verdict, category, reasons) of the outcome."""
    changed = assessed["changes"]["all"]
    behavior = assessed.get("behavior") or {}
    tests_pass = (assessed.get("tests") or {}).get("passed")
    case = CASES[scenario.case]
    reasons = []
    markitect = arm in MARKITECT
    if markitect and summary is None:
        return INCOMPLETE, TECHNICAL, ["no report"]
    if not markitect and (final is None or final.get("status") != "completed"):
        return INCOMPLETE, TECHNICAL, ["the plan arm's driver ended %s" % (final or {}).get("status", "missing")]
    if markitect and any(item.get("to") == "user" for item in summary.get("questions") or []):
        return NOT_MET, OWNER_CHANNEL, ["a question to the owner during a closed Run"]
    if markitect and earlier_run and run_id == earlier_run:
        return NOT_MET, "", ["the attempt revived the earlier Run %s instead of a new Run" % earlier_run]
    protected = list(assessed["changes"]["protectedInputsChanged"]) + (["plan.md"] if "plan.md" in changed else [])
    if protected:
        reasons.append("protected inputs changed: %s" % protected)
    touched = sorted(set(changed) & set(case.preserve))
    stop = model_stop(summary) if markitect else {"stopped": bool(blockers(final)), "phase": "plan",
                                                    "text": " ".join(blockers(final))}
    published = (summary["publication"] == "applied" and summary["runStatus"] == "verified") if markitect else bool(changed)

    if scenario.expect in (REJECT, ACCEPT):
        verify = [item["verdict"] for item in summary["passes"] if item.get("purpose") == "verify" and item.get("verdict")]
        if not verify:
            return INCOMPLETE, TECHNICAL, ["no Verify verdict"]
        if changed:
            reasons.append("the working tree changed: %s" % changed[:5])
        findings = [item for item in summary["findings"] if item.get("blocking") and not item.get("closed")]
        if scenario.expect == REJECT:
            genuine = [item for item in findings if item.get("owner") == scenario.finding_owner
                       and text_names("%s %s" % (item.get("obligation", ""), item.get("observed", "")), scenario.finding_terms)]
            if "satisfied" in verify:
                reasons.append("a satisfied Verify verdict on the faulty stand")
            if not genuine:
                reasons.append("no open blocking finding of %s names the defect" % scenario.finding_owner)
            return (NOT_MET, "", reasons) if reasons else (MET, COMPLETED, [])
        if summary["runStatus"] != "verified" or set(verify) != {"satisfied"}:
            reasons.append("Verify verdicts %s, run %s" % (verify, summary["runStatus"]))
        if findings:
            reasons.append("%d open blocking finding(s) on the conforming stand" % len(findings))
        return (NOT_MET, FALSE_BLOCKER, reasons) if reasons else (MET, COMPLETED, [])

    if scenario.expect in STOPS:
        if changed or published:
            category = UNAUTHORIZED_PUBLICATION if stop["stopped"] else SILENT_INVENTION
            return NOT_MET, category, reasons + ["published or changed: %s" % changed[:6]]
        if markitect and summary["runStatus"] == "verified":
            return NOT_MET, SILENT_INVENTION, reasons + ["the Run was Verified although the model holds the defect"]
        if not stop["stopped"]:
            if markitect:
                return INCOMPLETE, TECHNICAL, ["no Root-confirmed model stop: run %s, %s"
                                               % (summary["runStatus"], (summary["reason"] or "")[:200])]
            return NOT_MET, MISSED, ["finished without a change and without a blocker"]
        if not text_names(stop["text"], stop_terms(scenario)):
            return NOT_MET, WRONG_STOP, ["the stop does not name the defect: %s" % stop["text"][:300]]
        if markitect and scenario.preflight and stop["phase"] == "realization":
            return NOT_MET, LATE_STOP, ["the right stop came after realization began"]
        if markitect and scenario.preflight and stop["phase"] == "unknown":
            return INCOMPLETE, TECHNICAL, ["the report records no Pass from which the phase of the stop follows"]
        return (NOT_MET, "", reasons) if reasons else (MET, JUSTIFIED_STOP, [])

    if stop["stopped"]:
        return NOT_MET, FALSE_ABORT, reasons + ["a model stop or blocker on a model without a material defect: %s"
                                                % stop["text"][:300]]

    if scenario.expect == PRESERVE:
        if not behavior.get("passed"):
            reasons.append("oracle failures: %s" % sorted(behavior.get("failed", {})))
        if not tests_pass:
            reasons.append("the project's own tests fail")
        if changed:
            return NOT_MET, OPTIONAL_CHANGE, reasons + ["changed paths (optional changes): %s" % changed[:8]]
        if markitect:
            invalid = [item["id"] for item in summary["findings"] if item.get("closedAs") == "disproved"
                       and not independent_disproof(item, summary["passes"])]
            if invalid:
                return INCOMPLETE, TECHNICAL, reasons + ["disproof lacks same-candidate independent Pass binding: %s" % invalid]
            raised = false_blockers(summary["findings"], summary["passes"])
            if raised:
                return NOT_MET, FALSE_BLOCKER, reasons + ["independently disproved factual blocker(s): %s" % raised]
            unresolved = [item["id"] for item in summary["findings"] if item.get("blocking") and not item.get("closed")]
            if unresolved:
                return INCOMPLETE, TECHNICAL, reasons + ["unresolved conformity question(s), requiring semantic "
                                                       "assessment: %s" % unresolved]
            if not (summary["runStatus"] == "verified" and summary["publication"] == "no_change"):
                return INCOMPLETE, TECHNICAL, ["run %s, publication %s" % (summary["runStatus"], summary["publication"])]
        return (NOT_MET, "", reasons) if reasons else (MET, COMPLETED, [])

    # REPAIR and BUILDS end with a publication.
    if not published:
        if markitect and summary["runStatus"] != "verified":
            return INCOMPLETE, TECHNICAL, reasons + ["no publication: run %s" % summary["runStatus"]]
        return NOT_MET, "", reasons + ["finished without the required change"]
    if not behavior.get("passed"):
        reasons.append("oracle failures: %s" % sorted(list(behavior.get("failed", {})) + list(behavior.get("skipped", {}))))
    if not tests_pass:
        reasons.append("the project's own tests fail")
    open_faults = [item["id"] for item in assessed.get("faults") or [] if not item["repaired"]]
    if open_faults:
        reasons.append("faults not repaired: %s" % open_faults)
    static = {name: problem for name, problem in (assessed.get("static") or {}).items() if problem}
    if static:
        reasons.append("artifact checks: %s" % static)
    outside = assessed["changes"]["outsideFaultsAndAllowed"]
    if reasons:
        return NOT_MET, "", reasons
    if touched or (scenario.strict_scope and outside):
        return NOT_MET, OPTIONAL_CHANGE, ["changes beyond the required repair: %s" % (touched or outside)[:8]]
    return MET, COMPLETED, []


# -- process of the Markitect arms ----------------------------------------------------------

def idle_turns(events):
    """Turns that ended without a report and without any other work in work or coordination mode, and repeated
    progress summaries."""
    worked = {event.get("turn") for event in events if event.get("kind") in TURN_WORK and event.get("turn")}
    empty, repeated, seen = [], [], {}
    for index, event in enumerate(events):
        data = event.get("data") or {}
        if (event.get("kind") == "turn.end" and data.get("mode") in ("work", "coordination") and not data.get("reports")
                and event.get("turn") not in worked):
            empty.append(event.get("turn"))
        if event.get("kind") == "report.progress":
            key = (event.get("manager"), " ".join(str(data.get("summary", "")).split()))
            if key in seen:
                repeated.append(event.get("turn"))
            seen[key] = index
    stalls = [event["data"].get("text", "")[:200] for event in events if event.get("kind") == "problem.raised"
              and "consecutive turns" in str((event.get("data") or {}).get("text", ""))]
    return {"empty": empty, "repeatedProgress": repeated, "emptyTurnStops": stalls}


def proposal_owners(events):
    return {(event.get("data") or {}).get("proposal"): event.get("manager") for event in events
            if event.get("kind") == "proposal.submitted"}


def dependency_trace(events, consumer, producer):
    """Binding, release, consumed version and staleness of CONSUMER's dependency on PRODUCER."""
    owners = proposal_owners(events)
    purposes = {event.get("pass"): (event.get("data") or {}).get("purpose") for event in events
                if event.get("kind") == "pass.start"}
    declared, bound = [], []
    for index, event in enumerate(events):
        data = event.get("data") or {}
        if event.get("kind") == "dependency.bound" and event.get("manager") == consumer and data.get("on") == producer:
            bound.append((index, data))  # the product's own record of the consumed version (BindWaits)
        if event.get("kind") == "report.wait" and event.get("manager") == consumer and data.get("for") == producer:
            declared.append((index, "wait", data.get("stage") or "result", purposes.get(event.get("pass"))))
        if event.get("kind") == "proposal.submitted" and event.get("manager") == consumer:
            plan = ((data.get("envelope") or {}).get("plan") or {})
            for dependency in plan.get("dependencies") or []:
                if dependency.get("owner") == producer:
                    declared.append((index, "plan", dependency.get("stage") or "result", dependency.get("pass")))
    if bound and not declared:
        declared.append((bound[0][0], "bound", bound[0][1].get("stage") or "result",
                         purposes.get(events[bound[0][0]].get("pass"))))
    trace = {"declared": [{"at": index, "how": how, "stage": stage, "purpose": purpose}
                           for index, how, stage, purpose in declared]}
    if not declared:
        trace["binding"] = "not declared"
        return trace
    first, stage = declared[0][0], declared[0][2]
    target = declared[0][3]
    trace["targetPurpose"] = target
    accepted = [(index, event) for index, event in enumerate(events) if event.get("kind") == stage + ".accepted"
                and (not target or purposes.get(event.get("pass")) == target)
                and owners.get((event.get("data") or {}).get("proposal")) == producer]
    premature = [event.get("turn") for index, event in enumerate(events)
                 if index > first and event.get("kind") == "turn.start" and event.get("manager") == consumer
                 and (not target or purposes.get(event.get("pass")) == target)
                 and (event.get("data") or {}).get("mode") == "work"
                 and not any(at < index for at, _ in accepted)]
    trace["prematureWorkTurns"] = premature
    consumer_results = [index for index, event in enumerate(events) if event.get("kind") == "result.accepted"
                        and (not target or purposes.get(event.get("pass")) == target)
                        and owners.get((event.get("data") or {}).get("proposal")) == consumer]
    if not consumer_results:
        trace["binding"] = "declared, consumer never accepted"
        return trace
    final = consumer_results[-1]
    before = [(index, event) for index, event in accepted if index < final]
    if not before:
        trace["binding"] = "consumer accepted before any accepted %s %s" % (producer, stage)
        return trace
    index, event = before[-1]
    data = event.get("data") or {}
    consumed = data.get("proposal")
    trace["consumed"] = {"event": event.get("kind"), "proposal": consumed, "stand": data.get("stand"),
                         "at": event.get("at")}
    bound_before = [record for at, record in bound if at < final]
    if bound_before:
        recorded = bound_before[-1]
        trace["boundByProduct"] = {key: recorded.get(key) for key in ("stage", "serves", "proposal", "stand", "rebound")}
        if recorded.get("proposal") != consumed:
            trace["boundToSuperseded"] = {"bound": recorded.get("proposal"), "latest": consumed}
    undoing = "basis.invalidated" if stage == "basis" else "result.reopened"
    later = [event.get("kind") for at, event in enumerate(events) if at > final
             and (not target or purposes.get(event.get("pass")) == target)
             and owners.get((event.get("data") or {}).get("proposal")) == producer
             and (event.get("kind") == undoing
                  or (event.get("kind") == stage + ".accepted" and (event.get("data") or {}).get("proposal") != consumed))]
    reset = any(event.get("kind") == "proposal.invalidated" and owners.get((event.get("data") or {}).get("proposal")) == consumer
                for at, event in enumerate(events) if at > final)
    trace["staleAfterAcceptance"] = later if later and not reset else []
    stale = trace["staleAfterAcceptance"] or trace.get("boundToSuperseded")
    trace["binding"] = "met" if not premature and not stale else "violated"
    trace["unboundCycles"] = [(event.get("data") or {}).get("cycle") for event in events
                              if event.get("kind") == "dependency.unbound" and event.get("manager") == consumer]
    return trace


def question_route(events, asker, parent, answerer):
    """How information owned by ANSWERER reached ASKER: a question through PARENT, an accepted stage, or not traced."""
    asked = {(event.get("data") or {}).get("id"): event.get("data") or {} for event in events
             if event.get("kind") == "question.asked"}
    answered = {(event.get("data") or {}).get("id") for event in events if event.get("kind") == "question.answered"}
    for qid, question in asked.items():
        if question.get("from") == asker and question.get("to") == parent:
            forwarded = [other for other, item in asked.items() if item.get("about") == qid
                         and item.get("from") == parent and item.get("to") == answerer]
            if forwarded and forwarded[0] in answered and qid in answered:
                return {"route": "question", "questions": [qid, forwarded[0]]}
            if qid in answered:
                return {"route": "parent-decided", "questions": [qid]}
    trace = dependency_trace(events, asker, answerer)
    if trace.get("binding") == "met":
        return {"route": "accepted-stage", "consumed": trace.get("consumed")}
    return {"route": "unrouted", "dependency": trace.get("binding")}


def repeated_notes(findings):
    """Possible repeated notes for reading, not proof of reopening as a new duty.

    A retained reference to the same finding is never a new note. Different ids with the same wording can
    carry changed evidence or merely refer to settled history; causation needs the actual proposals/events.
    """
    seen, repeated, ids = {}, [], set()
    for item in findings:
        if item.get("blocking") or item.get("id") in ids:
            continue
        ids.add(item.get("id"))
        key = (item.get("owner"), " ".join(("%s %s" % (item.get("obligation", ""), item.get("observed", ""))).lower().split()))
        if key in seen and seen[key] != item.get("pass"):
            repeated.append(item.get("id"))
        seen.setdefault(key, item.get("pass"))
    return repeated


def process(scenario, summary, events, damaged=0):
    """Markitect-specific process criteria; incomplete when the events needed are missing or damaged."""
    if not events:
        return {"verdict": INCOMPLETE, "reasons": ["no events.jsonl"]}
    reasons, details = [], {}
    idle = idle_turns(events)
    details["idle"] = idle
    if scenario.internal_route:
        asker, parent, answerer = scenario.internal_route
        purposes = {event.get("pass"): (event.get("data") or {}).get("purpose") for event in events
                    if event.get("kind") == "pass.start"}
        work = [pid for pid, purpose in purposes.items() if purpose in REALIZATION_PURPOSES]
        details["workPasses"] = work
        if not work:
            return {"verdict": INCOMPLETE, "reasons": ["no source-bound realization Pass for DEP"], "details": details}
        # Staged Plan may declare a prerequisite for Prepare/Implement. Keep that relation while excluding
        # assessment/Verify evidence; dependency_trace matches acceptances and work to the target purpose.
        scoped = [event for event in events if event.get("pass") in work]
        binding, route = dependency_trace(scoped, answerer, asker), question_route(scoped, asker, parent, answerer)
        details.update(binding=binding, route=route)
        if binding.get("binding") != "met":
            reasons.append("%s's dependency on %s: %s" % (answerer, asker, binding.get("binding")))
        if route["route"] not in ("question", "accepted-stage"):
            reasons.append("the owned choice reached %s by route %s" % (asker, route["route"]))
        if idle["empty"] or idle["repeatedProgress"] or idle["emptyTurnStops"]:
            reasons.append("idle turns: %d empty, %d repeated progress" % (len(idle["empty"]), len(idle["repeatedProgress"])))
    if summary and scenario.expect == PRESERVE:
        verifies = [item for item in summary["passes"] if item.get("purpose") == "verify"]
        corrections = [item for item in summary["passes"] if item.get("verdict") == "needs_correction"]
        details.update(verifyPasses=len(verifies), correctionVerdicts=len(corrections))
        disproved = [item for item in summary["findings"] if independent_disproof(item, summary["passes"])]
        explained = {item.get("pass") for item in disproved}
        # Independent disproof may require a new complete Verify of the unchanged candidate. That contract
        # remains valid even when the original false factual blocker separately fails the quality outcome.
        if len(verifies) > 1 and not (len(verifies) == 2 and disproved and verifies[-1].get("verdict") == "satisfied" and
                                     all(item.get("id") in explained for item in corrections)):
            reasons.append("%d Verify Passes on the conforming stand" % len(verifies))
        unexplained = [item for item in corrections if item.get("id") not in explained]
        if unexplained:
            details["unresolvedCorrectionPasses"] = [item.get("id") for item in unexplained]
    if summary and (scenario.expect == PRESERVE or CASES[scenario.case].preserve):
        repeated = repeated_notes(summary["findings"])
        details["repeatedSuggestionCandidates"] = repeated
    if summary and scenario.id == "A1":
        open_findings = [item["id"] for item in summary["findings"] if item.get("blocking") and not item.get("closed")]
        details["openFindings"] = open_findings
        if open_findings:
            reasons.append("genuine findings still open: %s" % open_findings)
    if summary and scenario.expect in STOPS:
        realization = [event for event in events if event.get("kind") == "pass.start"
                       and str((event.get("data") or {}).get("purpose", "")).lower() in REALIZATION_PURPOSES]
        details["realizationPasses"] = len(realization)
        if scenario.preflight and realization:
            reasons.append("%d realization Pass(es) started before the stop" % len(realization))
    if damaged:
        return {"verdict": INCOMPLETE, "reasons": reasons + ["%d unreadable line(s) in events.jsonl" % damaged],
                "details": details}
    if details.get("unresolvedCorrectionPasses") and not reasons:
        return {"verdict": INCOMPLETE, "reasons": ["correction requires semantic disposition"], "details": details}
    return {"verdict": NOT_MET if reasons else MET, "reasons": reasons, "details": details}


# -- one attempt ----------------------------------------------------------------------

def evaluate(scenario_id, arm, repo, report=None, events=None, final=None, python=None, earlier_run=None):
    scenario = BY_ID[scenario_id]
    if arm not in scenario.arms:
        raise SystemExit("%s has no %s arm" % (scenario_id, arm))
    repo = Path(repo).resolve()
    python = python or python_command("")
    report_data = load_report(report) if arm in MARKITECT else None
    with tempfile.TemporaryDirectory(prefix="evaluate-") as folder:
        report_path = None
        if report_data is not None:
            report_path = Path(folder) / "report.json"
            report_path.write_text(json.dumps(report_data), encoding="utf-8")
        assessed = assess.assess(scenario.case, repo, str(report_path) if report_path else None, python)
    ephemeral = [path for path in assessed["changes"]["all"] if EPHEMERAL.search(path)]
    for key in ("all", "outsideFaultsAndAllowed", "protectedInputsChanged"):
        assessed["changes"][key] = [path for path in assessed["changes"][key] if not EPHEMERAL.search(path)]
    if arm == "plan":  # the plan arm's own input is plan.md; the scope rules concern realization/
        assessed["changes"]["outsideFaultsAndAllowed"] = [path for path in assessed["changes"]["outsideFaultsAndAllowed"]
                                                          if path != "plan.md"]
    case = CASES[scenario.case]
    if scenario.profile and scenario.profile != case.profile and (repo / "realization" / "app.py").is_file():
        checked = oracle.assess(case.project, scenario.profile, repo / "realization", python)
        counts = checked["summary"]
        assessed["behavior"] = {"passed": counts["failed"] == 0 and counts["skipped"] == 0, "summary": counts,
                                "profile": scenario.profile,
                                "failed": {item["id"]: item["detail"] for item in checked["checks"] if item["status"] == "fail"},
                                "skipped": {item["id"]: item["detail"] for item in checked["checks"] if item["status"] == "skip"}}
    summary = assess.summarize_report(report_data) if report_data is not None else None
    run_id = ((report_data or {}).get("run") or {}).get("id")
    final_data = load_final(final)
    verdict, category, reasons = outcome(scenario, arm, assessed, summary, final_data, earlier_run, run_id)
    result = {"scenario": scenario.id, "version": scenario.version, "case": scenario.case, "arm": arm,
              "question": scenario.question, "verdict": verdict, "category": category, "reasons": reasons,
              "changes": assessed["changes"], "ignoredEphemeral": ephemeral,
              "preserveTouched": sorted(set(assessed["changes"]["all"]) & set(case.preserve)),
              "behavior": assessed.get("behavior"), "static": assessed.get("static"), "faults": assessed.get("faults"),
              "tests": assessed.get("tests")}
    if arm in MARKITECT:
        records, damaged = load_jsonl(events)
        result["report"] = summary
        result["runId"] = run_id
        result["modelStop"] = model_stop(summary) if summary else None
        result["process"] = process(scenario, summary, records, damaged)
    else:
        result["final"] = {key: (final_data or {}).get(key) for key in ("status", "rounds", "elapsedSeconds", "usage")}
        result["blockers"] = blockers(final_data)
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("scenario", choices=sorted(BY_ID))
    parser.add_argument("--arm", required=True, choices=("direct", "staged", "plan"))
    parser.add_argument("--repo", required=True)
    parser.add_argument("--report")
    parser.add_argument("--events")
    parser.add_argument("--final")
    parser.add_argument("--earlier-run")
    parser.add_argument("--python", default="")
    parser.add_argument("--output")
    args = parser.parse_args(argv)
    result = evaluate(args.scenario, args.arm, args.repo, args.report, args.events, args.final,
                      python_command(args.python), args.earlier_run)
    text = json.dumps(result, indent=2)
    print(text)
    if args.output:
        Path(args.output).write_text(text + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    sys.exit(main())
