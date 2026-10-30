"""The report's acceptance areas: one table per criterion of concept 2.0 §15.

The ledger (report.py) may carry, besides its launches:

  launches[].repetition, launches[].step   for pairing and for chains
  chains: [{"id": "k-1", "variant": "K", "task": "makerspace", "repetition": 1,
            "judgment": "JUDGE_CHAIN_OUT_DIR", "verifyFull": "RESULT.json"}]
  contrasts: [{"variant": "2.0", "baseline": "K", "endpoint": "casesPassedShare",
               "minimumDifference": 0.05, "blockBy": "change"}]
               blockBy "change" reads the interval over each change's mean difference (task and step,
               over its repetitions), so that a change's repetitions do not count as independent pairs
  tasks.NAME.planted: "defect" or "gap", for a planted task

1. Correct           every landed change passes every hidden case
2. No drift          after each chain, the judge's drift checks of its end result, the
                     regressions along it and the variant's own verify --full, where it has one
3. Nothing skipped   every Result's items have a verdict with evidence (results.py)
4. Stops right       planted defects do not land, harmless gaps do (judging.stops)
5. Trustworthy       the paired reading of each contrast (reading.py), read a second time with every
                     unpaired unit scored as the change not done
"""
import json
import pathlib
import statistics

import reading
import results

# Endpoints of a contrast: the value of one launch, and whether higher is better.
ENDPOINTS = {
    "casesPassedShare": ("hidden cases passed, as a share of planted cases", True),
    "casesFailed": ("hidden cases failed, earlier steps' cases included", False),
    "defectsLeft": ("defects left for a person (human attention)", False),
}


def number(value):
    return "n/a" if value is None else (f"{value:.3f}".rstrip("0").rstrip(".") if isinstance(value, float) else str(value))


def start_scores(facts):
    """What each task and step's start scores, by the protocol's rule for a change not done: every earlier step's case
    passed and the step's own new cases failed. From the byStep of any judged launch of that task and step, whose last
    entry is the step itself (judging.checked_series)."""
    out = {}
    for f in facts:
        by = (f.get("areas") or {}).get("correctness", {}).get("byStep")
        if by:
            planned, own = sum(s["planned"] for s in by.values()), list(by.values())[-1]["planned"]
            out.setdefault((f["task"], f.get("step")), {"planned": planned, "passed": planned - own, "failed": own})
    return out


def not_done(name, start):
    """An endpoint's value for a change not done, scored as its start (start_scores), which also counts once as a step
    with no result; None when no judged launch of the task and step gives the counts."""
    if not start:
        return None
    if name == "casesPassedShare":
        return start["passed"] / start["planned"] if start["planned"] else None
    return float(start["failed"] + (name == "defectsLeft"))


def endpoint(f, name):
    if f.get("planted") or f["class"] != "result":
        return None
    if not f["delivered"]:
        return not_done(name, f.get("startScore"))
    if name == "casesPassedShare":
        if "areas" not in f:
            return None
        c = f["areas"]["correctness"]
        return c["passed"] / c["planned"] if c["planned"] else None
    att = f.get("attention")
    if name == "casesFailed":
        return float(att["casesFailed"]) if att and att.get("casesFailed") is not None else None
    return float(sum(att.values())) if att and None not in att.values() else None


def correct(facts, variants, table):
    rows = []
    for v in variants:
        landed = [f for f in facts if f["variant"] == v and f["class"] == "result" and not f.get("planted") and f.get("landed")]
        judged = [f for f in landed if "areas" in f]
        bad = [f["id"] for f in judged if f["areas"]["correctness"]["failed"]]
        rows.append([v, len(landed), len(judged), f"{len(judged) - len(bad)}/{len(judged)}" if judged else "n/a", ", ".join(bad) or "none"])
    return table("§15.1 Correct", ["Variant", "Landed changes", "Judged", "Passing every hidden case", "Landed changes failing a case"],
                 rows, "A landed change is a delivered result that changes the start. Hidden cases include every earlier step's cases.")


