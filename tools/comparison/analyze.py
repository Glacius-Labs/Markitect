"""Apply the comparison protocol's predeclared reading to a run ledger.

The ledger lists every counting run with the facts the protocol's criteria
need, taken from the runner or Apply record, the assessment, the confirmed
review findings and the census. Nothing here judges a candidate; it only
combines the recorded criteria C1-C6 into conforming runs, independent units,
exact intervals, the one-sided Fisher exact test and the reading of section 8
(Phase A) or B5 (Phase B).

  python -I -B analyze.py LEDGER.json [--phase A|B]

Ledger rows: {"arm": "M"|"B", "chain": 1..5, "scenario": "S1".."S4"|"T1".."T4",
"counts": true|false, "C1".."C6": true|false, "errors": [error types],
"unsupported_success": bool, "false_blocker": bool}. A missing criterion is
not conforming.
"""
import argparse
import json
import math
import pathlib
import sys

CRITERIA = ("C1", "C2", "C3", "C4", "C5", "C6")
CHAIN_STEPS = {"A": ("S1", "S2", "S3"), "B": ("T1", "T2", "T3", "T4")}


def conforming(row):
    return all(row.get(c) is True for c in CRITERIA)


def _binomial_tail(k, n, p):
    """P(X >= k) for X ~ Binomial(n, p); increasing in p."""
    return sum(math.comb(n, i) * p ** i * (1 - p) ** (n - i) for i in range(k, n + 1))


def _bisect(f, target):
    """The p in [0, 1] where the increasing function f reaches target."""
    lo, hi = 0.0, 1.0
    for _ in range(100):
        mid = (lo + hi) / 2
        if f(mid) < target:
            lo = mid
        else:
            hi = mid
    return (lo + hi) / 2


def clopper_pearson(k, n, level=0.95):
    """Exact two-sided interval: P(X >= k | lower) = P(X <= k | upper) = (1 - level) / 2."""
    if n == 0:
        return (None, None)
    alpha = (1 - level) / 2
    lower = 0.0 if k == 0 else _bisect(lambda p: _binomial_tail(k, n, p), alpha)
    upper = 1.0 if k == n else _bisect(lambda p: _binomial_tail(k + 1, n, p), 1 - alpha)
    return (round(lower, 4), round(upper, 4))


def fisher_one_sided(k_m, n_m, k_b, n_b):
    """P(M successes >= k_m) under the hypergeometric null with fixed margins."""
    total = k_m + k_b
    n = n_m + n_b
    denominator = math.comb(n, total)
    return sum(math.comb(n_m, x) * math.comb(n_b, total - x) for x in range(k_m, min(n_m, total) + 1)) / denominator


def units(rows, phase):
    """Independent units: whole chains, and for Phase A each S4 run."""
    out = {"M": [], "B": []}
    steps = CHAIN_STEPS[phase]
    for arm in out:
        chains = sorted({r["chain"] for r in rows if r["arm"] == arm and r["scenario"] in steps})
        for chain in chains:
            chain_rows = {r["scenario"]: r for r in rows if r["arm"] == arm and r["chain"] == chain and r["scenario"] in steps}
            out[arm].append(all(s in chain_rows and conforming(chain_rows[s]) for s in steps))
        if phase == "A":
            out[arm] += [conforming(r) for r in rows if r["arm"] == arm and r["scenario"] == "S4"]
    return out


def read(rows, phase):
    rows = [r for r in rows if r.get("counts", True)]
    u = units(rows, phase)
    k_m, n_m, k_b, n_b = sum(u["M"]), len(u["M"]), sum(u["B"]), len(u["B"])
    p = fisher_one_sided(k_m, n_m, k_b, n_b) if n_m and n_b else None
    scenarios = sorted({r["scenario"] for r in rows})
    per_scenario = {s: {arm: [sum(conforming(r) for r in rows if r["arm"] == arm and r["scenario"] == s),
                              sum(1 for r in rows if r["arm"] == arm and r["scenario"] == s)] for arm in ("M", "B")}
                    for s in scenarios}
    errors = {arm: sorted({e for r in rows if r["arm"] == arm for e in r.get("errors", [])}) for arm in ("M", "B")}
    m_only = sorted(set(errors["M"]) - set(errors["B"]))
    m_not_below = all(v["M"][0] >= v["B"][0] for v in per_scenario.values())
    if k_m > k_b:
        reading = "supports" if p is not None and p <= 0.05 and m_not_below and not m_only else "tendency"
    elif k_m == k_b:
        reading = "no quality advantage shown"
    else:
        reading = "counts against Markitect"
    runs = {arm: [sum(conforming(r) for r in rows if r["arm"] == arm), sum(1 for r in rows if r["arm"] == arm)] for arm in ("M", "B")}
    return {"phase": phase, "units": {"M": [k_m, n_m, clopper_pearson(k_m, n_m)], "B": [k_b, n_b, clopper_pearson(k_b, n_b)]},
            "fisher_one_sided_p": None if p is None else round(p, 4), "per_scenario": per_scenario,
            "runs": {arm: v + [clopper_pearson(*v)] for arm, v in runs.items()},
            "error_types": errors, "m_only_error_types": m_only, "m_not_below_in_any_scenario": m_not_below,
            "unsupported_success": {arm: sum(1 for r in rows if r["arm"] == arm and r.get("unsupported_success")) for arm in ("M", "B")},
            "false_blockers": {arm: sum(1 for r in rows if r["arm"] == arm and r.get("false_blocker")) for arm in ("M", "B")},
            "reading": reading}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("ledger")
    p.add_argument("--phase", choices=["A", "B"], default="A")
    a = p.parse_args(argv)
    rows = json.loads(pathlib.Path(a.ledger).read_text(encoding="utf-8"))
    print(json.dumps(read(rows, a.phase), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
