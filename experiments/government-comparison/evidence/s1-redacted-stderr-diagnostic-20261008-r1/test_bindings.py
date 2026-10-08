"""Narrow synthetic tests for prospective grant and capture binding validators."""

from __future__ import annotations

import copy
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import client


class BindingValidationTests(unittest.TestCase):
    def setUp(self):
        self._popen_guard = patch.object(
            client.subprocess, "Popen", side_effect=AssertionError("process-start-forbidden")
        )
        self._popen_guard.start()
        self.addCleanup(self._popen_guard.stop)
        self.packet = Path(__file__).resolve().parent
        self.authorization = json.loads((self.packet / "authorization-grant.json").read_text(encoding="utf-8"))
        self.grant = self.authorization["grant"]
        self.profile = json.loads((self.packet / "profile.json").read_text(encoding="utf-8"))

    def slot(self, **overrides):
        result = {
            "owner": "Scientist",
            "key": client.KEY,
            "status": client.SLOT_STATUS,
            "grantIssuedUtc": client.ISSUED,
            "assignedUtc": "2026-10-08T17:00:00Z",
        }
        result.update(overrides)
        return result

    def coordination(self, grant=None, slot=None, **unrelated):
        return {
            "threads": [{"name": "Scientist", "evidence": {
                "redactedStderrDiagnosticGrant": copy.deepcopy(self.grant if grant is None else grant)
            }}],
            "fullSuiteSlot": copy.deepcopy(self.slot() if slot is None else slot),
            **unrelated,
        }

    def authority(self):
        return {
            "sourceCoordinationPath": client.COORDINATION,
            "sourceJsonPointer": client.AUTHORITY_POINTER,
            "grant": copy.deepcopy(self.grant),
        }

    def live_with_document(self, authority, document):
        actual_read_bytes = Path.read_bytes
        target = Path(client.COORDINATION).resolve()

        def read_bytes(path):
            if path.resolve() == target:
                return json.dumps(document, separators=(",", ":")).encode("utf-8")
            return actual_read_bytes(path)

        with patch.object(Path, "read_bytes", read_bytes):
            return client.live_authority(authority)

    def test_profile_identity_and_exact_grant_owner_are_required(self):
        client.validate_profile(copy.deepcopy(self.profile), copy.deepcopy(self.grant))
        invalid = []
        wrong_profile_key = copy.deepcopy(self.profile)
        wrong_profile_key["key"] = "foreign-key"
        invalid.append((wrong_profile_key, copy.deepcopy(self.grant)))
        wrong_grant_key = copy.deepcopy(self.grant)
        wrong_grant_key["key"] = "foreign-key"
        invalid.append((copy.deepcopy(self.profile), wrong_grant_key))
        wrong_owner = copy.deepcopy(self.grant)
        wrong_owner["ownerThreadId"] = "foreign-thread"
        invalid.append((copy.deepcopy(self.profile), wrong_owner))
        wrong_issue = copy.deepcopy(self.grant)
        wrong_issue["issuedUtc"] = "2026-10-08T16:47:19Z"
        invalid.append((copy.deepcopy(self.profile), wrong_issue))
        wrong_binary = copy.deepcopy(self.grant)
        wrong_binary["executableSha256"] = "0" * 64
        invalid.append((copy.deepcopy(self.profile), wrong_binary))
        for profile, grant in invalid:
            with self.subTest(changed="identity-or-owner"):
                with self.assertRaises(ValueError):
                    client.validate_profile(profile, grant)

    def test_complete_live_grant_equality_distinguishes_boolean_from_number(self):
        client.validate_live_authority(copy.deepcopy(self.grant), self.slot(), self.grant)

        boolean_instead_of_one = copy.deepcopy(self.grant)
        boolean_instead_of_one["maxNewAppServerTrees"] = True
        with self.assertRaises(ValueError):
            client.validate_live_authority(boolean_instead_of_one, self.slot(), self.grant)

    def test_exact_active_slot_and_grant_are_required_but_later_assignment_is_valid(self):
        authority = self.authority()
        later = self.slot(assignedUtc="2026-10-08T17:30:00Z")
        live_grant, assigned = self.live_with_document(authority, self.coordination(slot=later))
        self.assertEqual(live_grant, self.grant)
        self.assertEqual(assigned, later["assignedUtc"])

        invalid_slots = (
            self.slot(owner="Architect"),
            self.slot(status="Queued"),
            self.slot(key="foreign-slot"),
            self.slot(grantIssuedUtc="2026-10-08T16:47:19Z"),
            self.slot(assignedUtc="2026-10-08T16:47:17Z"),
        )
        for slot in invalid_slots:
            with self.subTest(slot_owner=slot["owner"], slot_status=slot["status"]):
                with self.assertRaises(ValueError):
                    self.live_with_document(authority, self.coordination(slot=slot))

        invalid_grants = []
        for name, value in (("key", "foreign-grant"), ("status", "Revoked"),
                            ("issuedUtc", "2026-10-08T16:47:19Z")):
            changed = copy.deepcopy(self.grant)
            changed[name] = value
            invalid_grants.append(changed)
        for grant in invalid_grants:
            changed_authority = self.authority()
            changed_authority["grant"] = copy.deepcopy(grant)
            with self.assertRaises(ValueError):
                self.live_with_document(changed_authority, self.coordination(grant=grant))

        for field, value in (("sourceCoordinationPath", "C:/foreign/coordination.json"),
                             ("sourceJsonPointer", "threads[name=Architect].evidence.grant")):
            bad_authority = self.authority()
            bad_authority[field] = value
            with self.assertRaises(ValueError):
                self.live_with_document(bad_authority, self.coordination())

    def test_live_authority_ignores_unrelated_whole_state_cursor_and_time(self):
        authority = self.authority()
        first = self.coordination(revision=1, updatedAt="2026-10-08T17:01:00Z", cursor="cursor-a")
        second = self.coordination(revision=987, updatedAt="2026-10-08T18:00:00Z", cursor="cursor-b")
        documents = iter((first, second))
        actual_read_bytes = Path.read_bytes
        target = Path(client.COORDINATION).resolve()

        def read_bytes(path):
            if path.resolve() == target:
                return json.dumps(next(documents), separators=(",", ":")).encode("utf-8")
            return actual_read_bytes(path)

        with patch.object(Path, "read_bytes", read_bytes):
            first_result = client.live_authority(authority)
            second_result = client.live_authority(authority)
        self.assertEqual(first_result, second_result)
        self.assertEqual(first_result[1], "2026-10-08T17:00:00Z")

    def test_synthetic_request_freeze_identity_slot_and_time_mismatches_are_rejected(self):
        assigned = "2026-10-08T17:00:00Z"
        source_commit = "a" * 40
        request = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": assigned,
            "sourceCommit": source_commit,
        }
        freeze = {
            "key": client.KEY,
            "grantIssuedUtc": client.ISSUED,
            "slotAssignedUtc": assigned,
            "sourceCommit": source_commit,
            "frozenAtUtc": "2026-10-08T17:01:00Z",
        }
        client.validate_binding_identity(request, freeze, assigned)

        bad_cases = []
        changed = copy.deepcopy(request)
        changed["key"] = "foreign-key"
        bad_cases.append((changed, copy.deepcopy(freeze), assigned))
        changed = copy.deepcopy(freeze)
        changed["grantIssuedUtc"] = "2026-10-08T16:47:19Z"
        bad_cases.append((copy.deepcopy(request), changed, assigned))
        changed = copy.deepcopy(request)
        changed["slotAssignedUtc"] = "2026-10-08T17:00:01Z"
        bad_cases.append((changed, copy.deepcopy(freeze), assigned))
        changed = copy.deepcopy(freeze)
        changed["sourceCommit"] = "b" * 40
        bad_cases.append((copy.deepcopy(request), changed, assigned))
        changed = copy.deepcopy(freeze)
        changed["frozenAtUtc"] = "2026-10-08T16:59:59Z"
        bad_cases.append((copy.deepcopy(request), changed, assigned))
        for candidate_request, candidate_freeze, candidate_assigned in bad_cases:
            with self.subTest(binding="request-freeze-identity-or-order"):
                with self.assertRaises(ValueError):
                    client.validate_binding_identity(candidate_request, candidate_freeze, candidate_assigned)

    def test_capture_caps_private_path_and_collector_pin_are_bound(self):
        profile = {"limits": copy.deepcopy(client.LIMITS)}
        grant = copy.deepcopy(self.grant)
        client.validate_capture_bindings(profile, grant)

        for name, value in (("maxStderrCaptureBytes", 1023),
                            ("maxStderrDiagnosticLines", 3),
                            ("maxRedactedOutputBytes", 2047),
                            ("privateExcerptRelativePath", "other/file.json")):
            changed = copy.deepcopy(grant)
            changed[name] = value
            with self.subTest(binding=name):
                with self.assertRaises(ValueError):
                    client.validate_capture_bindings(profile, changed)

        changed_profile = {"limits": {**client.LIMITS, "maxStderrCaptureBytes": 2048}}
        with self.assertRaises(ValueError):
            client.validate_capture_bindings(changed_profile, grant)

        contract_path = Path(client.ROOT) / "diagnostic-contract.json"
        actual_read_bytes = Path.read_bytes
        original_contract = json.loads(contract_path.read_text(encoding="utf-8"))
        changed_contract = copy.deepcopy(original_contract)
        changed_contract["collectorSha256"] = "0" * 64
        target = contract_path.resolve()

        def read_bytes(path):
            if path.resolve() == target:
                return json.dumps(changed_contract, separators=(",", ":")).encode("utf-8")
            return actual_read_bytes(path)

        with patch.object(Path, "read_bytes", read_bytes):
            with self.assertRaises(ValueError):
                client.validate_capture_bindings(profile, grant)


if __name__ == "__main__":
    unittest.main()
