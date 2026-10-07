"""Controls for experiment validity, never scored as actor capability."""
import ast
import json
from pathlib import Path
import tempfile
import sys
import subprocess
import unittest

from harness import probe, validate_result
from prepare import ROOT, git, prepare
from release import release, staged_checks


class BindingAndReleaseTests(unittest.TestCase):
    def test_reproducible_empty_and_brownfield_starts(self):
        with tempfile.TemporaryDirectory() as temp:
            a = prepare(Path(temp) / "a")
            b = prepare(Path(temp) / "b")
            self.assertEqual(a["starts"], b["starts"])
            self.assertEqual(a["starts"]["greenfield"]["files"], {})
            self.assertEqual(len(a["cells"]), 6)
            for cell in a["cells"]:
                self.assertEqual(git(cell["repository"], "status", "--porcelain"), "")
                self.assertEqual(subprocess.check_output(["git", "-C", cell["repository"], "status", "--porcelain"], text=True).strip(), "")

    def test_failed_adapter_preserved_and_live_refused(self):
        with tempfile.TemporaryDirectory() as temp:
            temp = Path(temp)
            manifest = prepare(temp / "fixture")
            released = release(1, temp / "inputs")
            cell = manifest["cells"][0]
            request = {"schemaVersion": 1, "operation": "probe", "mode": "fixture", "trialId": "negative-control",
                       **cell, "actorRepository": cell["repository"], "releasedInputs": released["inputs"],
                       "limits": {"wallSeconds": 1, "maxActorCalls": 0}}
            event = probe(request, [sys.executable, "-c", "raise SystemExit(9)"], temp / "failed")
            self.assertEqual(event["status"], "failed")
            self.assertEqual(event["exitCode"], 9)
            self.assertTrue((temp / "failed" / "events.jsonl").is_file())
            with self.assertRaises(ValueError):
                probe({**request, "mode": "live"}, [sys.executable], temp / "live")
            first = Path(released["inputs"][0]["path"])
            first.write_text("changed input", encoding="utf-8")
            with self.assertRaises(ValueError):
                probe(request, [sys.executable], temp / "stale")

    def test_stale_or_cross_trial_result_rejected(self):
        request = {"schemaVersion": 1, "trialId": "a", "operation": "probe", "mode": "fixture"}
        request_digest = "e" * 64
        result = {**request, "requestSha256": request_digest, "status": "readiness_gap", "candidateCommit": None, "capabilities": [], "gaps": [], "receipts": [], "usage": None}
        validate_result(request, result, request_digest)
        for field, wrong in (("trialId", "b"), ("schemaVersion", 2), ("operation", "run_task"), ("mode", "live"), ("requestSha256", "f" * 64)):
            with self.subTest(field=field), self.assertRaises(ValueError):
                validate_result(request, {**result, field: wrong}, request_digest)
        with self.assertRaises(ValueError):
            validate_result(request, {**result, "candidateCommit": "a" * 40}, request_digest)

    def test_future_checks_absent_until_release(self):
        for cutoff in range(1, 7):
            source = staged_checks(cutoff)
            ast.parse(source)
            self.assertEqual("boundary-six" in source, cutoff >= 5)
            self.assertEqual("parallel-" in source, cutoff >= 2)
            self.assertEqual("cancel releases once" in source, cutoff >= 4)

    def test_handoff_outside_repo_current_cards_only(self):
        with tempfile.TemporaryDirectory() as temp:
            destination = Path(temp) / "task1"
            manifest = release(1, destination)
            self.assertEqual(len(list(destination.glob("0*.md"))), 1)
            self.assertFalse((destination / ".git").exists())
            self.assertNotIn("becomes 6", (destination / "brief.md").read_text(encoding="utf-8"))
            self.assertTrue(all(len(entry["sha256"]) == 64 for entry in manifest["inputs"]))
            self.assertEqual(json.loads((destination / "release.json").read_text())["throughTask"], 1)


if __name__ == "__main__":
    unittest.main()
