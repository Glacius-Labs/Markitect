"""Offline tests of the acceptance areas (concept 2.0 §15): the Result check, the paired reading and the report's tables."""
import json
import pathlib
import shutil
import sys
import tempfile
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import reading  # noqa: E402
import report  # noqa: E402
import results  # noqa: E402

RUN = HERE / "testdata" / "failures" / "result-k03"


def item(name, **change):
    return {"manager": "billing", "item": name, "part": "own", "verdict": "conforms", "evidence": "tests pass", "checker": "billing/checker",
            "approved": True, **change}


def result_page(outcome="landed", items=None, **change):
    return {"format": 1, "run": "r1", "command": "apply", "outcome": outcome, "requiredChecks": {"ran": True, "wholeRepository": True},
            "items": items if items is not None else [item("billing/invoice")], "notParticipating": [], **change}


class ResultCheckTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)

    def tearDown(self):
        self._tmp.cleanup()

    def run_with(self, *pages, changed=True):
        run = self.tmp / f"run-{len(list(self.tmp.iterdir()))}"
        shutil.copytree(RUN, run)
        if not changed:
            record = json.loads((run / "suite.json").read_text())
            record["result"]["changedPaths"] = []
            (run / "suite.json").write_text(json.dumps(record))
        (run / "variant" / "results").mkdir()
        for i, page in enumerate(pages):
            (run / "variant" / "results" / f"r{i}.json").write_text(json.dumps(page))
        return run

    def test_a_complete_result_passes(self):
        checked = results.check_run(self.run_with(result_page()))
        self.assertTrue(checked["complete"], checked)
        self.assertEqual(checked["results"][0]["items"], 1)

    def test_skipped_unsupported_and_unapproved_items_are_found(self):
        page = result_page(items=[item("a", verdict=None), item("b", evidence=" "), item("c", approved=False), item("d")],
                           requiredChecks={"ran": True, "wholeRepository": False})
        checked = results.check_run(self.run_with(page))
        self.assertFalse(checked["complete"])
        r = checked["results"][0]
        self.assertEqual((r["skipped"], r["unsupported"], r["unapproved"], r["checksIncomplete"]),
                         (["billing:a"], ["billing:b"], ["billing:c"], True))

    def test_the_outcome_must_match_what_the_run_delivered(self):
        self.assertIn("no Result landed", results.check_run(self.run_with(result_page("not-landed")))["problems"][0])
        self.assertIn("no change", results.check_run(self.run_with(result_page(), changed=False))["problems"][0])
        self.assertTrue(results.check_run(self.run_with(result_page("not-landed"), changed=False))["complete"])
        self.assertIn("not a Result", results.check_run(self.run_with({"format": 2}))["problems"][0])
        self.assertIn("no checklist items", results.check_run(self.run_with(result_page(items=[])))["problems"][0])

    def test_a_run_without_results_has_nothing_to_check(self):
        self.assertIsNone(results.check_run(RUN))


class ReadingTest(unittest.TestCase):
    def test_the_t_quantiles_match_the_tables(self):
        for df, value in ((1, 12.706), (4, 2.776), (9, 2.262), (30, 2.042)):
            self.assertAlmostEqual(reading.t_quantile(0.975, df), value, places=3)

    def test_each_reading(self):
        self.assertEqual(reading.paired([0.1, 0.2, 0.15, 0.05], 0.05)["reading"], "better")
        self.assertEqual(reading.paired([-0.1, -0.2, -0.15, -0.05], 0.05)["reading"], "worse")
        excluded = reading.paired([0.0, 0.01, -0.01, 0.0], 0.05)
        self.assertEqual(excluded["reading"], "a meaningful advantage is excluded")
        self.assertTrue(excluded["atLeastAsGood"])
        wide = reading.paired([0.3, -0.3, 0.2, -0.25], 0.05)
        self.assertEqual(wide["reading"], "undecided")
        self.assertFalse(wide["atLeastAsGood"])
        self.assertEqual(reading.paired([0.1], 0.05)["reading"], "undecided (fewer than two pairs)")


