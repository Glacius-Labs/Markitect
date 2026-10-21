"""Classify and score Model assessment attempts; see README.md.

Usage:
  assess.py preflight PASS_JSON
  assess.py attempt --report APPLY_JSON [--events EVENTS_JSONL] [--control CONTROL_JSON] [--timing TIMING_TXT] --output OUT_JSON
  assess.py series OUT_JSON...

`attempt` reads the report the grant-gated harness printed and, when given, the
run's events. It classifies the outcome from the common envelope alone, then
scores it against the control's expectation, which the Agents never see.
Matching a signal to the known issue is an aid for the evaluator, not a verdict:
every signal stays listed for manual adjudication.
"""
import argparse
import json
import re
import sys
from pathlib import Path

ACCEPTED, CLARIFICATION, MODEL_ERROR, INCOMPLETE = "accepted", "needs_clarification", "model_error", "incomplete"
CONCLUSIONS = (ACCEPTED, CLARIFICATION, MODEL_ERROR)
MODEL_ERROR_REASON = "Root judged the model erroneous or contradictory: "
OPEN_FINDINGS_REASON = "the model assessment left genuine findings open"
NAME = re.compile(r"^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$")


def preflight(declaration):
    """The problems of an assessment Pass declaration; the run refuses the same."""
    problems = []
    if set(declaration) != {"purpose", "mandate", "guidance", "corrections"}:
        problems.append("fields must be exactly purpose, mandate, guidance, corrections")
    if not NAME.match(str(declaration.get("purpose", ""))) or declaration.get("purpose") == "verify":
        problems.append("purpose must be a valid name other than verify")
    for field in ("mandate", "guidance"):
        if not str(declaration.get(field, "")).strip():
            problems.append(field + " must not be empty")
    corrections = declaration.get("corrections")
    if not isinstance(corrections, list) or not corrections or len(set(corrections)) != len(corrections) or \
            any(not NAME.match(str(c)) or c == "verify" for c in corrections):
        problems.append("corrections must be distinct valid names other than verify")
    return problems


def classify(report):
    """The assessment conclusion of one harness report.

    accepted: the run verified, so the review Pass was clean, and the harness
    reported no error. model_error: Root's model-error decision ended the run.
    needs_clarification: Root asked the user and the sealed last Pass needs
    correction for an open genuine finding of the team. Everything else is
    incomplete, with the reason kept separately from model quality: technical
    (a run that ended before its conclusion) or unrouted (the team left genuine
    findings open without asking the user).
    """
    run = report.get("run")
    if not run:
        return {"conclusion": INCOMPLETE, "incomplete": "technical", "why": "no run: preparation " + str(report.get("status")), "status": None}
    status, reason = run.get("status"), run.get("reason", "")
    questions = [q for q in run.get("questions", []) if q.get("to") == "user"]
    aborts = [p for p in run.get("problems", []) if p.get("manager") == "." and p.get("abort") and p.get("category") == "model"]
    open_genuine = [f for f in run.get("findings", []) if f.get("blocking") and not f.get("closed") and f.get("raisedBy") != "markitect"]
    passes = run.get("passes", [])
    last = passes[-1] if passes else {}
    out = {"status": status, "reason": reason, "questions": len(questions), "openGenuineFindings": len(open_genuine),
           "passes": len(passes), "lastPassSealed": bool(last.get("sealed")), "lastVerdict": last.get("verdict", ""),
           "reportStatus": report.get("status"), "reportError": report.get("error", "")}
    if status == "failed" and aborts and reason.startswith(MODEL_ERROR_REASON):
        out.update(conclusion=MODEL_ERROR, why="Root ended the assessment with a model error")
    elif status == "verified" and report.get("status") == "passed" and not report.get("error"):
        out.update(conclusion=ACCEPTED, why="clean assessment Pass")
    elif status == "incomplete" and reason == OPEN_FINDINGS_REASON and last.get("sealed") and \
            last.get("verdict") == "needs_correction" and questions and open_genuine:
        out.update(conclusion=CLARIFICATION, why="Root asked the user and kept genuine findings open")
    elif status == "incomplete" and reason == OPEN_FINDINGS_REASON and open_genuine and not questions:
        out.update(conclusion=INCOMPLETE, incomplete="unrouted", why="genuine findings open without a question to the user")
    else:
        out.update(conclusion=INCOMPLETE, incomplete="technical", why="the run ended %s before a conclusion: %s %s" % (status, reason, report.get("error", "")))
    return out