def no_drift(base, chains, table):
    rows = []
    for chain in chains:
        record = results.read(base / chain["judgment"] / "chain-judgment.json") if chain.get("judgment") else None
        drift = (record or {}).get("finalDrift")
        verify = None
        if chain.get("verifyFull"):
            page = results.read(base / chain["verifyFull"])
            checked = results.check(page)
            state = (page or {}).get("state")
            if checked["problems"]:
                verify = "unreadable"
            elif state not in ("verified", "findings"):
                verify = f"{state or 'no state'}: no verdict"  # a verify that did not check every item is no clean bill
            else:
                verify = f"{checked['findings']} findings"
        unaccounted = drift.get("unaccounted") if drift else None
        rows.append([chain["id"], chain["variant"], chain["task"], "yes" if (record or {}).get("lastStepJudged") else "no",
                     drift["step"] if drift else "n/a", len(drift["forgottenRules"]) if drift else "n/a",
                     len(drift["staleFiles"]) if drift else "n/a", "unknown" if drift and unaccounted is None else (len(unaccounted) if drift else "n/a"),
                     len(record["regressions"]) if record else "n/a", verify or "no verify --full (slot empty)"])
    return table("§15.2 No drift", ["Chain", "Variant", "Task", "Reached and judged its last step", "Drift judged at", "Forgotten rules",
                                    "Stale files", "Unaccounted files", "Regressions along the chain", "Own verify --full"], rows,
                 "After the series, the judge's drift checks of the chain's end result, the same for every variant: earlier steps' cases "
                 "that fail (forgotten rules), retired terms left in the source (stale files) and files nothing accounts for. "
                 "An independent verify --full of the chain's end state goes in its slot, for every variant whose chain has one; a "
                 "verify that did not check every item shows its state.")


def nothing_skipped(facts, variants, table):
    rows = []
    for v in variants:
        mine = [f for f in facts if f["variant"] == v and f["class"] == "result"]
        checked = [f["results"] for f in mine if f.get("results")]
        if not checked:
            rows.append([v, len(mine), "n/a", 0, "n/a", "n/a", "n/a", "n/a", "n/a", "no Results: the variant keeps no checklist"])
            continue
        # A variant that returns Results returns one for every delivered launch; a launch without one skipped everything.
        missing = [f["id"] for f in mine if f.get("delivered") and not f.get("results")]
        all_results = [r for c in checked for r in c["results"]]
        rows.append([v, len(mine), len(missing), len(all_results), sum(r.get("items", 0) for r in all_results),
                     sum(len(r.get("skipped", [])) for r in all_results), sum(len(r.get("unsupported", [])) for r in all_results),
                     sum(len(r.get("unapproved", [])) for r in all_results), sum(bool(r.get("checksIncomplete")) for r in all_results),
                     "; ".join([f"{i}: no Result" for i in missing] + [p for c in checked for p in c["problems"]]) or "none"])
    return table("§15.3 Nothing skipped", ["Variant", "Launches", "Delivered without a Result", "Results", "Items", "Without verdict",
                                           "Without evidence", "Without approving checker", "Checks not on the whole repository", "Problems"],
                 rows, "Each Result a variant returns in results/ (results.py), checked item by item. A variant that returns Results "
                       "must return one for every delivered launch.")


def stops_right(facts, variants, table):
    rows = []
    for v in variants:
        mine = [f for f in facts if f["variant"] == v and f["class"] == "result" and f.get("plantKind")]
        row = [v]
        for kind in ("defect", "gap"):
            these = [f for f in mine if f["plantKind"] == kind]
            right = sum(bool(f.get("stops", {}).get("correct")) for f in these)
            row += [f"{right}/{len(these)}" if these else "n/a", sum(not f["delivered"] for f in these)]
        rows.append(row)
    return table("§15.4 Stops right", ["Variant", "Planted defects without landing", "Defects with no result",
                                       "Harmless gaps landed", "Gaps with no result"], rows,
                 "Scored the same way for every variant: a result equal to its start did not land. A launch with no result has no "
                 "outcome and counts as wrong. The plants were recorded before the runs.")


