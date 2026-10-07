"""Deterministic tests for synthetic context package/checker mechanics only."""
from __future__ import annotations

import json
import hashlib
import io
from pathlib import Path
import tempfile
import unittest
from contextlib import redirect_stdout

from context_smoke import canonical_json, check_access_observation, check_observation, main, prepare, probe_current_identity, sha256


class ContextSmokeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="markitect-context-smoke-")
        self.root = Path(self.temp.name) / "attempt-1"
        self.manifest = prepare(self.root)

    def tearDown(self):
        self.temp.cleanup()

    def observation(self, *, observed_text=None, mode="synthetic-observation-fixture"):
        files = self.manifest["injectedFiles"]
        text = observed_text or (
            self.manifest["syntheticMarkers"]["releasedContext"] + "\n" +
            self.manifest["syntheticMarkers"]["actorOwn"] + "\n"
        )
        return {
            "mode": mode,
            "attemptId": self.manifest["attemptId"],
            "manifestSha256": self.manifest["manifestSha256"],
            "actorInvoked": False,
            "modelInvoked": False,
            "effectivePromptSha256": self.manifest["promptFile"]["sha256"],
            "effectiveFiles": [{"path": f["path"], "sha256": f["sha256"]} for f in files],
            "observedText": text,
        }

    def access_observation(self, *, granted=True):
        access = self.manifest["accessProbe"]
        process = access["childProcess"]
        return {
            "mode": "synthetic-access-observation-fixture",
            "attemptId": self.manifest["attemptId"],
            "manifestSha256": self.manifest["manifestSha256"],
            "actorInvoked": False,
            "modelInvoked": False,
            "before": access["before"].copy(),
            "after": access["expectedAfter"].copy(),
            "controlRead": {
                "path": self.manifest["control"]["path"],
                "granted": granted,
                "contentSha256": access["before"]["controlSha256"] if granted else None,
            },
            "scratchWrite": {
                "path": access["scratch"]["path"],
                "contentSha256": access["scratch"]["sha256"],
            },
            "childProcess": {
                "started": True,
                "argv": process["argv"],
                "returnCode": process["expectedReturnCode"],
                "stdoutSha256": process["expectedStdoutSha256"],
                "stderrSha256": process["expectedStderrSha256"],
                "shell": False,
            },
        }

    def test_prepare_records_exact_synthetic_prompt_and_file_hashes(self):
        self.assertFalse(self.manifest["actorInvoked"])
        self.assertFalse(self.manifest["modelInvoked"])
        self.assertFalse(self.manifest["inferencePerformed"])
        self.assertTrue(self.manifest["syntheticOnly"])
        prompt_path = Path(self.manifest["promptFile"]["path"])
        prompt_bytes = prompt_path.read_bytes()
        self.assertEqual(prompt_bytes.decode("utf-8"), self.manifest["exactInjectedPrompt"])
        self.assertEqual(hashlib.sha256(prompt_bytes).hexdigest(), self.manifest["promptFile"]["sha256"])
        for item in self.manifest["injectedFiles"]:
            raw = Path(item["path"]).read_bytes()
            self.assertIsInstance(raw, bytes)
            self.assertEqual(raw.decode("utf-8"), item["contentUtf8"])
            self.assertEqual(hashlib.sha256(raw).hexdigest(), item["sha256"])
        self.assertFalse(self.manifest["control"]["suppliedToActor"])
        access = self.manifest["accessProbe"]
        self.assertFalse(access["before"]["scratchExists"])
        self.assertEqual(access["before"]["controlSha256"], access["expectedAfter"]["controlSha256"])
        card = Path(self.manifest["accessCard"]["file"]["path"])
        self.assertEqual(card.read_text(encoding="utf-8"), self.manifest["accessCard"]["exactText"])
        self.assertEqual(hashlib.sha256(card.read_bytes()).hexdigest(), self.manifest["accessCard"]["file"]["sha256"])
        helper = access["childProcess"]["helperFile"]
        self.assertEqual(Path(helper["path"]).read_text(encoding="utf-8"), helper["contentUtf8"])
        self.assertEqual(hashlib.sha256(Path(helper["path"]).read_bytes()).hexdigest(), helper["sha256"])
        self.assertNotIn(self.manifest["syntheticMarkers"]["unreleasedSynthetic"], self.manifest["exactInjectedPrompt"])
        self.assertNotIn(self.manifest["syntheticMarkers"]["actorWrite"], self.manifest["exactInjectedPrompt"])
        self.assertNotIn(self.manifest["syntheticMarkers"]["childProcess"], self.manifest["exactInjectedPrompt"])
        self.assertNotIn(self.manifest["accessCard"]["file"]["path"], [f["path"] for f in self.manifest["injectedFiles"]])
        self.assertNotIn(helper["path"], [f["path"] for f in self.manifest["injectedFiles"]])
        self.assertIn(self.manifest["control"]["path"], self.manifest["accessCard"]["exactText"])
        self.assertIn(access["scratch"]["path"], self.manifest["accessCard"]["exactText"])
        self.assertIn(json.dumps(access["childProcess"]["argv"]), self.manifest["accessCard"]["exactText"])
        self.assertIn(repr(access["scratch"]["contentUtf8"]), self.manifest["accessCard"]["exactText"])

    def test_access_checker_verifies_synthetic_before_after_and_allowed_operations(self):
        result = check_access_observation(self.root, self.access_observation())
        self.assertEqual(result["status"], "verified-observation")
        self.assertEqual(result["capabilities"], {
            "controlRead": "granted", "scratchWrite": True, "childProcess": True,
        })
        self.assertFalse(result["isolationEstablished"])

    def test_check_access_cli_accepts_synthetic_fixture_without_launching_actor(self):
        observation_path = self.root / "observation-fixture.json"
        observation_path.write_text(json.dumps(self.access_observation()), encoding="utf-8")
        output = io.StringIO()
        with redirect_stdout(output):
            code = main(["check-access", "--attempt-root", str(self.root), "--observation", str(observation_path)])
        self.assertEqual(code, 0)
        self.assertEqual(json.loads(output.getvalue())["status"], "verified-observation")

    def test_access_checker_records_denied_control_read_as_a_valid_measurement(self):
        result = check_access_observation(self.root, self.access_observation(granted=False))
        self.assertEqual(result["status"], "verified-observation")
        self.assertEqual(result["capabilities"]["controlRead"], "denied")

    def test_access_checker_rejects_changed_state_and_unapproved_child_process(self):
        observation = self.access_observation()
        observation["after"]["scratchSha256"] = "0" * 64
        observation["childProcess"]["argv"] = ["python", "unexpected.py"]
        result = check_access_observation(self.root, observation)
        self.assertEqual(result["status"], "invalid-observation")
        self.assertIn("after.scratchSha256 mismatch", result["errors"])
        self.assertIn("child process argv mismatch", result["errors"])

    def test_checker_accepts_only_exact_injected_inputs_and_expected_markers(self):
        result = check_observation(self.root, self.observation())
        self.assertEqual(result["status"], "passed")
        self.assertTrue(result["mechanicalOnly"])
        self.assertFalse(result["isolationEstablished"])

    def test_checker_rejects_synthetic_control_leak(self):
        text = self.manifest["syntheticMarkers"]["releasedContext"] + " " + self.manifest["syntheticMarkers"]["actorOwn"]
        text += " " + self.manifest["syntheticMarkers"]["unreleasedSynthetic"]
        result = check_observation(self.root, self.observation(observed_text=text))
        self.assertEqual(result["status"], "failed")
        self.assertIn("unreleased synthetic control marker leaked into observed text", result["errors"])

    def test_checker_rejects_extra_or_changed_input_and_wrong_attempt(self):
        observation = self.observation()
        observation["effectiveFiles"] = observation["effectiveFiles"][:1]
        observation["attemptId"] = "another-attempt"
        result = check_observation(self.root, observation)
        self.assertEqual(result["status"], "failed")
        self.assertIn("attemptId mismatch", result["errors"])
        self.assertIn("effective input files do not match the explicit injected-file list", result["errors"])

    def test_synthetic_observation_cannot_claim_actor_or_model_started(self):
        observation = self.observation()
        observation["actorInvoked"] = True
        result = check_observation(self.root, observation)
        self.assertEqual(result["status"], "failed")
        self.assertIn("synthetic observation fixture must say actorInvoked=false and modelInvoked=false", result["errors"])

    def test_child_process_can_read_artificial_sibling_under_current_identity(self):
        result = probe_current_identity(self.root)
        self.assertTrue(result["controlReadGranted"])
        self.assertTrue(result["generatedChildProcess"]["matchesPreparedExpectation"])
        self.assertFalse(result["generatedChildProcess"]["shell"])
        self.assertFalse(result["actorInvoked"])
        self.assertFalse(result["modelInvoked"])
        self.assertEqual(result["mode"], "local-same-user-access-probe")
        persisted = json.loads((self.root / "same-user-access-observation.json").read_text(encoding="utf-8"))
        self.assertTrue(persisted["controlReadGranted"])

    def test_current_identity_probe_rejects_hash_consistent_tampered_helper(self):
        manifest = self.manifest.copy()
        manifest["accessProbe"] = json.loads(json.dumps(self.manifest["accessProbe"]))
        child = manifest["accessProbe"]["childProcess"]
        child["argv"] = ["python", "unapproved.py"]
        helper = child["helperFile"]
        tampered_content = "import sys\nsys.stdout.write('unexpected')\n"
        helper_path = Path(helper["path"])
        helper_path.write_text(tampered_content, encoding="utf-8")
        helper["contentUtf8"] = tampered_content
        helper["sha256"] = hashlib.sha256(tampered_content.encode("utf-8")).hexdigest()
        helper["bytes"] = len(tampered_content.encode("utf-8"))
        manifest.pop("manifestSha256")
        manifest["manifestSha256"] = sha256(canonical_json(manifest))
        (self.root / "manifest.json").write_bytes(canonical_json(manifest))
        with self.assertRaisesRegex(ValueError, "fixed local interpreter/helper"):
            probe_current_identity(self.root)

    def test_prepare_refuses_checkout_destination_and_existing_destination(self):
        checkout_target = Path(__file__).resolve().parents[3] / "experiments" / "government-comparison" / "context-smoke-test-target"
        with self.assertRaises(ValueError):
            prepare(checkout_target)
        with self.assertRaises(ValueError):
            prepare("relative-context-smoke-target")
        with self.assertRaises(FileExistsError):
            prepare(self.root)


if __name__ == "__main__":
    unittest.main()