def judgment(commit, failed=0, planned=19, stops=None, areas=True):
    value = {"valid": True, "commit": commit, "problems": [], "areas": None}
    if areas:
        value["areas"] = {"correctness": {"planned": planned, "passed": planned - failed, "failed": failed, "failedCases": [f"c{i}" for i in range(failed)]},
                          "buildAndTests": {"build": {"passed": True}, "tests": {"passed": True}},
                          "scope": {"changedPaths": [], "unrequested": [], "protectedTouched": []}}
    if stops:
        value["stops"] = stops
    return value


class AcceptanceReportTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        commit = json.loads((RUN / "suite.json").read_text())["result"]["commit"]
        launches = []

        def add(name, variant, task, repetition=None, step=None, page=None, **judged):
            shutil.copytree(RUN, self.tmp / name)
            if page:
                (self.tmp / name / "variant" / "results").mkdir()
                (self.tmp / name / "variant" / "results" / "r.json").write_text(json.dumps(page))
            (self.tmp / (name + "-j")).mkdir()
            (self.tmp / (name + "-j") / "judgment.json").write_text(json.dumps(judgment(commit, **judged)))
            launches.append({"id": name, "variant": variant, "task": task, "run": name, "judgment": name + "-j",
                             "repetition": repetition, "step": step})

        # Three repetitions of one step: 2.0 passes everything, K fails 1, 2 and 0 of 19 cases.
        for rep, k_failed in ((1, 1), (2, 2), (3, 0)):
            add(f"v-{rep}", "2.0", "makerspace", rep, "C3", page=result_page())
            add(f"k-{rep}", "K", "makerspace", rep, "C3", failed=k_failed)
        add("v-skip", "2.0", "makerspace", 4, "C3", page=result_page(items=[item("x", verdict=None)]))
        planted_stop = {"kind": "defect", "expected": "no-landing", "landed": True, "correct": False, "recordedBeforeRun": True}
        add("v-plant", "2.0", "plant-1", areas=False, stops={**planted_stop, "landed": False, "correct": True})
        add("k-plant", "K", "plant-1", areas=False, stops=planted_stop)
        chain = self.tmp / "chain-k"
        chain.mkdir()
        (chain / "chain-judgment.json").write_text(json.dumps({
            "steps": [], "regressions": [{"case": "c1", "passedAt": "W0", "failsAt": "C2"}], "lastStepJudged": True,
            "finalDrift": {"step": "C9", "forgottenRules": ["c1"], "staleFiles": [{"path": "x.go"}], "unaccounted": []}}))
        (self.tmp / "verify.json").write_text(json.dumps(result_page("not-landed", command="verify", state="findings",
                                                                     findings=[{"definition": "d"}])))
        (self.tmp / "verify-incomplete.json").write_text(json.dumps(result_page("not-landed", command="verify", state="incomplete")))
        (self.tmp / "ledger.json").write_text(json.dumps({
            "format": 1, "series": "acceptance", "tasks": {"makerspace": {}, "plant-1": {"planted": "defect"}}, "launches": launches,
            "chains": [{"id": "chain-k", "variant": "K", "task": "makerspace", "judgment": "chain-k"},
                       {"id": "chain-v", "variant": "2.0", "task": "makerspace", "judgment": "chain-k", "verifyFull": "verify.json"},
                       {"id": "chain-i", "variant": "K", "task": "makerspace", "judgment": "chain-k",
                        "verifyFull": "verify-incomplete.json"}],
            "contrasts": [{"variant": "2.0", "baseline": "K", "endpoint": "casesPassedShare", "minimumDifference": 0.05}]}))
        self.text, self.data = report.report(self.tmp / "ledger.json")

    def tearDown(self):
        self._tmp.cleanup()

    def row(self, title, first):
        part = self.text.split(f"## {title}\n", 1)[1].split("\n## ", 1)[0]
        return next(line for line in part.splitlines() if line.startswith(f"| {first} |"))

    def test_one_table_per_criterion(self):
        for title in ("§15.1 Correct", "§15.2 No drift", "§15.3 Nothing skipped", "§15.4 Stops right", "§15.5 At least as trustworthy as K"):
            self.assertIn(f"## {title}\n", self.text)

    def test_correct_counts_landed_changes_that_pass_every_case(self):
        self.assertEqual(self.row("§15.1 Correct", "K"), "| K | 3 | 3 | 1/3 | k-1, k-2 |")
        self.assertEqual(self.row("§15.1 Correct", "2.0"), "| 2.0 | 4 | 4 | 4/4 | none |")

    def test_a_planted_defect_counts_under_stops_right_only(self):
        self.assertEqual(self.row("§15.4 Stops right", "2.0"), "| 2.0 | 1/1 | 0 | n/a | 0 |")
        self.assertEqual(self.row("§15.4 Stops right", "K"), "| K | 0/1 | 0 | n/a | 0 |")
        self.assertTrue(self.row("Delivery", "K").startswith("| K | 3/3 |"))

    def test_nothing_skipped_reads_every_result(self):
        self.assertEqual(self.row("§15.3 Nothing skipped", "2.0"), "| 2.0 | 5 | 1 | 4 | 4 | 1 | 0 | 0 | 0 | v-plant: no Result |")
        self.assertIn("no Results: the variant keeps no checklist", self.row("§15.3 Nothing skipped", "K"))

    def test_no_drift_shows_the_final_drift_and_the_verify_slot(self):
        self.assertEqual(self.row("§15.2 No drift", "chain-k"),
                         "| chain-k | K | makerspace | yes | C9 | 1 | 1 | 0 | 1 | no verify --full (slot empty) |")
        self.assertTrue(self.row("§15.2 No drift", "chain-v").endswith("| 1 findings |"))
        # A verify that did not check every item is no clean bill: its state shows instead of "0 findings".
        self.assertTrue(self.row("§15.2 No drift", "chain-i").endswith("| incomplete: no verdict |"))

    def test_the_contrast_pairs_by_task_step_and_repetition(self):
        contrast = self.data["contrasts"][0]
        self.assertEqual((contrast["pairs"], contrast["unpaired"]), (3, 1))
        self.assertAlmostEqual(contrast["mean"], 1 / 19)
        self.assertEqual(contrast["reading"], "undecided")
        # Three pairs are too few: the interval reaches below minus the minimum meaningful difference.
        self.assertLess(contrast["low"], -0.05)
        # K's missing repetition 4 scored as not done: these judgments do not count by step, so its start fails all 19 cases.
        self.assertTrue(self.row("§15.5 At least as trustworthy as K", "2.0 vs K").endswith(
            "| undecided | undecided, [-0.467, 1.046] | no |"))
        self.assertFalse(contrast["atLeastAsGood"])

    def test_an_unknown_endpoint_is_refused(self):
        ledger = json.loads((self.tmp / "ledger.json").read_text())
        ledger["contrasts"][0]["endpoint"] = "speed"
        (self.tmp / "ledger.json").write_text(json.dumps(ledger))
        with self.assertRaisesRegex(ValueError, "unknown endpoint"):
            report.report(self.tmp / "ledger.json")


