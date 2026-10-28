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
        (self.tmp / "verify.json").write_text(json.dumps(result_page("not-landed", command="verify", findings=[{"definition": "d"}])))
        (self.tmp / "ledger.json").write_text(json.dumps({
            "format": 1, "series": "acceptance", "tasks": {"makerspace": {}, "plant-1": {"planted": "defect"}}, "launches": launches,
            "chains": [{"id": "chain-k", "variant": "K", "task": "makerspace", "judgment": "chain-k"},
                       {"id": "chain-v", "variant": "2.0", "task": "makerspace", "judgment": "chain-k", "verifyFull": "verify.json"}],
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
        self.assertEqual(self.row("§15.3 Nothing skipped", "2.0"), "| 2.0 | 5 | 4 | 4 | 1 | 0 | 0 | 0 | none |")
        self.assertIn("no Results: the variant keeps no checklist", self.row("§15.3 Nothing skipped", "K"))

    def test_no_drift_shows_the_final_drift_and_the_verify_slot(self):
        self.assertEqual(self.row("§15.2 No drift", "chain-k"),
                         "| chain-k | K | makerspace | yes | C9 | 1 | 1 | 0 | 1 | no verify --full (slot empty) |")
        self.assertTrue(self.row("§15.2 No drift", "chain-v").endswith("| 1 findings |"))

    def test_the_contrast_pairs_by_task_step_and_repetition(self):
        contrast = self.data["contrasts"][0]
        self.assertEqual((contrast["pairs"], contrast["unpaired"]), (3, 1))
        self.assertAlmostEqual(contrast["mean"], 1 / 19)
        self.assertEqual(contrast["reading"], "undecided")
        # Three pairs are too few: the interval reaches below minus the minimum meaningful difference.
        self.assertLess(contrast["low"], -0.05)
        self.assertTrue(self.row("§15.5 At least as trustworthy as K", "2.0 vs K").endswith("| undecided | no |"))

    def test_an_unknown_endpoint_is_refused(self):
        ledger = json.loads((self.tmp / "ledger.json").read_text())
        ledger["contrasts"][0]["endpoint"] = "speed"
        (self.tmp / "ledger.json").write_text(json.dumps(ledger))
        with self.assertRaisesRegex(ValueError, "unknown endpoint"):
            report.report(self.tmp / "ledger.json")


if __name__ == "__main__":
    unittest.main()
