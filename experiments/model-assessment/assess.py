"""Classify and score Model assessment attempts; see README.md and contract.md.

Usage:
  assess.py preflight PASS_JSON
  assess.py attempt --report REPORT_JSON [--evidence EVIDENCE_DIR] [--control CONTROL_JSON] [--timing TIMING_TXT] --output OUT_JSON
  assess.py series [--ledger LEDGER] OUT_JSON...

`attempt` reads the report the grant-gated harness wrote and, when given, the
run's evidence directory (events and transcripts). It classifies the outcome
from the common envelope alone, measures the actual delivery of each
reviewer's own Definitions, then scores it against the control's expectation,
which the Agents never see. Matching a signal to the known issue is an aid for
the evaluator, not a verdict: every signal stays listed for manual adjudication.
"""
import argparse
import json
import re
import sys
from pathlib import Path

ACCEPTED, GAP, ERROR, INCOMPLETE = "accepted", "model_gap", "model_error", "incomplete"
CONCLUSIONS = (ACCEPTED, GAP, ERROR)
PROBLEMS = (GAP, ERROR)
MODEL_ERROR_REASON = "Root judged the model erroneous or contradictory: "
PROBLEM_REASON = "the model assessment confirmed a material model problem"
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
    if declaration.get("corrections") != [ERROR, GAP]:
        problems.append("corrections must be exactly [model_error, model_gap]: the classes of a confirmed finding")
    return problems


def team_findings(run):
    """The open genuine findings of the team; an Orchestrator finding is no model judgment."""
    return [f for f in run.get("findings", []) if f.get("blocking") and not f.get("closed") and f.get("raisedBy") != "markitect"]


def classify(report):
    """The assessment conclusion of one harness report.

    accepted: the run verified, so the assessment Pass was clean, and the harness
    reported no error. model_gap or model_error: Root's integrated result sealed
    the Pass as needing correction for open genuine findings of the team, classed
    by their correction (model_error when any is one); or, as a second route,
    Root's model-error decision ended the run. Everything else is incomplete:
    a technical end that is never an acceptance or a rejection of the model.
    """
    run = report.get("run")
    if not run:
        return {"conclusion": INCOMPLETE, "why": "no run: preparation " + str(report.get("status")), "status": None}
    status, reason = run.get("status"), run.get("reason", "")
    aborts = [p for p in run.get("problems", []) if p.get("manager") == "." and p.get("abort") and p.get("category") == "model"]
    open_team = team_findings(run)
    passes = run.get("passes", [])
    last = passes[-1] if passes else {}
    out = {"status": status, "reason": reason, "openTeamFindings": len(open_team), "passes": len(passes),
           "lastPassSealed": bool(last.get("sealed")), "lastVerdict": last.get("verdict", ""),
           "reportStatus": report.get("status"), "reportError": report.get("error", "")}
    if status == "failed" and aborts and reason.startswith(MODEL_ERROR_REASON):
        gap = re.search(r"\bmodel[_ ]gap\b", aborts[-1].get("text", ""), re.IGNORECASE)
        out.update(conclusion=GAP if gap else ERROR, route="abort", why="Root ended the assessment with a model-problem decision")
    elif status == "verified" and report.get("status") == "passed" and not report.get("error"):
        out.update(conclusion=ACCEPTED, route="result", why="clean assessment Pass")
    elif status == "incomplete" and reason == PROBLEM_REASON and last.get("sealed") and last.get("verdict") == "needs_correction" and open_team:
        error = any(f.get("correction") == ERROR for f in open_team)
        out.update(conclusion=ERROR if error else GAP, route="result", why="Root's integrated result confirmed genuine findings")
    else:
        out.update(conclusion=INCOMPLETE, why="the run ended %s before a conclusion: %s %s" % (status, reason, report.get("error", "")))
    return out


def signals(run):
    """Everything the team raised that the owner reads to correct the model."""
    out = []
    for p in run.get("problems", []):
        out.append({"kind": "model_decision" if p.get("abort") else "problem", "id": p.get("id"), "by": p.get("manager"),
                    "text": "%s: %s" % (p.get("category"), p.get("text", "")), "blocking": bool(p.get("abort"))})
    for f in run.get("findings", []):
        if f.get("raisedBy") == "markitect":
            continue
        out.append({"kind": "finding" if f.get("blocking") else "note", "id": f.get("id"), "by": f.get("raisedBy"),
                    "owner": f.get("owner"), "class": f.get("correction", ""), "text": "%s: %s" % (f.get("obligation", ""), f.get("observed", "")),
                    "blocking": bool(f.get("blocking")) and not f.get("closed"), "closed": bool(f.get("closed")),
                    "closedAs": f.get("closedAs", ""), "disputes": len(f.get("disputes") or [])})
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
    """Whether a text concerns a target: by its terms when it has any, otherwise by a cited Definition."""
    if not target:
        return False
    if target.get("terms"):
        return any(re.search(term, text, re.IGNORECASE) for term in target["terms"])
    return bool(set(cited(text, definitions)) & set(target.get("definitions", [])))


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