class BlockedContrastTest(unittest.TestCase):
    """A change's repetitions are not independent: with blockBy "change" the interval is read over each change's mean."""

    def facts(self):
        out = []
        for task, cases in (("a-fixed-C1", ((10, 9), (10, 10))), ("a-fixed-C2", ((8, 8), (7, 8)))):
            for rep, (m, k) in enumerate(cases, 1):
                for variant, passed in (("M", m), ("K", k)):
                    out.append({"variant": variant, "task": task, "step": task[-2:], "repetition": rep, "class": "result", "delivered": True,
                                "areas": {"correctness": {"passed": passed, "planned": 10, "failed": 10 - passed}},
                                "attention": {"noResult": 0, "casesFailed": 10 - passed, "confirmedCriticals": None}})
        return out

    def contrast(self, **extra):
        import acceptance
        _, data = acceptance.trustworthy(self.facts(), [{"variant": "M", "baseline": "K", "minimumDifference": 1, **extra}],
                                         lambda *a: "")
        return data[0]

    def test_pairs_or_changes(self):
        free = self.contrast(endpoint="casesFailed")
        self.assertEqual((free["pairs"], free["blocks"]), (4, None))
        self.assertAlmostEqual(free["mean"], 0.0)
        blocked = self.contrast(endpoint="casesFailed", blockBy="change")
        self.assertEqual((blocked["pairs"], blocked["blocks"]), (4, 2))
        # C1: M fails 0 and 0 against K's 1 and 0 (mean +0.5 for M); C2: M fails 2 and 3 against 2 and 2 (mean -0.5).
        self.assertAlmostEqual(blocked["mean"], 0.0)
        self.assertGreater(blocked["high"] - blocked["low"], free["high"] - free["low"])

    def test_cases_failed_counts_lower_as_better(self):
        facts = self.facts()
        self.assertEqual([f["attention"]["casesFailed"] for f in facts if f["variant"] == "M"], [0, 0, 2, 3])
        self.assertAlmostEqual(self.contrast(endpoint="casesPassedShare")["mean"], 0.0)


