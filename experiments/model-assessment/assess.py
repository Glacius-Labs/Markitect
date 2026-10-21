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

ACCEPTED, GAP, ERROR, UNCLASSED, INCOMPLETE = "accepted", "model_gap", "model_error", "model_problem", "incomplete"
CONCLUSIONS = (ACCEPTED, GAP, ERROR, UNCLASSED)
PROBLEMS = (GAP, ERROR, UNCLASSED)
CORRECTIONS = [ERROR, GAP, "unclassified"]
MODEL_ERROR_REASON = "Root judged the model erroneous or contradictory: "
PROBLEM_REASON = "the model assessment confirmed a material model problem: "
ACCEPTED_REASON = "the model assessment accepted the model; this entry realizes nothing"
LIMIT = re.compile(r"^(the deadline was reached|the run was cancelled|the Pass budget of \d+ ended|the agent turn budget of \d+ ended)$")
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
    if declaration.get("corrections") != CORRECTIONS:
        problems.append("corrections must be exactly [model_error, model_gap, unclassified]: the classes of a confirmed finding, with an omitted class visible")
    return problems


def confirmed(run, pass_id):
    """The team's open genuine findings the last Pass confirmed: none disputed in it.

    An Orchestrator finding is no model judgment, and a finding Root disputed in
    the sealed Pass is a dissent, not a confirmation.
    """
    return [f for f in run.get("findings", []) if f.get("blocking") and not f.get("closed") and f.get("raisedBy") != "markitect"
            and not any(d.get("pass") == pass_id for d in f.get("disputes") or [])]


def classify(report):
    """The assessment conclusion of one harness report.

    accepted: the assessment-only run ended Incomplete with its explicit
    acceptance reason and a sealed Satisfied Pass, and the harness reported no
    error. This verifies no artifacts. model_gap or model_error: Root's integrated result
    sealed the Pass as needing correction for confirmed genuine findings of the
    team, classed by their correction (model_error when any is one, model_gap
    when any is one, model_problem when every class was omitted); or, as a
    second route, Root's model-error decision ended the run (model_error unless
    its text names model_gap). Everything else is incomplete and never an
    acceptance or a rejection of the model: limit (a deadline or budget ended
    unfinished work), unresolved (a sealed negative Pass without a confirmed
    finding), technical or preparation.
    """
    run = report.get("run")
    if not run:
        return {"conclusion": INCOMPLETE, "incomplete": "preparation", "why": "no run: preparation " + str(report.get("status")), "status": None}
    status, reason = run.get("status"), run.get("reason", "")
    aborts = [p for p in run.get("problems", []) if p.get("manager") == "." and p.get("abort") and p.get("category") == "model"]
    passes = run.get("passes", [])
    last = passes[-1] if passes else {}
    found = confirmed(run, last.get("id"))
    out = {"status": status, "reason": reason, "confirmedFindings": len(found), "passes": len(passes),
           "lastPassSealed": bool(last.get("sealed")), "lastVerdict": last.get("verdict", ""),
           "reportStatus": report.get("status"), "reportError": report.get("error", "")}
    assessment = last.get("purpose") == "assess" and last.get("review")
    sealed_negative = assessment and status == "failed" and reason.startswith(PROBLEM_REASON) and last.get("sealed") and last.get("verdict") == "needs_correction"
    if report.get("error"):
        out.update(conclusion=INCOMPLETE, incomplete="technical", why="the harness reported an execution or evidence error: " + str(report["error"]))
    elif assessment and status == "failed" and aborts and reason.startswith(MODEL_ERROR_REASON):
        gap = re.search(r"\bmodel[_ ]gap\b", aborts[-1].get("text", ""), re.IGNORECASE)
        out.update(conclusion=GAP if gap else ERROR, route="abort", why="Root ended the assessment with a model-problem decision")
    elif assessment and status == "incomplete" and reason == ACCEPTED_REASON and last.get("sealed") and last.get("verdict") == "satisfied" and report.get("status") == "passed" and not report.get("error"):
        out.update(conclusion=ACCEPTED, route="result", why="clean assessment Pass")
    elif sealed_negative and found:
        classes = {f.get("correction") for f in found}
        conclusion = ERROR if ERROR in classes else GAP if GAP in classes else UNCLASSED
        out.update(conclusion=conclusion, route="result", classes=sorted(c or "" for c in classes), why="Root's integrated result confirmed genuine findings")
    elif sealed_negative:
        out.update(conclusion=INCOMPLETE, incomplete="unresolved", why="the sealed Pass needs correction without a confirmed finding")
    else:
        kind = "limit" if status == "incomplete" and LIMIT.match(reason) else "technical"
        out.update(conclusion=INCOMPLETE, incomplete=kind, why="the run ended %s before a conclusion: %s %s" % (status, reason, report.get("error", "")))
    return out


