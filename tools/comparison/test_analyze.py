"""The predeclared reading, exact intervals and Fisher thresholds."""

import unittest

import analyze

ALL = {c: True for c in analyze.CRITERIA}


def phase_a(m_chains, b_chains, m_s4, b_s4, m_errors=(), b_errors=()):
    rows = []
    for arm, chains, s4, errors in (("M", m_chains, m_s4, m_errors), ("B", b_chains, b_s4, b_errors)):
        for chain, ok in enumerate(chains, 1):
            for scenario in ("S1", "S2", "S3"):
                rows.append({"arm": arm, "chain": chain, "scenario": scenario, **ALL, "C2": bool(ok) or scenario != "S2",
                             "errors": [] if ok or scenario != "S2" else list(errors) or ["E04-admission"]})
        for chain, ok in enumerate(s4, 1):
            rows.append({"arm": arm, "chain": chain, "scenario": "S4", **ALL, "C6": bool(ok), "errors": [] if ok else ["C6"]})
    return rows


class AnalyzeTests(unittest.TestCase):
    def test_exact_intervals(self):
        self.assertEqual(analyze.clopper_pearson(3, 3), (0.2924, 1.0))
        self.assertEqual(analyze.clopper_pearson(0, 3), (0.0, 0.7076))
        self.assertEqual(analyze.clopper_pearson(6, 6)[0], 0.5407)
        self.assertEqual(analyze.clopper_pearson(12, 12)[0], 0.7354)
        low, high = analyze.clopper_pearson(1, 3)
        self.assertAlmostEqual(low, 0.0084, places=3)
        self.assertAlmostEqual(high, 0.9057, places=3)

    def test_fisher_thresholds_of_the_protocol(self):
        self.assertAlmostEqual(analyze.fisher_one_sided(6, 6, 2, 6), 0.0303, places=4)
        self.assertAlmostEqual(analyze.fisher_one_sided(5, 6, 1, 6), 0.0400, places=4)
        self.assertGreater(analyze.fisher_one_sided(6, 6, 3, 6), 0.05)
        self.assertAlmostEqual(analyze.fisher_one_sided(3, 3, 0, 3), 0.05, places=4)
        self.assertAlmostEqual(analyze.fisher_one_sided(5, 5, 1, 5), 0.0238, places=4)

    def test_readings(self):
        supports = analyze.read(phase_a([1, 1, 1], [1, 0, 0], [1, 1, 1], [0, 1, 0]), "A")
        self.assertEqual((supports["units"]["M"][:2], supports["units"]["B"][:2]), ([6, 6], [2, 6]))
        self.assertEqual(supports["reading"], "supports")
        self.assertEqual(analyze.read(phase_a([1, 1, 1], [1, 1, 0], [1, 1, 1], [1, 0, 0]), "A")["reading"], "tendency")
        self.assertEqual(analyze.read(phase_a([1, 1, 0], [1, 1, 0], [1, 1, 1], [1, 1, 1]), "A")["reading"], "no quality advantage shown")
        self.assertEqual(analyze.read(phase_a([1, 0, 0], [1, 1, 0], [1, 1, 1], [1, 1, 1]), "A")["reading"], "counts against Markitect")
        # An error type only M shows downgrades a significant result to a tendency.
        rows = phase_a([1, 1, 1], [1, 0, 0], [1, 1, 1], [0, 1, 0])
        rows.append({"arm": "M", "chain": 9, "scenario": "S9", **ALL, "C3": False, "errors": ["X01-due-date-range"], "counts": True})
        self.assertEqual(analyze.read(rows, "A")["reading"], "tendency")

    def test_additional_observations_do_not_count(self):
        rows = phase_a([1, 1, 1], [1, 0, 0], [1, 1, 1], [0, 1, 0])
        rows += [{"arm": "B", "chain": 4, "scenario": s, **ALL, "counts": False} for s in ("S1", "S2", "S3")]
        self.assertEqual(analyze.read(rows, "A")["units"]["B"][:2], [2, 6])

    def test_a_missing_chain_step_is_not_conforming(self):
        rows = [{"arm": arm, "chain": 1, "scenario": s, **ALL} for arm in ("M", "B") for s in ("T1", "T2", "T3", "T4")]
        rows = [r for r in rows if not (r["arm"] == "B" and r["scenario"] == "T4")]
        result = analyze.read(rows, "B")
        self.assertEqual((result["units"]["M"][:2], result["units"]["B"][:2]), ([1, 1], [0, 1]))

    def test_rank_sum_thresholds_of_addendum_c(self):
        # Complete separation is the only significant outcome with three chains per arm (1 of 20 splits).
        self.assertAlmostEqual(analyze.rank_sum_one_sided([0, 1, 2], [3, 4, 5]), 0.05)
        self.assertGreater(analyze.rank_sum_one_sided([0, 1, 4], [2, 3, 5]), 0.05)
        # Five chains per arm: complete separation is 1 of 252 splits.
        self.assertAlmostEqual(analyze.rank_sum_one_sided([0] * 5, [1] * 5), 1 / 252)
        # Ties are mid-ranked: equal values give no evidence either way.
        self.assertEqual(analyze.rank_sum_one_sided([2, 2, 2], [2, 2, 2]), 1.0)

    def test_phase_c_readings(self):
        def chains(arm, per_chain, step_errors=None):
            rows = []
            for chain, missed in enumerate(per_chain, 1):
                for i, step in enumerate(analyze.CHAIN_STEPS["C"]):
                    value = missed if i == 0 else 0
                    rows.append({"arm": arm, "chain": chain, "scenario": step, **ALL, "C2": value == 0,
                                 "missed_probe_families": value, "errors": (step_errors or []) if value else []})
            return rows
        supports = analyze.read_c(chains("M", [0, 0, 0]) + chains("B", [2, 3, 1], ["P-AUDIT"]))
        self.assertEqual((supports["missed_total"], supports["rank_sum_one_sided_p"], supports["reading"]),
                         ({"M": 0, "B": 6}, 0.05, "supports"))
        self.assertEqual(analyze.read_c(chains("M", [0, 2, 0]) + chains("B", [1, 1, 1]))["reading"], "tendency")
        self.assertEqual(analyze.read_c(chains("M", [0, 0, 0]) + chains("B", [0, 0, 0]))["reading"], "no quality advantage shown")
        self.assertEqual(analyze.read_c(chains("M", [1, 1, 0]) + chains("B", [0, 0, 1]))["reading"], "counts against Markitect")
        # An error type only M shows downgrades a significant result.
        rows = chains("M", [0, 0, 0]) + chains("B", [2, 3, 1], ["P-AUDIT"])
        rows[1].update(C6=False, errors=["C9-census"])
        self.assertEqual(analyze.read_c(rows)["reading"], "tendency")
        # A step without a candidate counts its blocker class separately.
        rows = chains("M", [30, 0, 0]) + chains("B", [0, 0, 0])
        rows[0]["blocker"] = "justified"
        result = analyze.read_c(rows)
        self.assertEqual((result["blockers"]["M"], result["reading"]), ({"justified": 1, "false": 0}, "counts against Markitect"))


if __name__ == "__main__":
    unittest.main()