def correctness(failed, earlier=35, own=4, step="C2"):
    """A series judgment's correctness: the earlier steps' cases all passed, `failed` of the step's own failed."""
    return {"planned": earlier + own, "passed": earlier + own - failed, "failed": failed,
            "byStep": {"W0": {"planned": earlier, "passed": earlier, "failed": 0}, step: {"planned": own, "passed": own - failed, "failed": failed}}}


class NotDoneTest(unittest.TestCase):
    """A change not done scores as its start, whether a launch delivered nothing or a unit has no result at all."""

    def fact(self, variant, rep, task="g-fixed-C2", failed=0, delivered=True):
        f = {"variant": variant, "task": task, "step": task[-2:], "repetition": rep, "class": "result", "delivered": delivered}
        if delivered:
            f.update(areas={"correctness": correctness(failed, step=task[-2:])},
                     attention={"noResult": 0, "casesFailed": failed, "confirmedCriticals": 0})
        return f

    def contrast(self, facts, endpoint="casesPassedShare"):
        import acceptance
        text, data = acceptance.trustworthy(facts, [{"variant": "M", "baseline": "K", "endpoint": endpoint, "minimumDifference": 0.05}],
                                            lambda title, header, rows, note=None: rows)
        return data[0], text[0]

    def test_a_launch_that_delivered_nothing_scores_as_its_start(self):
        import acceptance
        facts = [self.fact("K", 1, failed=1), self.fact("M", 1, delivered=False)]
        starts = acceptance.start_scores(facts)
        self.assertEqual(starts, {("g-fixed-C2", "C2"): {"planned": 39, "passed": 35, "failed": 4}})
        facts[1]["startScore"] = starts[("g-fixed-C2", "C2")]
        # Not 0 and every case: the earlier steps' cases pass, the step's own four fail, and the step counts once more.
        self.assertEqual([acceptance.endpoint(facts[1], e) for e in ("casesPassedShare", "casesFailed", "defectsLeft")], [35 / 39, 4.0, 5.0])
        facts[1]["startScore"] = None
        self.assertIsNone(acceptance.endpoint(facts[1], "casesPassedShare"))
        # A judge that does not count by step has only the step's own cases: the start fails them all.
        single = {"variant": "K", "task": "library-m1", "step": "M1", "class": "result", "delivered": True,
                  "areas": {"correctness": {"planned": 19, "passed": 18, "failed": 1}}}
        self.assertEqual(acceptance.start_scores([single]), {("library-m1", "M1"): {"planned": 19, "passed": 0, "failed": 19}})

    def test_units_without_a_result_cannot_decide_the_verdict_by_dropping_out(self):
        paired = [self.fact("M", 1), self.fact("K", 1, failed=1), self.fact("M", 2), self.fact("K", 2), self.fact("M", 3), self.fact("K", 3)]
        # M has no result on C3 in any repetition: over the pairs alone it would be at least as good.
        dropped = [self.fact("K", rep, task="g-fixed-C3") for rep in (1, 2, 3)]
        contrast, row = self.contrast(paired + dropped)
        self.assertEqual((contrast["pairs"], contrast["unpaired"], contrast["reading"]), (3, 3, "a meaningful advantage is excluded"))
        self.assertFalse(contrast["notDone"]["atLeastAsGood"])
        self.assertEqual((contrast["notDone"]["pairs"], contrast["atLeastAsGood"], row[-1]), (6, None, "undecided (3 unpaired)"))
        # K's missing units scored as not done agree with the pairs: then the verdict stands.
        contrast, row = self.contrast(paired + [self.fact("M", rep, task="g-fixed-C3") for rep in (1, 2, 3)])
        self.assertEqual((contrast["atLeastAsGood"], row[-1]), (True, "yes"))
        contrast, row = self.contrast(paired)
        self.assertEqual((contrast["notDone"], row[-2], row[-1]), (None, "none unpaired", "yes"))

    def test_a_result_without_a_value_is_not_scored_as_not_done(self):
        # Defects left need a review: K's unreviewed result in repetition 4 has no value, but it is a result, not a unit
        # without one, so the second reading leaves it out instead of scoring it as K's change not done.
        import acceptance
        facts = [self.fact(v, rep) for rep in (1, 2, 3) for v in ("M", "K")]
        unreviewed, nothing = self.fact("K", 4), self.fact("M", 4, delivered=False)
        unreviewed["attention"]["confirmedCriticals"] = None
        nothing["startScore"] = acceptance.start_scores(facts)[("g-fixed-C2", "C2")]
        contrast, _ = self.contrast(facts + [unreviewed, nothing], endpoint="defectsLeft")
        self.assertEqual((contrast["pairs"], contrast["unpaired"], contrast["notDone"]["pairs"]), (3, 1, 3))


