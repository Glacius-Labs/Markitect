"""Apply the comparison protocol's predeclared reading to a run ledger.

The ledger lists every counting run with the facts the protocol's criteria
need, taken from the runner or Apply record, the assessment, the confirmed
review findings and the census. Nothing here judges a candidate; it only
combines the recorded criteria C1-C6 into conforming runs, independent units,
exact intervals, the one-sided Fisher exact test and the reading of section 8
(Phase A) or B5 (Phase B). Phase C (addendum C6) reads the chains' missed
obligations instead: the sum over W0-C9 of failed probe families per chain,
compared with an exact one-sided rank-sum (Mann-Whitney) permutation test.

  python -I -B analyze.py LEDGER.json [--phase A|B|C]

Ledger rows: {"arm": "M"|"B", "chain": 1..5, "scenario": "S1".."S4"|"T1".."T4"|"W0".."C9",
"counts": true|false, "C1".."C6": true|false, "errors": [error types],
"unsupported_success": bool, "false_blocker": bool}; Phase C rows also carry
"missed_probe_families" (int; every family of the step when there is no
candidate) and "blocker" (null, "justified" or "false"). A missing criterion is
not conforming.
"""
import argparse
import itertools
import json
import math
import pathlib
import sys

CRITERIA = ("C1", "C2", "C3", "C4", "C5", "C6")
CHAIN_STEPS = {"A": ("S1", "S2", "S3"), "B": ("T1", "T2", "T3", "T4"),
               "C": ("W0", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8", "C9")}


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


def rank_sum_one_sided(m, b):
    """Exact one-sided Mann-Whitney test that M's values are lower.

    P(rank sum of M <= observed) over every split of the pooled values into
    groups of the arms' sizes, with mid-ranks for ties.
    """
    pooled = list(m) + list(b)
    order = sorted(range(len(pooled)), key=lambda i: pooled[i])
    ranks = [0.0] * len(pooled)
    i = 0
    while i < len(order):
        j = i
        while j + 1 < len(order) and pooled[order[j + 1]] == pooled[order[i]]:
            j += 1
        for k in range(i, j + 1):
            ranks[order[k]] = (i + j) / 2 + 1
        i = j + 1
    observed = sum(ranks[:len(m)])
    splits = list(itertools.combinations(range(len(pooled)), len(m)))
    return sum(1 for s in splits if sum(ranks[k] for k in s) <= observed + 1e-9) / len(splits)


def complete_chains(rows):
    """Counted Phase C rows by arm and chain; refuse anything the rule cannot read.

    Every counted chain needs exactly one row per step with an integer
    missed_probe_families, and both arms need the same number of chains: a gap
    or a duplicate would otherwise count as zero or twice.
    """
    steps = CHAIN_STEPS["C"]
    chains = {"M": {}, "B": {}}
    for r in rows:
        if not r.get("counts", True) or r["scenario"] not in steps:
            continue
        where = f'{r["arm"]} chain {r["chain"]} {r["scenario"]}'
        if not isinstance(r.get("missed_probe_families"), int) or isinstance(r["missed_probe_families"], bool):
            raise ValueError(where + ": missed_probe_families must be an integer")
        chain = chains[r["arm"]].setdefault(r["chain"], {})
        if r["scenario"] in chain:
            raise ValueError(where + ": duplicate row")
        chain[r["scenario"]] = r
    for arm, by_chain in chains.items():
        for number, chain in by_chain.items():
            missing = [s for s in steps if s not in chain]
            if missing:
                raise ValueError(f"{arm} chain {number} is incomplete: missing {', '.join(missing)}")
    if not chains["M"] or len(chains["M"]) != len(chains["B"]):
        raise ValueError(f'not evaluable: {len(chains["M"])} M chains, {len(chains["B"])} B chains')
    return chains


def read_c(rows):
    """Addendum C6: missed obligations per chain, the per-step check and the error-type downgrade."""
    chains = complete_chains(rows)
    rows = [r for arm in chains.values() for chain in arm.values() for r in chain.values()]
    m = [sum(r["missed_probe_families"] for r in chains["M"][c].values()) for c in sorted(chains["M"])]
    b = [sum(r["missed_probe_families"] for r in chains["B"][c].values()) for c in sorted(chains["B"])]
    p = rank_sum_one_sided(m, b) if m and b else None
    present = CHAIN_STEPS["C"]
    per_step = {s: {arm: {"missed": sum(r["missed_probe_families"] for r in rows if r["arm"] == arm and r["scenario"] == s),
                          "conforming": [sum(conforming(r) for r in rows if r["arm"] == arm and r["scenario"] == s),
                                         sum(1 for r in rows if r["arm"] == arm and r["scenario"] == s)]}
                    for arm in ("M", "B")} for s in present}
    errors = {arm: sorted({e for r in rows if r["arm"] == arm for e in r.get("errors", [])}) for arm in ("M", "B")}
    m_only = sorted(set(errors["M"]) - set(errors["B"]))
    m_not_worse = all(v["M"]["missed"] <= v["B"]["missed"] for v in per_step.values())
    if sum(m) < sum(b):
        reading = "supports" if p is not None and p <= 0.05 and m_not_worse and not m_only else "tendency"
    elif sum(m) == sum(b):
        reading = "no quality advantage shown"
    else:
        reading = "counts against Markitect"
    runs = {arm: [sum(conforming(r) for r in rows if r["arm"] == arm), sum(1 for r in rows if r["arm"] == arm)] for arm in ("M", "B")}
    blockers = {arm: {kind: sum(1 for r in rows if r["arm"] == arm and r.get("blocker") == kind) for kind in ("justified", "false")}
                for arm in ("M", "B")}
    return {"phase": "C", "missed_per_chain": {"M": m, "B": b}, "missed_total": {"M": sum(m), "B": sum(b)},
            "rank_sum_one_sided_p": None if p is None else round(p, 4), "per_step": per_step,
            "m_not_worse_at_any_step": m_not_worse, "conforming_steps": {arm: v + [clopper_pearson(*v)] for arm, v in runs.items()},
            "error_types": errors, "m_only_error_types": m_only, "blockers": blockers,
            "unsupported_success": {arm: sum(1 for r in rows if r["arm"] == arm and r.get("unsupported_success")) for arm in ("M", "B")},
            "reading": reading}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("ledger")
    p.add_argument("--phase", choices=["A", "B", "C"], default="A")
    a = p.parse_args(argv)
    rows = json.loads(pathlib.Path(a.ledger).read_text(encoding="utf-8"))
    print(json.dumps(read_c(rows) if a.phase == "C" else read(rows, a.phase), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
