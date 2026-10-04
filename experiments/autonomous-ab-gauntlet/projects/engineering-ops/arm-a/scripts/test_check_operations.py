import unittest

from check_operations import EXPECTED_RUNS, inventory

class OperationsSeedTests(unittest.TestCase):
    def test_managed_roots_and_exact_exclusion(self):
        roots, owners, exclusions = inventory()
        self.assertIn("internal/config", roots)
        self.assertIn("internal/config/parse.go", owners)
        self.assertNotIn("testdata/vendor-snapshots", roots)
        self.assertEqual(
            exclusions["testdata/vendor-snapshots"],
            "immutable parser-test input; no production policy ownership",
        )

    def test_hook_and_pipeline_cover_each_declared_fast_check(self):
        self.assertEqual(
            set(EXPECTED_RUNS),
            {"Operations policy", "Operations checker tests", "Root module tests", "Nested module tests"},
        )

if __name__ == "__main__":
    unittest.main()