def signals(run):
    """Everything the team raised that an owner would read."""
    out = []
    for q in run.get("questions", []):
        if q.get("to") == "user":
            out.append({"kind": "question", "id": q.get("id"), "by": q.get("from"), "text": q.get("text", ""), "blocking": True})
    for p in run.get("problems", []):
        out.append({"kind": "model_error" if p.get("abort") else "problem", "id": p.get("id"), "by": p.get("manager"),
                    "text": "%s: %s" % (p.get("category"), p.get("text", "")), "blocking": bool(p.get("abort") or p.get("escalated"))})
    for f in run.get("findings", []):
        out.append({"kind": "finding" if f.get("blocking") else "note", "id": f.get("id"), "by": f.get("raisedBy"),
                    "owner": f.get("owner"), "text": "%s: %s" % (f.get("obligation", ""), f.get("observed", "")),
                    "blocking": bool(f.get("blocking")), "closed": bool(f.get("closed")), "closedAs": f.get("closedAs", ""),
                    "disputes": len(f.get("disputes") or [])})
    return out


def cited(text, definitions):
    """The Definition identities a text names, in full or by a bracketed or quoted local id."""
    found = []
    for identity in definitions:
        local = identity.rsplit("/", 1)[-1]
        if re.search(r"(?<![\w/-])" + re.escape(identity) + r"(?![\w-])", text) or \
                re.search(r"(\[\[|`|\")" + re.escape(local) + r"(\]\]|\||`|\")", text):
            found.append(identity)
    return found


def matches(text, target, definitions):
    if not target:
        return False
    if set(cited(text, definitions)) & set(target.get("definitions", [])):
        return True
    return any(re.search(term, text, re.IGNORECASE) for term in target.get("terms", []))


def load_report(text):
    """The report JSON, also when the harness binary's own PASS or FAIL lines surround it."""
    if not text.strip():
        return {}
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        lines = text.splitlines()
        starts = [i for i, line in enumerate(lines) if line.startswith("{")]
        ends = [i for i, line in enumerate(lines) if line.startswith("}")]
        if not starts or not ends or ends[-1] < starts[0]:
            return {}
        return json.loads("\n".join(lines[starts[0]:ends[-1] + 1]))


def events_of(path):
    if not path or not Path(path).exists():
        return []
    events = []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        if line.strip():
            events.append(json.loads(line))
    return events


def process(events):
    """Disagreement and coverage evidence from the run's events."""
    returned = [e for e in events if e.get("kind") in ("basis.returned", "result.returned")]
    disputed = [e for e in events if e.get("kind") == "finding.disputed"]
    accepted = {e.get("data", {}).get("proposal") for e in events if e.get("kind") == "result.accepted"}
    coverage = {"obligations": 0, "withModelSource": 0, "open": 0}
    for e in events:
        data = e.get("data", {})
        if e.get("kind") != "proposal.submitted" or data.get("kind") != "result" or data.get("proposal") not in accepted:
            continue
        for c in (data.get("envelope") or {}).get("obligations") or []:
            coverage["obligations"] += 1
            coverage["open"] += c.get("status") == "open"
            if any(str(item.get("source", "")).startswith(".markitect/") for item in c.get("evidence") or []):
                coverage["withModelSource"] += 1
    return {"returned": [{"manager": e.get("manager"), "by": e.get("data", {}).get("by"), "reason": e.get("data", {}).get("reason")} for e in returned],
            "disputes": len(disputed), "acceptedResultCoverage": coverage}


def timing_of(path):
    values = {}
    if path and Path(path).exists():
        for token in Path(path).read_text(encoding="utf-8").split():
            if "=" in token:
                key, value = token.split("=", 1)
                values[key] = value
    return values


