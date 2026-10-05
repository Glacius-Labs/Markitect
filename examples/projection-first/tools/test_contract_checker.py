import unittest

from check_contract import ROOT, expected_rows, load_fixture


class ContractCheckerTests(unittest.TestCase):
    def test_fixture_exercises_filter_tie_break_and_complete_rows(self):
        view, process, fixture = load_fixture()
        rows = expected_rows(view, fixture["cases"][1]["items"])
        self.assertEqual([row["id"] for row in rows], ["early", "tie-a", "tie-z", "late"])
        self.assertEqual(rows[-1]["note"], "keep me")
        self.assertEqual(rows[1]["extra"], {"priority": 1})
        self.assertNotIn("closed", [row["id"] for row in rows])
        self.assertEqual(process["spec"]["preCommitCommand"], "python tools/check_contract.py")

    def test_empty_case_has_no_rows(self):
        view, _process, fixture = load_fixture()
        self.assertEqual(expected_rows(view, fixture["cases"][0]["items"]), [])

    def test_checker_reads_canonical_resource_and_fixture(self):
        view, process, fixture = load_fixture(ROOT)
        self.assertEqual(view["spec"]["testCases"], ["fixtures/listings.json"])
        self.assertIn("python tools/check_contract.py", process["spec"]["verificationCommands"])
        self.assertEqual(len(fixture["cases"]), 2)

    def test_duplicate_selected_ids_are_rejected(self):
        view, _process, _fixture = load_fixture()
        rows = [
            {"id": "same", "status": "Open", "openedAt": "2026-01-01T00:00:00Z"},
            {"id": "same", "status": "Open", "openedAt": "2026-01-02T00:00:00Z"},
        ]
        with self.assertRaisesRegex(ValueError, "unique stable ids"):
            expected_rows(view, rows)


if __name__ == "__main__":
    unittest.main()