def paired_reading(values, contrast, higher):
    """The paired reading of the units both variants have a value for; blockBy "change" reads it over each change's mean."""
    diffs = [(key, (pair[contrast["variant"]] - pair[contrast["baseline"]]) * (1 if higher else -1))
             for key, pair in sorted(values.items(), key=lambda kv: json.dumps(kv[0], default=str))
             if contrast["variant"] in pair and contrast["baseline"] in pair]
    if contrast.get("blockBy") == "change":
        blocks = {}
        for (task, step, _), diff in diffs:
            blocks.setdefault((task, step), []).append(diff)
        return {**reading.paired([statistics.fmean(b) for b in blocks.values()], float(contrast["minimumDifference"])),
                "pairs": len(diffs), "blocks": len(blocks)}
    return {**reading.paired([d for _, d in diffs], float(contrast["minimumDifference"])), "blocks": None}


def interval(result):
    return "n/a" if result["low"] is None else f"[{number(result['low'])}, {number(result['high'])}]"


def trustworthy(facts, contrasts, table):
    """Each contrast read twice when units are unpaired: over the pairs, and with every unpaired unit scored as the
    change not done for the variant that has no result there, so that units which end without a result cannot decide the
    verdict by dropping out. Yes only when both readings say yes, no only when both say no, otherwise undecided; so also
    when an unpaired unit has no start score to fill it with."""
    rows, data = [], []
    starts = start_scores(facts)
    for contrast in contrasts:
        name = contrast["endpoint"]
        if name not in ENDPOINTS:
            raise ValueError(f"unknown endpoint {name}; known: {', '.join(ENDPOINTS)}")
        higher, arms = ENDPOINTS[name][1], (contrast["variant"], contrast["baseline"])
        values = {}
        for f in facts:
            if f["variant"] in arms and f.get("repetition") is not None:
                value = endpoint(f, name)
                if value is not None:
                    values.setdefault((f["task"], f.get("step"), f["repetition"]), {})[f["variant"]] = value
        result = paired_reading(values, contrast, higher)
        unpaired = sum(len(pair) == 1 for pair in values.values())
        verdict, second = result["atLeastAsGood"], None
        if unpaired:
            filled = {}
            for key, pair in values.items():
                missing = not_done(name, starts.get(key[:2]))
                filled[key] = {**({arm: missing for arm in arms} if missing is not None else {}), **pair}
            second = paired_reading(filled, contrast, higher)
            unfilled = any(len(pair) == 1 for pair in filled.values())
            verdict = verdict if verdict == second["atLeastAsGood"] and not unfilled else None
        data.append({**contrast, **result, "unpaired": unpaired, "notDone": second, "atLeastAsGood": verdict})
        rows.append([f"{contrast['variant']} vs {contrast['baseline']}", ENDPOINTS[name][0], number(contrast["minimumDifference"]),
                     result["pairs"] if result["blocks"] is None else f"{result['pairs']} in {result['blocks']} changes", unpaired,
                     number(result["mean"]), interval(result), result["reading"],
                     "none unpaired" if second is None else f"{second['reading']}, {interval(second)}",
                     {True: "yes", False: "no"}.get(verdict, f"undecided ({unpaired} unpaired)" if unpaired else "undecided")])
    text = table("§15.5 At least as trustworthy as K", ["Contrast", "Endpoint", "Minimum meaningful difference", "Pairs", "Unpaired",
                                                         "Mean difference", "95 % interval", "Reading",
                                                         "Reading with unpaired units as not done", "At least as good"], rows,
                 "Pairs are the two variants' launches of the same task, step and repetition; a positive difference favors the "
                 "variant. Where pairs are counted in changes, the interval is read over each change's mean difference. At least "
                 "as good: the interval's lower end lies above minus the minimum meaningful difference. Where units are unpaired, "
                 "the contrast is read a second time with each scored as the change not done for the variant without a result: "
                 "its start, which passes the earlier steps' cases and fails the step's own. Yes or no only when both readings "
                 "agree; otherwise undecided.")
    return text, data


def tables(ledger, base, facts, table):
    variants = sorted({f["variant"] for f in facts})
    out = ["# Acceptance (concept 2.0 §15)", "", "One table per criterion; the owner decides on them.", "",
           correct(facts, variants, table), no_drift(pathlib.Path(base), ledger.get("chains") or [], table),
           nothing_skipped(facts, variants, table), stops_right(facts, variants, table)]
    text, contrasts = trustworthy(facts, ledger.get("contrasts") or [], table)
    out.append(text)
    return "\n".join(out), {"contrasts": contrasts}
