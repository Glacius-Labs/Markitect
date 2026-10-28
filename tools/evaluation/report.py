"""The series report: one table per area, from every launch, judgment and review of a series. No overall score.

A series is described by its ledger, a JSON file that lists every launch,
voided ones included, with paths relative to the ledger:

  {"format": 1, "series": "NAME", "signatures": "signatures.json",
   "tasks": {"library-m1": {"inputEffort": null, "note": "task.md is rendered from the model"}},
   "launches": [{"id": "k-02", "variant": "K", "task": "library-m1", "run": "RUN_DIR",
                 "judgment": "JUDGE_OUT_DIR", "review": "REVIEW_OUT_DIR"}]}

`signatures` (optional) are the variant runtime signatures of classify.py,
written before the series. Each launch is classified first; only results are
counted in the areas, and the void rate shows the rest. A result without a
delivered commit, or one that timed out, counts as a failure in Correctness
and Delivery.

Areas, each a table per variant:
  human attention (primary)  defects left for a person: failed cases, a failed build, failed tests,
                             unrequested paths, protected inputs touched, confirmed critical findings,
                             steps with no result; and the task's input effort
  correctness                hidden cases passed of planned
  build and tests            the result's own build and tests
  scope                      unrequested paths and protected inputs touched
  code quality               confirmed critical findings, and how often reviewers guessed the variant
  delivery                   results delivered and timeouts
  void rate                  launches, void by kind, invalid
  cost (descriptive)         wall time, tokens over every transcript, and the cost the CLI states
"""
import json
import pathlib
import statistics

import classify
import tokens

FORMAT = 1


