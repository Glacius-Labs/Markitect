"""The report's acceptance areas: one table per criterion of concept 2.0 §15.

The ledger (report.py) may carry, besides its launches:

  launches[].repetition, launches[].step   for pairing and for chains
  chains: [{"id": "k-1", "variant": "K", "task": "makerspace", "repetition": 1,
            "judgment": "JUDGE_CHAIN_OUT_DIR", "verifyFull": "RESULT.json"}]
  contrasts: [{"variant": "2.0", "baseline": "K", "endpoint": "casesPassedShare",
               "minimumDifference": 0.05}]
  tasks.NAME.planted: "defect" or "gap", for a planted task

1. Correct           every landed change passes every hidden case
2. No drift          after each chain, the judge's drift checks of its end result, the
                     regressions along it and the variant's own verify --full, where it has one
3. Nothing skipped   every Result's items have a verdict with evidence (results.py)
4. Stops right       planted defects do not land, harmless gaps do (judging.stops)
5. Trustworthy       the paired reading of each contrast (reading.py)
"""
import json
import pathlib

import reading
import results

# Endpoints of a contrast: the value of one launch, and whether higher is better.
ENDPOINTS = {
    "casesPassedShare": ("hidden cases passed, as a share of planted cases", True),
    "defectsLeft": ("defects left for a person (human attention)", False),
}


def number(value):
    return "n/a" if value is None else (f"{value:.3f}".rstrip("0").rstrip(".") if isinstance(value, float) else str(value))


def endpoint(f, name):
    if f.get("planted") or f["class"] != "result":
        return None
    if name == "casesPassedShare":
        if not f["delivered"]:
            return 0.0
        if "areas" not in f:
            return None
        c = f["areas"]["correctness"]
        return c["passed"] / c["planned"] if c["planned"] else None
    att = f.get("attention")
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
            checked = results.check(results.read(base / chain["verifyFull"]))
            verify = "unreadable" if checked["problems"] else f"{checked['findings']} findings"
        unaccounted = drift.get("unaccounted") if drift else None
        rows.append([chain["id"], chain["variant"], chain["task"], "yes" if (record or {}).get("lastStepJudged") else "no",
                     drift["step"] if drift else "n/a", len(drift["forgottenRules"]) if drift else "n/a",
                     len(drift["staleFiles"]) if drift else "n/a", "unknown" if drift and unaccounted is None else (len(unaccounted) if drift else "n/a"),
                     len(record["regressions"]) if record else "n/a", verify or "no verify --full (slot empty)"])
    return table("§15.2 No drift", ["Chain", "Variant", "Task", "Reached and judged its last step", "Drift judged at", "Forgotten rules",
                                    "Stale files", "Unaccounted files", "Regressions along the chain", "Own verify --full"], rows,
                 "After the series, the judge's drift checks of the chain's end result, the same for every variant: earlier steps' cases "
                 "that fail (forgotten rules), retired terms left in the source (stale files) and files nothing accounts for. "
                 "The 2.0 arm's own independent verify --full goes in its slot; K has none.")


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


def trustworthy(facts, contrasts, table):
    rows, data = [], []
    for contrast in contrasts:
        name = contrast["endpoint"]
        if name not in ENDPOINTS:
            raise ValueError(f"unknown endpoint {name}; known: {', '.join(ENDPOINTS)}")
        higher = ENDPOINTS[name][1]
        values = {}
        for f in facts:
            if f["variant"] in (contrast["variant"], contrast["baseline"]) and f.get("repetition") is not None:
                value = endpoint(f, name)
                if value is not None:
                    values.setdefault((f["task"], f.get("step"), f["repetition"]), {})[f["variant"]] = value
        diffs = [(pair[contrast["variant"]] - pair[contrast["baseline"]]) * (1 if higher else -1)
                 for _, pair in sorted(values.items(), key=lambda kv: json.dumps(kv[0], default=str))
                 if contrast["variant"] in pair and contrast["baseline"] in pair]
        result = reading.paired(diffs, float(contrast["minimumDifference"]))
        unpaired = sum(len(pair) == 1 for pair in values.values())
        data.append({**contrast, **result, "unpaired": unpaired})
        rows.append([f"{contrast['variant']} vs {contrast['baseline']}", ENDPOINTS[name][0], number(contrast["minimumDifference"]),
                     result["pairs"], unpaired, number(result["mean"]),
                     "n/a" if result["low"] is None else f"[{number(result['low'])}, {number(result['high'])}]", result["reading"],
                     {True: "yes", False: "no", None: "undecided"}[result["atLeastAsGood"]]])
    text = table("§15.5 At least as trustworthy as K", ["Contrast", "Endpoint", "Minimum meaningful difference", "Pairs", "Unpaired",
                                                         "Mean difference", "95 % interval", "Reading", "At least as good"], rows,
                 "Pairs are the two variants' launches of the same task, step and repetition; a positive difference favors the "
                 "variant. At least as good: the interval's lower end lies above minus the minimum meaningful difference.")
    return text, data


def tables(ledger, base, facts, table):
    variants = sorted({f["variant"] for f in facts})
    out = ["# Acceptance (concept 2.0 §15)", "", "One table per criterion; the owner decides on them.", "",
           correct(facts, variants, table), no_drift(pathlib.Path(base), ledger.get("chains") or [], table),
           nothing_skipped(facts, variants, table), stops_right(facts, variants, table)]
    text, contrasts = trustworthy(facts, ledger.get("contrasts") or [], table)
    out.append(text)
    return "\n".join(out), {"contrasts": contrasts}
