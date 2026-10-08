"""Offline producer/consumer regression using sealed inputs and temporary grants."""
from __future__ import annotations

import copy
from dataclasses import replace
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import native_fixture_budget as budget
import government_native_profile as profiles

PACKAGE = Path(__file__).parents[1]
SEALED = PACKAGE / "evidence/government-scope-native-20261008-r5"
spec = importlib.util.spec_from_file_location(
    "released_binding_driver", PACKAGE / "run-native-integration-r5.py")
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)


def sealed_bytes(path, expected_sha):
    raw = path.read_bytes()
    if hashlib.sha256(raw).hexdigest() != expected_sha:
        raise AssertionError(f"sealed test input changed: {path}")
    return raw


class GovernmentReleasedBindingTests(unittest.TestCase):
    def setUp(self):
        # These tests must never launch an experimental process or open a ledger.
        self.addCleanup(patch.stopall)
        patch("subprocess.Popen", side_effect=AssertionError("offline test forbids processes")).start()
        patch("sqlite3.connect", side_effect=AssertionError("offline test forbids ledgers")).start()
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.original = self.root / "original-grant.json"
        self.original.write_bytes(sealed_bytes(
            PACKAGE / "evidence/native-integration/run-1/external-snapshots/released-native-grant.json",
            "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"))
        self.profile_name = getattr(self, "profile_name", "r5")
        self.profile = profiles.profile(self.profile_name)
        self.marker = self.profile.marker
        self.snapshot = self.root / self.profile.snapshot_path.name
        self.snapshot.write_bytes(sealed_bytes(self.profile.snapshot_path, self.profile.snapshot_sha))
        envelope = json.loads(sealed_bytes(self.profile.envelope_path, self.profile.envelope_sha))
        envelope["sourceCoordinationPath"] = str(self.snapshot.resolve())
        self.successor = self.root / self.profile.envelope_path.name
        self.successor.write_bytes(driver.dispatch.encoded(envelope) + b"\n")
        relocated_profile = replace(
            self.profile, external_root=Path(driver.binding(self.successor)["path"]).parent,
            evidence_directory=Path(driver.binding(self.successor)["path"]).parent / "evidence",
            envelope_sha=driver.binding(self.successor)["sha256"],
            snapshot_sha=driver.binding(self.snapshot)["sha256"])
        patch.object(profiles, self.profile_name.upper(), relocated_profile).start()
        patch.object(budget, "R3_BASE_GRANT_PATH", str(self.original)).start()
        patch.object(budget, "R3_BASE_GRANT_SHA256", driver.binding(self.original)["sha256"]).start()
        self.produced = driver.fixture_grant_bindings(
            driver.binding(self.original), driver.binding(self.successor),
            driver.binding(self.snapshot), self.profile_name)
        # Reuse the sealed failed Request's context. Only the exact production
        # helper supplies the tested grant fields and released-input construction.
        self.request = json.loads(sealed_bytes(
            SEALED / "raw-external/government/released/request.json",
            "20c2d21a49a4c0ac86f9e91e7681d607c9dd7c293ab3abfde96dcddf2c265d1f"))
        if self.profile_name == "r6":
            self.request.pop("nativeFixtureR5Grant", None)
            self.request["dispatchId"] = self.profile.dispatch_id
            self.request["task"]["id"] = self.profile.task_id
            old_profile_paths = {str(item.get("path", "")) for item in self.request["releasedInputs"]
                                 if isinstance(item, dict) and
                                 (item.get("sourceKey") or "r5" in Path(item.get("path", "")).name)}
            self.request["releasedInputs"] = [item for item in self.request["releasedInputs"]
                                               if item.get("path") not in old_profile_paths]
        self.request.update(copy.deepcopy(self.produced))

    def consume(self):
        return getattr(budget, f"validate_{self.profile_name}_grant_binding")(
            self.request, self.original, driver.binding(self.original)["sha256"])

    def test_production_bindings_pass_the_real_consumer(self):
        admitted = self.consume()
        self.assertEqual(admitted["grantKey"], self.profile.key)
        self.assertEqual(admitted["sourceCoordinationPath"], str(self.snapshot.resolve()))
        self.assertEqual([set(item) for item in self.produced["releasedInputs"]],
                         [{"path", "sha256"}] * 3)
        self.assertEqual(set(self.produced["nativeFixtureGrant"]),
                         {"path", "sha256", "sourceKey"})
        self.assertEqual(set(self.produced[self.marker]),
                         {"path", "sha256", "sourceKey"})

    def test_old_three_field_released_entry_is_rejected(self):
        self.request["releasedInputs"][1] = dict(self.request[self.marker])
        with self.assertRaisesRegex(ValueError, f"{self.marker} must be an exact released input"):
            self.consume()

    def test_missing_or_different_successor_release_is_rejected(self):
        baseline = copy.deepcopy(self.request)
        self.request["releasedInputs"].pop(1)
        with self.assertRaisesRegex(ValueError, "exact released input"):
            self.consume()
        self.request = baseline
        other = self.root / "other-successor.json"
        other.write_bytes(self.successor.read_bytes())
        self.request["releasedInputs"][1] = driver.binding(other)
        with self.assertRaisesRegex(ValueError, "exact released input"):
            self.consume()

    def test_wrong_release_or_metadata_digest_is_rejected(self):
        baseline = copy.deepcopy(self.request)
        self.request["releasedInputs"][1]["sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "exact released input"):
            self.consume()
        self.request = baseline
        self.request[self.marker]["sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "does not bind the exact released source file"):
            self.consume()

    def test_wrong_source_key_is_rejected_for_both_grants(self):
        baseline = copy.deepcopy(self.request)
        for field in ("nativeFixtureGrant", self.marker):
            with self.subTest(field=field):
                self.request = copy.deepcopy(baseline)
                self.request[field]["sourceKey"] = "closed-or-other-source"
                with self.assertRaises(ValueError):
                    self.consume()

    def test_snapshot_requires_its_separate_exact_file_release(self):
        baseline = copy.deepcopy(self.request)
        for mutation in ("missing", "digest", "metadata"):
            with self.subTest(mutation=mutation):
                self.request = copy.deepcopy(baseline)
                if mutation == "missing":
                    self.request["releasedInputs"].pop(2)
                elif mutation == "digest":
                    self.request["releasedInputs"][2]["sha256"] = "0" * 64
                else:
                    self.request["releasedInputs"][2]["sourceKey"] = self.profile.key
                with self.assertRaisesRegex(ValueError, "canonical source snapshot.*exact released"):
                    self.consume()

    def test_original_release_keeps_the_existing_two_field_contract(self):
        self.request["releasedInputs"][0] = dict(self.request["nativeFixtureGrant"])
        with self.assertRaisesRegex(ValueError, "nativeFixtureGrant must be an exact released input"):
            self.consume()


class GovernmentReleasedBindingR6Tests(GovernmentReleasedBindingTests):
    """Exercise the same production producer against the real closed R6 consumer."""
    profile_name = "r6"


if __name__ == "__main__":
    unittest.main()
