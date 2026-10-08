"""Actual Government fixture wire bytes; no native, wrapper or delegate process.

Assertions follow 04e225d agentexec Response/validateResponse field rules.
They inspect serialized bytes, not a substitute implementation of Host admission.
"""
import base64
import hashlib
import json
from pathlib import Path
import unittest

from fixtures.government_positive import deterministic_delegate as fixture
import government_roles

ROOT = Path(__file__).resolve().parents[1]
R3_STDOUT = (ROOT / "evidence/native-integration/run-3/external-snapshots/r3/government/role-evidence" /
             "579807ab283d8dfd5484b28076523cabd6e666050cdec8ec91cf58732e0dc9f7/stdout.log")
REQUIRED_ARRAYS = {"candidateFiles", "evidenceRefs", "verifierObservations", "uncertainty"}
REQUIRED_FIELDS = REQUIRED_ARRAYS | {"apiVersion", "runId", "nonce", "role", "inputDigest", "outcome"}


def invocation(phase, *, good_candidate=True, subjects=True, identity=True):
    role = "executor" if phase == "execute" else "verifier"
    files = [("inventory/reservation.go", fixture.SOURCE),
             ("inventory/reservation_test.go", fixture.TEST)]
    if not good_candidate:
        files = [(path, "package inventory\n") for path, _ in files]
    return {
        "apiVersion": fixture.API, "runId": "offline-serialization-" + phase,
        "nonce": "fixed-offline-nonce-" + phase, "inputDigest": "sha256:" + "a" * 64,
        "request": {"role": role, "projectionId": "government/" + phase + "/root",
                    "scopeIds": ["requirement/inventory/safe-release"] if subjects else [],
                    "policyIds": [],
                    "artifacts": [{"path": path, "content": base64.b64encode(content.encode()).decode()}
                                  for path, content in files] if phase != "execute" else [],
                    "context": {"candidate": {"id": "sha256:" + "b" * 64},
                                "evidence": {"id": "sha256:" + "c" * 64, "round": 1}} if identity else {}}}


class GovernmentResponseSerializationTests(unittest.TestCase):
    def assert_wire_shape(self, request, phase):
        raw = fixture.response_bytes(request, phase)
        self.assertIsInstance(raw, bytes)
        self.assertTrue(raw.endswith(b"\n"))
        value = json.loads(raw)
        self.assertEqual(set(value), REQUIRED_FIELDS)
        self.assertNotIn(b'"candidateJson"', raw)
        for name in REQUIRED_ARRAYS:
            self.assertIsInstance(value[name], list, name)
        for name in ("apiVersion", "runId", "nonce", "inputDigest"):
            self.assertEqual(value[name], request[name])
        self.assertEqual(value["role"], request["request"]["role"])
        return value

    def test_executor_serialized_bytes_omit_inference_and_keep_required_empty_arrays(self):
        value = self.assert_wire_shape(invocation("execute"), "execute")
        self.assertEqual(value["outcome"], "proposed")
        self.assertEqual(value["verifierObservations"], [])
        self.assertEqual(value["uncertainty"], [])
        self.assertEqual({item["path"]: item["content"] for item in value["candidateFiles"]},
                         {"inventory/reservation.go": fixture.SOURCE,
                          "inventory/reservation_test.go": fixture.TEST})

    def test_verifier_review_serialized_success_failure_and_incomplete_shapes(self):
        for good, subjects, expected in ((True, True, "passed"), (False, True, "failed"),
                                         (True, False, "incomplete")):
            with self.subTest(good_candidate=good, subjects=subjects):
                value = self.assert_wire_shape(invocation("review", good_candidate=good, subjects=subjects), "review")
                self.assertEqual(value["outcome"], expected)
                self.assertEqual(value["candidateFiles"], [])
                if expected != "incomplete":
                    self.assertTrue(value["verifierObservations"])
                    self.assertTrue(all(item["outcome"] == expected and item["subject"] and item["detail"]
                                        for item in value["verifierObservations"]))
                else:
                    self.assertEqual(value["verifierObservations"], [])
                    self.assertTrue(value["uncertainty"])

    def test_verifier_vote_serialized_assent_objection_and_missing_identity_shapes(self):
        for good, identity, expected in ((True, True, "passed"), (False, True, "failed"),
                                         (True, False, "incomplete")):
            with self.subTest(good_candidate=good, identity=identity):
                value = self.assert_wire_shape(invocation("vote", good_candidate=good, identity=identity), "vote")
                self.assertEqual(value["candidateFiles"], [])
                self.assertEqual(value["outcome"], expected)
                if identity:
                    observation = value["verifierObservations"][0]
                    detail = json.loads(observation["detail"])
                    self.assertEqual(detail["outcome"], "assent" if good else "objection")
                    self.assertEqual(detail["evidenceId"], "sha256:" + "c" * 64)
                    self.assertEqual(detail["materialCandidateId"], "sha256:" + "b" * 64)
                    self.assertEqual(detail["round"], 1)
                else:
                    self.assertEqual(value["verifierObservations"], [])
                    self.assertTrue(value["uncertainty"])

    def test_retained_r3_bytes_show_null_regression_without_rescoring_history(self):
        raw = R3_STDOUT.read_bytes()
        self.assertEqual(hashlib.sha256(raw).hexdigest(),
                         "7d434a9c103ced86de2886a74bdc7ce589c7956b0f4f5bb356fc896f6b9c6092")
        historical = json.loads(raw)
        self.assertIn("candidateJson", historical)
        self.assertIsNone(historical["candidateJson"])
        current = self.assert_wire_shape(invocation("execute"), "execute")
        self.assertEqual(current["candidateFiles"], historical["candidateFiles"])
        # A new source-derived offline response is not a replay or new acceptance.
        self.assertNotEqual(current["runId"], historical["runId"])

    def test_fixed_delegate_pin_matches_only_the_corrected_source(self):
        actual = hashlib.sha256(Path(fixture.__file__).read_bytes()).hexdigest()
        self.assertEqual(actual, government_roles._GOVERNMENT_FIXTURE_DELEGATE_SHA256)

    def test_inference_support_is_not_added_to_this_fixture(self):
        request = invocation("execute")
        request["request"]["role"] = "infer"
        with self.assertRaisesRegex(ValueError, "unsupported agentexec role"):
            fixture.response_bytes(request, "execute")