class OwnerRuleTest(unittest.TestCase):
    """The owner's rule (30 October 2026): at least as good only when both case contrasts and the landed rate hold."""

    OWN = {"W0": 34, "C1": 3, "C2": 2, "C3": 4, "C4": 4, "C5": 2, "C6": 2, "C7": 1}
    CONTRASTS = [{"variant": "M", "baseline": "K", "endpoint": "casesPassedShare", "minimumDifference": 0.05, "blockBy": "change"},
                 {"variant": "M", "baseline": "K", "endpoint": "casesFailed", "minimumDifference": 1, "blockBy": "change"},
                 {"variant": "M", "baseline": "K", "endpoint": "ownCasesPassedShare", "blockBy": "change"}]

    def fact(self, variant, step, done=True, delivered=True):
        steps = list(self.OWN)[:list(self.OWN).index(step) + 1]
        by = {s: {"planned": self.OWN[s], "passed": self.OWN[s], "failed": 0} for s in steps}
        if not done:
            by[step] = {"planned": self.OWN[step], "passed": 0, "failed": self.OWN[step]}
        failed, planned = by[step]["failed"], sum(s["planned"] for s in by.values())
        f = {"variant": variant, "task": f"gridshare-fixed-{step}", "step": step, "repetition": 1, "class": "result",
             "delivered": delivered, "landed": done}
        if delivered:
            f.update(areas={"correctness": {"planned": planned, "passed": planned - failed, "failed": failed, "byStep": by}},
                     attention={"noResult": 0, "casesFailed": failed, "confirmedCriticals": 0})
        return f

    def test_topics_example_is_no_longer_at_least_as_good(self):
        """K perfect; M does not land C2 (it delivers nothing), C5 and C6 (it delivers its start)."""
        import acceptance
        facts = [self.fact("K", s) for s in list(self.OWN)[1:]]
        facts += [self.fact("M", s, done=s not in ("C2", "C5", "C6"), delivered=s != "C2") for s in list(self.OWN)[1:]]
        starts = acceptance.start_scores(facts)
        for f in facts:
            if not f["delivered"]:
                f["startScore"] = starts[(f["task"], f["step"])]
        _, contrasts = acceptance.trustworthy(facts, self.CONTRASTS, lambda *a, **k: "")
        share, failed, own = contrasts
        # The share alone would say at least as good, with three of seven changes undone.
        self.assertAlmostEqual(share["mean"], -0.019, places=3)
        self.assertAlmostEqual(share["low"], -0.041, places=3)
        self.assertTrue(share["atLeastAsGood"])
        self.assertFalse(failed["atLeastAsGood"])
        self.assertAlmostEqual(own["mean"], -3 / 7)
        self.assertIsNone(own["atLeastAsGood"])
        rows = []
        _, verdicts = acceptance.at_least_as_good(facts, contrasts, lambda title, header, r, note=None: rows.extend(r))
        self.assertEqual(rows, [["M vs K", "yes", "no", "4/7 against 7/7", "4/7 against 7/7", "no", "no"]])
        self.assertFalse(verdicts[0]["atLeastAsGood"])

    def test_all_three_must_hold(self):
        import acceptance
        facts = [self.fact(v, s) for s in list(self.OWN)[1:] for v in ("M", "K")]
        _, contrasts = acceptance.trustworthy(facts, self.CONTRASTS, lambda *a, **k: "")
        _, verdicts = acceptance.at_least_as_good(facts, contrasts, lambda *a, **k: "x")
        # Identical results: both contrasts read zero difference with no spread, and the rates are equal.
        self.assertEqual((verdicts[0]["parts"], verdicts[0]["atLeastAsGood"]),
                         ({"casesPassedShare": True, "casesFailed": True, "landedRate": True}, True))
        # M's result of C7 is void after its relaunches: the landed rate holds over the pairs, not with C7 as not landed.
        facts = [f for f in facts if (f["variant"], f["step"]) != ("M", "C7")]
        _, contrasts = acceptance.trustworthy(facts, self.CONTRASTS, lambda *a, **k: "")
        _, verdicts = acceptance.at_least_as_good(facts, contrasts, lambda *a, **k: "x")
        self.assertIsNone(verdicts[0]["parts"]["landedRate"])
        self.assertEqual(verdicts[0]["landed"], [{"M": (6, 6), "K": (6, 6)}, {"M": (6, 7), "K": (7, 7)}])