def jsonl(path):
    if not path or not Path(path).is_file():
        return []
    out = []
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        if line.strip():
            out.append(json.loads(line))
    return out


def strings(value):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for v in value.values():
            yield from strings(v)
    elif isinstance(value, list):
        for v in value:
            yield from strings(v)


def flat(text):
    return " ".join(text.split())


def process(events, run):
    """Disagreement, coverage and communication evidence from the run's events."""
    returned = [e for e in events if e.get("kind") in ("basis.returned", "result.returned")]
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
    findings = [f for f in run.get("findings", []) if f.get("raisedBy") != "markitect"]
    return {"returned": [{"manager": e.get("manager"), "by": e.get("data", {}).get("by"), "reason": e.get("data", {}).get("reason")} for e in returned],
            "disputes": sum(len(f.get("disputes") or []) for f in findings),
            "disproved": sum(f.get("closedAs") == "disproved" for f in findings),
            "internalQuestions": len([q for q in run.get("questions", []) if q.get("to") != "user"]),
            "acceptedResultCoverage": coverage}


def delivery(evidence, events, control):
    """Whether the first prompt sent to each reviewer of each Pass carried its own Definitions in full.

    The expected Definitions and bodies come from the frozen packet; the sent
    payload from the turn's transcript at the adapter boundary. A body counts as
    delivered when its whitespace-normalized text occurs in the sent text; an
    identity alone counts as reference only. This is technical delivery, not
    evidence that an Agent read or followed it.
    """
    texts = (control or {}).get("definitionTexts") or {}
    if not evidence or not texts:
        return None
    first = {}
    for e in events:
        if e.get("kind") == "turn.start" and (e.get("data") or {}).get("role") == "reviewer":
            first.setdefault((e.get("pass"), e.get("manager")), e.get("turn"))
    out = {"reviewers": 0, "expected": 0, "full": 0, "referenceOnly": 0, "missing": [], "noTranscript": []}
    for (pass_id, manager), turn in sorted(first.items()):
        own = {i: d for i, d in texts.items() if d.get("package") == manager}
        lines = jsonl(Path(evidence) / "transcripts" / f"{turn}.jsonl")
        sent = flat(" ".join(s for line in lines if line.get("kind") == "sent" for s in strings(line.get("data"))))
        out["reviewers"] += 1
        if not lines:
            out["noTranscript"].append(turn)
            continue
        for identity, d in own.items():
            out["expected"] += 1
            if d.get("body") and flat(d["body"]) in sent and identity in sent:
                out["full"] += 1
            elif identity in sent:
                out["referenceOnly"] += 1
                out["missing"].append(f"{pass_id} {manager} {identity}: body not sent")
            else:
                out["missing"].append(f"{pass_id} {manager} {identity}: not sent")
    return out


def user_question_attempts(evidence):
    """Calls of send to the user, which a closed Run refuses; each is an attempted external question."""
    attempts = []
    if not evidence:
        return attempts
    for path in sorted((Path(evidence) / "transcripts").glob("*.jsonl")):
        for line in jsonl(path):
            data = line.get("data") or {}
            arguments = data.get("arguments") or {}
            if line.get("kind") == "tool_call" and data.get("name") == "send" and isinstance(arguments, dict) and arguments.get("to") == "user":
                attempts.append({"turn": path.stem, "text": arguments.get("text", "")})
    return attempts


def timing_of(path):
    values = {}
    if path and Path(path).exists():
        for token in Path(path).read_text(encoding="utf-8").split():
            if "=" in token:
                key, value = token.split("=", 1)
                values[key] = value
    return values