def read(path):
    try:
        value = json.loads(pathlib.Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return None
    return value if isinstance(value, dict) else None


def launch_facts(base, launch, signatures):
    """Everything the report needs from one launch."""
    run = base / launch["run"]
    if not run.is_dir():
        # A wrong path in the ledger is the ledger's error, not a launch without a record.
        raise ValueError(f"launch {launch['id']}: {run} does not exist")
    record = read(run / "suite.json") or {}
    facts = {"id": launch["id"], "variant": launch["variant"], "task": launch["task"],
             **classify.classify(run, signatures, launch["variant"])}
    if facts["class"] != "result":
        return facts
    reported = record.get("reported") or {}
    counted = tokens.count(run / "variant") if (run / "variant").is_dir() else None
    facts.update(delivered=bool(record.get("delivered")), timedOut=bool(record.get("timedOut")), wallSeconds=record.get("wallSeconds"),
                 costUsd=(reported.get("usage") or {}).get("costUsd"), tokens=counted if counted and counted["responses"] else None)
    judgment = read(base / launch["judgment"] / "judgment.json") if launch.get("judgment") else None
    if facts["delivered"]:
        if not judgment or not judgment.get("valid") or judgment.get("commit") != (record.get("result") or {}).get("commit"):
            facts["missing"] = "no valid judgment of this result"
        else:
            facts["areas"] = judgment["areas"]
    review = read(base / launch["review"] / "review.json") if launch.get("review") else None
    if facts["delivered"] and review and review.get("valid") and review.get("commit") == (record.get("result") or {}).get("commit"):
        facts["review"] = {"confirmed": review["criticals"]["confirmed"], "valid": review["criticals"]["valid"],
                           "reported": review["criticals"]["reported"], "guesses": [g["variant"] for g in review["guesses"]]}
    return facts


def planned_cases(facts):
    """The planned case count per task, from its judgments: a run without a result fails every case."""
    planned = {}
    for f in facts:
        if "areas" in f:
            planned[f["task"]] = max(planned.get(f["task"], 0), f["areas"]["correctness"]["planned"])
    return planned


def attention(f, planned):
    """The defects a person would still have to find and repair, per kind; None when the run is not judged."""
    if not f["delivered"]:
        return {"noResult": 1, "casesFailed": planned.get(f["task"]), "confirmedCriticals": 0}
    if "areas" not in f:
        return None
    a = f["areas"]
    return {"noResult": 0, "casesFailed": a["correctness"]["failed"], "buildFailed": int(not a["buildAndTests"]["build"]["passed"]),
            "testsFailed": int(not a["buildAndTests"]["tests"]["passed"]), "unrequested": len(a["scope"]["unrequested"]),
            "protected": len(a["scope"]["protectedTouched"]),
            "confirmedCriticals": f["review"]["confirmed"] if "review" in f else None}


def spread(values):
    """mean (min-max, n) of the known values."""
    values = [v for v in values if v is not None]
    if not values:
        return "n/a"
    mean = statistics.fmean(values)
    text = f"{mean:.2f}".rstrip("0").rstrip(".")
    return f"{text} ({min(values)}-{max(values)}, n={len(values)})" if len(values) > 1 else f"{text} (n=1)"


def share(hits, n):
    return f"{hits}/{n}" if n else "n/a"


def table(title, header, rows, note=None):
    lines = [f"## {title}", ""]
    if note:
        lines += [note, ""]
    lines += ["| " + " | ".join(header) + " |", "|" + "---|" * len(header)]
    lines += ["| " + " | ".join(str(c).replace("|", "\\|") for c in row) + " |" for row in rows]
    return "\n".join(lines) + "\n"


def report(ledger_path):
    ledger_path = pathlib.Path(ledger_path)
    ledger = read(ledger_path)
    if not ledger or ledger.get("format") != FORMAT or not isinstance(ledger.get("launches"), list):
        raise ValueError(f"{ledger_path} is not a series ledger")
    base = ledger_path.parent
    signatures = read(base / ledger["signatures"]) if ledger.get("signatures") else None
    facts = [launch_facts(base, launch, signatures) for launch in ledger["launches"]]
    planned = planned_cases(facts)
    for f in facts:
        if f["class"] == "result":
            f["attention"] = attention(f, planned)
    variants = sorted({f["variant"] for f in facts})
    tasks = ledger.get("tasks") or {}
    out = [f"# Series report: {ledger.get('series', ledger_path.stem)}", "",
           "One table per area; there is no overall score. Counted are the launches classified as results; "
           "a result without a delivered commit, or one that timed out, counts as a failure in Correctness and Delivery. "
           "Values are mean (min-max, n).", ""]
    by = {v: [f for f in facts if f["variant"] == v and f["class"] == "result"] for v in variants}
    missing = [f"{f['id']}: {f['missing']}" for f in facts if f.get("missing")]
    if missing:
        out += ["**Not judged:** " + "; ".join(missing) + ".", ""]

    rows = []
    for v in variants:
        att = [f["attention"] for f in by[v] if f.get("attention")]
        # A total only where every count is known: a delivered result needs its review too.
        total = [sum(a.values()) if None not in a.values() else None for a in att]
        rows.append([v, len(by[v]), spread(total), spread([a["casesFailed"] for a in att]), spread([a.get("buildFailed") for a in att]),
                     spread([a.get("testsFailed") for a in att]), spread([a.get("unrequested") for a in att]),
                     spread([a.get("protected") for a in att]), spread([a.get("confirmedCriticals") for a in att]),
                     sum(a["noResult"] for a in att)])
    effort = "; ".join(f"{t}: {(d or {}).get('inputEffort') or 'unavailable'}" + (f" ({d['note']})" if (d or {}).get("note") else "")
                       for t, d in sorted(tasks.items())) or "not stated"
    out.append(table("Human attention (primary)", ["Variant", "Results", "Defects left per run", "Cases failed", "Build failed",
                                                     "Tests failed", "Unrequested paths", "Protected inputs touched",
                                                     "Confirmed critical findings", "Steps with no result"], rows,
                     "Defects a person would still have to find and repair, per run. A result that does not build fails every case; "
                     "a step without a result fails every case and counts once more as a step with no result. The total covers runs "
                     "with every count known. Input effort per task: " + effort + "."))

    rows = []
    for v in variants:
        judged = [f for f in by[v] if "areas" in f or not f["delivered"]]
        passed = [f["areas"]["correctness"]["passed"] if "areas" in f else 0 for f in judged]
        plan = [f["areas"]["correctness"]["planned"] if "areas" in f else planned.get(f["task"]) for f in judged]
        clean = sum(1 for f in judged if "areas" in f and not f["areas"]["correctness"]["failed"] and not f["timedOut"])
        failed = sorted({c for f in judged if "areas" in f for c in f["areas"]["correctness"]["failedCases"]})
        rows.append([v, spread(passed), spread(plan), share(clean, len(judged)), ", ".join(failed) or "none"])
    out.append(table("Correctness", ["Variant", "Cases passed", "Cases planned", "Runs passing every case", "Cases failed in any run"], rows))

    rows = []
    for v in variants:
        judged = [f for f in by[v] if "areas" in f]
        rows.append([v, share(sum(f["areas"]["buildAndTests"]["build"]["passed"] for f in judged), len(judged)),
                     share(sum(f["areas"]["buildAndTests"]["tests"]["passed"] for f in judged), len(judged))])
    out.append(table("Build and tests", ["Variant", "Own build passes", "Own tests pass"], rows,
                     "The result's own build and tests, run by the judge."))

    rows = []
    for v in variants:
        judged = [f for f in by[v] if "areas" in f]
        rows.append([v, spread([len(f["areas"]["scope"]["changedPaths"]) for f in judged]),
                     spread([len(f["areas"]["scope"]["unrequested"]) for f in judged]),
                     share(sum(bool(f["areas"]["scope"]["protectedTouched"]) for f in judged), len(judged)),
                     ", ".join(sorted({p for f in judged for p in f["areas"]["scope"]["unrequested"]})) or "none"])
    out.append(table("Scope", ["Variant", "Changed paths", "Unrequested paths", "Runs touching a protected input", "Unrequested paths seen"], rows))

    rows = []
    for v in variants:
        reviewed = [f for f in by[v] if "review" in f]
        guesses = [g for f in reviewed for g in f["review"]["guesses"]]
        rows.append([v, share(len(reviewed), sum(f["delivered"] for f in by[v])), spread([f["review"]["reported"] for f in reviewed]),
                     spread([f["review"]["valid"] for f in reviewed]), spread([f["review"]["confirmed"] for f in reviewed]),
                     share(sum(g == v for g in guesses), len(guesses))])
    out.append(table("Code quality (secondary)", ["Variant", "Results reviewed", "Critical findings reported", "Passing the citation check",
                                                   "Confirmed", "Variant guessed right"], rows,
                     "A blinded review against the task's rubric; a critical finding counts when the citation check passes and a "
                     "second reviewer confirms it. All reviewers are Claude, the writers' family: a stated limit."))

    rows = [[v, share(sum(f["delivered"] and not f["timedOut"] for f in by[v]), len(by[v])), sum(not f["delivered"] for f in by[v]),
             sum(f["timedOut"] for f in by[v])] for v in variants]
    out.append(table("Delivery", ["Variant", "Delivered in time", "No result", "Timeouts"], rows))

    rows = []
    for v in variants:
        launched = [f for f in facts if f["variant"] == v]
        void = [f for f in launched if f["class"] == "void"]
        kinds = ", ".join(f"{k} {sum(f['kind'] == k for f in void)}" for k in sorted({f["kind"] for f in void})) or "none"
        rows.append([v, len(launched), share(len(void), len(launched)), kinds, sum(f["class"] == "invalid" for f in launched),
                     ", ".join(f["id"] for f in void if f["stopsSeries"]) or "none"])
    out.append(table("Void rate", ["Variant", "Launches", "Void", "Void by kind", "Invalid", "Stopping the series"], rows,
                     "A void launch is a technical failure: recorded, relaunched and never counted."))

    rows = []
    for v in variants:
        t = [f["tokens"] for f in by[v] if f.get("tokens")]
        rows.append([v, spread([f["wallSeconds"] for f in by[v]]), spread([x["responses"] for x in t]),
                     spread([x["inputTokens"] + x["cacheCreationTokens"] + x["cacheReadTokens"] for x in t]),
                     spread([x["outputTokens"] for x in t]), spread([x["subagentResponses"] for x in t]),
                     spread([round(f["costUsd"], 2) if f.get("costUsd") is not None else None for f in by[v]])])
    out.append(table("Cost (descriptive)", ["Variant", "Wall seconds", "Responses", "Input tokens incl. cache", "Output tokens",
                                             "Subagent responses", "USD as the CLI states it"], rows,
                     "Reported, not optimized. Tokens are counted by the suite over every transcript of a run, subagents included."))

    rows = [[f["id"], f["variant"], f["task"], f["class"], f["kind"] or "",
             json.dumps(f["evidence"], ensure_ascii=False) if f["evidence"] else ""] for f in facts]
    out.append(table("Launches", ["Launch", "Variant", "Task", "Class", "Kind", "Evidence"], rows))
    return "\n".join(out), {"format": FORMAT, "series": ledger.get("series"), "launches": facts}


def write(ledger_path, out_dir):
    out_dir = pathlib.Path(out_dir)
    if out_dir.exists():
        raise ValueError(f"{out_dir} exists")
    text, data = report(ledger_path)
    out_dir.mkdir(parents=True)
    (out_dir / "report.md").write_text(text, encoding="utf-8")
    (out_dir / "report.json").write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    return data