def signals(run, conclusion=None):
    """Everything the team raised that the owner reads to correct the model.

    A blocking signal is one a model-problem conclusion rests on: a confirmed
    finding or Root's model-error decision. Without such a conclusion nothing
    blocks, so an unfinished attempt inflates no owner effort or false alarm. A
    disputed finding is dissent, a note a suggestion.
    """
    passes = run.get("passes", [])
    concluded = conclusion in PROBLEMS
    found = {f.get("id") for f in confirmed(run, passes[-1].get("id") if passes else None)} if concluded else set()
    out = []
    for p in run.get("problems", []):
        root_decision = bool(p.get("abort")) and p.get("manager") == "."
        out.append({"kind": "model_decision" if p.get("abort") else "problem", "id": p.get("id"), "by": p.get("manager"),
                    "text": "%s: %s" % (p.get("category"), p.get("text", "")), "blocking": concluded and root_decision})
    for f in run.get("findings", []):
        if f.get("raisedBy") == "markitect":
            continue
        if not f.get("blocking"):
            kind = "note"
        elif f.get("id") in found:
            kind = "finding"
        else:
            kind = "closed" if f.get("closed") else "disputed" if f.get("disputes") else "open"
        out.append({"kind": kind, "id": f.get("id"), "by": f.get("raisedBy"), "owner": f.get("owner"), "pass": f.get("pass"),
                    "class": f.get("correction", ""), "obligation": f.get("obligation", ""),
                    "text": "%s: %s" % (f.get("obligation", ""), f.get("observed", "")),
                    "blocking": f.get("id") in found, "closed": bool(f.get("closed")),
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
        if not starts:
            return {}
        try:
            # The common projector may print compact or indented JSON. Decode
            # one complete value before the harness's trailing PASS/FAIL lines.
            return json.JSONDecoder().raw_decode("\n".join(lines[starts[0]:]))[0]
        except json.JSONDecodeError:
            # A killed harness leaves a truncated report: no run, so incomplete.
            return {}


def jsonl(path):
    """The decodable lines of a JSON Lines file; a killed writer may leave a truncated last line."""
    if not path or not Path(path).is_file():
        return []
    out = []
    for line in Path(path).read_text(encoding="utf-8", errors="replace").splitlines():
        if line.strip():
            try:
                value = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(value, dict):
                out.append(value)
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


def pass_number(pass_id):
    """The number of a Pass identity p<N>, for ordering; 0 when it has none."""
    match = re.fullmatch(r"p(\d+)", str(pass_id or ""))
    return int(match.group(1)) if match else 0


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
    # A finding raised again in a later Pass on an obligation whose earlier finding was disproved.
    disproved = {}
    for f in findings:
        if f.get("closedAs") == "disproved":
            disproved[f.get("obligation")] = min(pass_number(f.get("closedPass")), disproved.get(f.get("obligation"), 1 << 30))
    reraised = [f.get("id") for f in findings if f.get("obligation") in disproved and pass_number(f.get("pass")) > disproved[f.get("obligation")]]
    return {"returned": [{"manager": e.get("manager"), "by": e.get("data", {}).get("by"), "reason": e.get("data", {}).get("reason")} for e in returned],
            "disputes": sum(len(f.get("disputes") or []) for f in findings),
            "disproved": sum(f.get("closedAs") == "disproved" for f in findings),
            "reraisedAfterDisproof": reraised,
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


def user_question_attempts(evidence, events=()):
    """Calls that a closed Run refuses as questions to the user: send to the user and Root's escalate."""
    attempts = []
    if not evidence:
        return attempts
    root = {e.get("turn") for e in events if e.get("kind") == "turn.start" and e.get("manager") == "."}
    for path in sorted((Path(evidence) / "transcripts").glob("*.jsonl")):
        for line in jsonl(path):
            data = line.get("data")
            if line.get("kind") != "tool_call" or not isinstance(data, dict) or not isinstance(data.get("arguments"), dict):
                continue
            arguments = data["arguments"]
            if data.get("name") == "send" and arguments.get("to") == "user" or data.get("name") == "escalate" and path.stem in root:
                attempts.append({"turn": path.stem, "tool": data.get("name"), "text": arguments.get("text", "")})
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
    harness_exit = (timing or {}).get("exit")
    if harness_exit is not None and str(harness_exit) != "0":
        classification = {"conclusion": INCOMPLETE, "incomplete": "technical", "harnessExit": str(harness_exit),
                          "why": "the harness process ended unsuccessfully after writing its report",
                          "reportClassification": classification}
    run = report.get("run") or {}
    definitions = control.get("definitions", []) if control else []
    raised = signals(run, classification.get("conclusion"))
    for s in raised:
        s["cites"] = cited(s["text"], definitions)
        s["matchesIssue"] = matches(s["text"], control.get("issue") if control else None, definitions)
        s["citesFreedom"] = matches(s["text"], control.get("freedom") if control else None, definitions)
        # The control's named observations, such as known minor ambiguities or invented requirements.
        s["tags"] = [name for name, target in sorted(((control or {}).get("tags") or {}).items()) if matches(s["text"], target, definitions)]
    blocking = [s for s in raised if s["blocking"]]
    turns = run.get("turns", [])
    usage = run.get("usage", {})
    asked = user_question_attempts(evidence, events)
    out = {"classification": classification, "signals": raised, "process": process(events, run),
           "delivery": delivery(evidence, events, control), "userQuestionAttempts": asked,
           "ownerEffort": {"confirmedFindings": len(blocking), "confirmedChars": sum(len(s["text"]) for s in blocking),
                           "notes": sum(s["kind"] == "note" for s in raised), "disputedOpen": sum(s["kind"] == "disputed" for s in raised)},
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
        # The class that counts is the class of the finding on the known issue, not of an unrelated one.
        out["issueClasses"] = sorted({s.get("class") or classification["conclusion"] for s in blocking if s["matchesIssue"]})
        out["missedIssue"] = concluded and not (out["blocked"] and out["issueRaised"])
    if concluded:
        if control.get("issue"):
            out["expectationMet"] = out["blocked"] and out["issueRaised"] and expected in out["issueClasses"]
        else:
            out["expectationMet"] = conclusion == expected
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
    if control and control.get("expected") not in (ACCEPTED, GAP, ERROR):
        raise SystemExit("the control's expected conclusion must be accepted, model_gap or model_error")
    result = score(report, classify(report), control, events, timing_of(args.timing), evidence)
    Path(args.output).write_text(json.dumps(result, indent=1), encoding="utf-8")
    print(json.dumps({"conclusion": result["classification"]["conclusion"], "expectationMet": result.get("expectationMet")}))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