def score(report, classification, control, events=(), timing=None, evidence=None):
    run = report.get("run") or {}
    definitions = control.get("definitions", []) if control else []
    raised = signals(run)
    for s in raised:
        s["cites"] = cited(s["text"], definitions)
        s["matchesIssue"] = matches(s["text"], control.get("issue") if control else None, definitions)
        s["citesFreedom"] = matches(s["text"], control.get("freedom") if control else None, definitions)
        # The control's named observations, such as known minor ambiguities or invented requirements.
        s["tags"] = [name for name, target in sorted(((control or {}).get("tags") or {}).items()) if matches(s["text"], target, definitions)]
    blocking = [s for s in raised if s["blocking"]]
    turns = run.get("turns", [])
    usage = run.get("usage", {})
    asked = user_question_attempts(evidence)
    out = {"classification": classification, "signals": raised, "process": process(events, run),
           "delivery": delivery(evidence, events, control), "userQuestionAttempts": asked,
           "ownerEffort": {"confirmedFindings": len(blocking), "confirmedChars": sum(len(s["text"]) for s in blocking),
                           "notes": sum(s["kind"] == "note" for s in raised)},
           "cost": {"turns": len(turns), "costUSD": usage.get("costUSD"), "inputTokens": usage.get("inputTokens"),
                    "outputTokens": usage.get("outputTokens"), "wallSeconds": (timing or {}).get("wall_seconds")}}
    if not control:
        return out
    expected, conclusion = control["expected"], classification["conclusion"]
    concluded = conclusion in CONCLUSIONS
    out["control"] = {"id": control["id"], "expected": expected}
    out["blocked"] = conclusion in PROBLEMS
    out["falseAbort"] = expected == ACCEPTED and out["blocked"]
    out["expectationMet"] = None
    if control.get("issue"):
        out["issueRaised"] = any(s["matchesIssue"] for s in blocking)
        out["missedIssue"] = concluded and not (out["blocked"] and out["issueRaised"])
    if concluded:
        out["expectationMet"] = conclusion == expected and out.get("issueRaised", True)
    out["falseAlarmCandidates"] = [s["id"] for s in blocking if not s["matchesIssue"]]
    out["uncitedBlocking"] = [s["id"] for s in blocking if not s["cites"]]
    return out


def series(results, ledger=""):
    """Per control: conclusions, met expectations and the evaluator's material.

    With the ledger, every recorded attempt counts, also one that was never
    assessed, so a missing assessment cannot shrink the series.
    """
    by = {}

    def entry(control):
        return by.setdefault(control, {"attempts": 0, "conclusions": {}, "met": 0, "incomplete": 0, "blocked": 0, "falseAborts": 0,
                                       "missedIssues": 0, "falseAlarmCandidates": 0, "userQuestionAttempts": 0, "turns": 0,
                                       "costUSD": 0.0, "wallSeconds": 0})
    for line in ledger.splitlines():
        fields = line.split()
        if len(fields) >= 2:
            e = entry(fields[0])
            e["ledgerAttempts"] = e.get("ledgerAttempts", 0) + 1
    for r in results:
        e = entry((r.get("control") or {}).get("id", "unbound"))
        conclusion = r["classification"]["conclusion"]
        e["attempts"] += 1
        e["conclusions"][conclusion] = e["conclusions"].get(conclusion, 0) + 1
        e["met"] += r.get("expectationMet") is True
        e["incomplete"] += conclusion == INCOMPLETE
        e["blocked"] += r.get("blocked") is True
        e["falseAborts"] += r.get("falseAbort") is True
        e["missedIssues"] += r.get("missedIssue") is True
        e["falseAlarmCandidates"] += len(r.get("falseAlarmCandidates", []))
        e["userQuestionAttempts"] += len(r.get("userQuestionAttempts", []))
        e["turns"] += r["cost"]["turns"]
        e["costUSD"] += r["cost"]["costUSD"] or 0
        e["wallSeconds"] += int(r["cost"]["wallSeconds"] or 0)
    for e in by.values():
        e["agreement"] = len(e["conclusions"]) == 1
        if "ledgerAttempts" in e:
            e["unassessed"] = e["ledgerAttempts"] - e["attempts"]
    return by


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__.split("\n", 1)[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("preflight")
    p.add_argument("declaration")
    a = sub.add_parser("attempt")
    a.add_argument("--report", required=True)
    a.add_argument("--evidence")
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
    evidence = args.evidence if args.evidence and Path(args.evidence).is_dir() else None
    events = jsonl(Path(evidence) / "events.jsonl") if evidence else []
    result = score(report, classify(report), control, events, timing_of(args.timing), evidence)
    Path(args.output).write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps({"conclusion": result["classification"]["conclusion"], "expectationMet": result.get("expectationMet")}))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