def score(report, classification, control, events=(), timing=None):
    run = report.get("run") or {}
    definitions = control.get("definitions", []) if control else []
    raised = signals(run)
    for s in raised:
        s["cites"] = cited(s["text"], definitions)
        s["matchesIssue"] = matches(s["text"], control.get("issue") if control else None, definitions)
        s["citesFreedom"] = matches(s["text"], control.get("freedom") if control else None, definitions)
    blocking = [s for s in raised if s["blocking"]]
    turns = run.get("turns", [])
    usage = run.get("usage", {})
    out = {"classification": classification, "signals": raised, "process": process(events),
           "ownerEffort": {"questions": sum(s["kind"] == "question" for s in raised),
                           "questionChars": sum(len(s["text"]) for s in raised if s["kind"] == "question"),
                           "genuineFindings": sum(s["kind"] == "finding" for s in raised),
                           "notes": sum(s["kind"] == "note" for s in raised)},
           "cost": {"turns": len(turns), "costUSD": usage.get("costUSD"), "inputTokens": usage.get("inputTokens"),
                    "outputTokens": usage.get("outputTokens"), "wallSeconds": (timing or {}).get("wall_seconds")}}
    if not control:
        return out
    expected = control["expected"]
    concluded = classification["conclusion"] in CONCLUSIONS
    out["control"] = {"id": control["id"], "expected": expected}
    out["expectationMet"] = None
    if control.get("issue"):
        out["issueRaised"] = any(s["matchesIssue"] for s in blocking)
        out["missedIssue"] = concluded and not out["issueRaised"]
    if concluded:
        out["expectationMet"] = classification["conclusion"] == expected and out.get("issueRaised", True)
    out["falseAlarmCandidates"] = [s["id"] for s in blocking if not s["matchesIssue"]]
    out["uncitedBlocking"] = [s["id"] for s in blocking if not s["cites"]]
    return out


def series(results, ledger=""):
    """Per control: conclusions, met expectations and the evaluator's material.

    With the ledger, every recorded attempt counts, also one that was never
    assessed, so a missing assessment cannot shrink the series.
    """
    by = {}
    for line in ledger.splitlines():
        fields = line.split()
        if len(fields) >= 2:
            entry = by.setdefault(fields[0], {"attempts": 0, "conclusions": {}, "met": 0, "incomplete": 0, "issueRaised": 0,
                                              "falseAlarmCandidates": 0, "questions": 0, "turns": 0, "costUSD": 0.0, "wallSeconds": 0})
            entry["ledgerAttempts"] = entry.get("ledgerAttempts", 0) + 1
    for r in results:
        control = (r.get("control") or {}).get("id", "unbound")
        entry = by.setdefault(control, {"attempts": 0, "conclusions": {}, "met": 0, "incomplete": 0, "issueRaised": 0,
                                        "falseAlarmCandidates": 0, "questions": 0, "turns": 0, "costUSD": 0.0, "wallSeconds": 0})
        conclusion = r["classification"]["conclusion"]
        entry["attempts"] += 1
        entry["conclusions"][conclusion] = entry["conclusions"].get(conclusion, 0) + 1
        entry["met"] += r.get("expectationMet") is True
        entry["incomplete"] += conclusion == INCOMPLETE
        entry["issueRaised"] += r.get("issueRaised") is True
        entry["falseAlarmCandidates"] += len(r.get("falseAlarmCandidates", []))
        entry["questions"] += r["ownerEffort"]["questions"]
        entry["turns"] += r["cost"]["turns"]
        entry["costUSD"] += r["cost"]["costUSD"] or 0
        entry["wallSeconds"] += int(r["cost"]["wallSeconds"] or 0)
    for entry in by.values():
        entry["agreement"] = len(entry["conclusions"]) == 1
        if "ledgerAttempts" in entry:
            entry["unassessed"] = entry["ledgerAttempts"] - entry["attempts"]
    return by


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("preflight")
    p.add_argument("declaration")
    a = sub.add_parser("attempt")
    a.add_argument("--report", required=True)
    a.add_argument("--events")
    a.add_argument("--control")
    a.add_argument("--timing")
    a.add_argument("--output", required=True)
    s = sub.add_parser("series")
    s.add_argument("--ledger")
    s.add_argument("results", nargs="*")
    args = parser.parse_args(argv[1:])
    if args.command == "preflight":
        problems = preflight(json.loads(Path(args.declaration).read_text(encoding="utf-8")))
        print(json.dumps({"declaration": args.declaration, "problems": problems}, indent=1))
        return 1 if problems else 0
    if args.command == "series":
        ledger = Path(args.ledger).read_text(encoding="utf-8") if args.ledger else ""
        print(json.dumps(series([json.loads(Path(r).read_text(encoding="utf-8")) for r in args.results], ledger), indent=1))
        return 0
    report = load_report(Path(args.report).read_text(encoding="utf-8")) if Path(args.report).exists() else {}
    control = json.loads(Path(args.control).read_text(encoding="utf-8")) if args.control else None
    result = score(report, classify(report), control, events_of(args.events), timing_of(args.timing))
    Path(args.output).write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps({"conclusion": result["classification"]["conclusion"], "expectationMet": result.get("expectationMet")}))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