class ChainCountTest(unittest.TestCase):
    """A chain's steps share one task name: a step that delivered nothing is scored by its own step's counts."""

    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = pathlib.Path(self._tmp.name)
        commit = json.loads((RUN / "suite.json").read_text())["result"]["commit"]
        launches = []
        for name, step, earlier, own, failed in (("c1", "C1", 27, 3, 0), ("c2", "C2", 30, 4, 1), ("c1-none", "C1", 27, 3, None)):
            shutil.copytree(RUN, self.tmp / name)
            if failed is None:
                record = json.loads((self.tmp / name / "suite.json").read_text())
                record.update(delivered=False, result=None, resultError="no result.bundle")
                (self.tmp / name / "suite.json").write_text(json.dumps(record))
            else:
                (self.tmp / (name + "-j")).mkdir()
                value = judgment(commit)
                value["areas"]["correctness"] = {**correctness(failed, earlier, own, step), "failedCases": [f"c{i}" for i in range(failed)]}
                (self.tmp / (name + "-j") / "judgment.json").write_text(json.dumps(value))
            launches.append({"id": name, "variant": "K", "task": "lib-chain", "run": name, "step": step, "repetition": 1,
                             **({"judgment": name + "-j"} if failed is not None else {})})
        (self.tmp / "ledger.json").write_text(json.dumps({"format": 1, "series": "chains", "tasks": {"lib-chain": {}}, "launches": launches}))
        self.text, _ = report.report(self.tmp / "ledger.json")

    def tearDown(self):
        self._tmp.cleanup()

    def test_the_count_is_keyed_by_task_and_step(self):
        cells = lambda title: next(l for l in self.text.split(f"## {title}\n", 1)[1].splitlines() if l.startswith("| K |")).split(" | ")
        # The step without a result fails C1's own three cases, not the chain's largest count (34), and passes C1's earlier 27.
        self.assertEqual(cells("Human attention (primary)")[3], "1.33 (0-3, n=3)")
        self.assertEqual(cells("Correctness")[1:3], ["30 (27-33, n=3)", "31.33 (30-34, n=3)"])


if __name__ == "__main__":
    unittest.main()
