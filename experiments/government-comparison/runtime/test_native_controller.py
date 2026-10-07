"""Synthetic tests for the importable controller API; never starts Markitect."""
from __future__ import annotations

import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch

import classic
import classic_integration
import native_controller
import native_fixture_budget
from native_fixture_budget import (CORRECTION_BASIS, CORRECTION_KEY, CORRECTION_POINTER,
                                   KEY, PROCESS_SECONDS_RESERVED, SOURCE_THREAD, FixtureBudget)


class _Ledger:
    def __init__(self):
        self.rows = []

    def running_bound(self, _task):
        return 100.0, None

    def controller_elapsed(self, _dispatch):
        return 0.0

    def record_controller_process(self, dispatch_id, action, started, receipt):
        row = {"dispatchId": dispatch_id, "action": action, "sequence": len(self.rows) + 1,
               "start": started, "processReceipt": receipt}
        self.rows.append(row)
        return row


class _Budget(FixtureBudget):
    def __init__(self, grant_path, binary):
        self.grant_path = Path(grant_path).resolve()
        self.grant_sha = native_controller.digest(self.grant_path.read_bytes())
        self.binaries = {"classic": str(Path(binary).resolve())}
        self.reservations = []
        self.receipts = []

    def reserve(self, product, label, argv):
        self.reservations.append((product, label, argv))

    def finish(self, product, label, receipt):
        if receipt.get("argv") != self.reservations[-1][2]:
            raise AssertionError("budget receipt must echo exact reserved argv")
        self.receipts.append((product, label, receipt))


class NativeClassicControllerTests(unittest.TestCase):
    def test_classic_bootstrap_failure_terminalizes_claim_without_product_process(self):
        with tempfile.TemporaryDirectory(prefix="classic-bootstrap-failure-") as temporary:
            root = Path(temporary)
            request_path = root / "request.json"
            raw = b"synthetic exact request"
            request_path.write_bytes(raw)
            auth_path = root / "authorization.json"
            import government_roles
            auth_path.write_text(json.dumps({"fixtureAuthorization": government_roles.NATIVE_FIXTURE_AUTH}),
                                 encoding="utf-8")
            evidence = root / "already-created"
            evidence.mkdir()
            request = {"arm": "classic", "operation": "run_task", "mode": "mechanical",
                       "dispatchId": "synthetic-bootstrap-failure", "task": {"id": "task"},
                       "purpose": "synthetic pre-effect reservation test", "evidenceDirectory": str(evidence),
                       "releasedInputs": [], "product": {"classic": {
                           "controllerActions": list(native_controller.CLASSIC_ACTIONS),
                           "roleAuthorization": {"path": str(auth_path.resolve()),
                                                 "sha256": native_controller.digest(auth_path.read_bytes())}}}}

            class ClaimLedger:
                def __init__(self):
                    self.reserved = False
                    self.claimed = False
                    self.final = None

                def dispatch_record(self, _dispatch):
                    return None

                def reserve_controller_dispatch(self, *_args):
                    self.reserved = True

                def claim_dispatch(self, _dispatch, _phase):
                    self.claimed = True
                    return True

                def finish_controller_dispatch(self, _dispatch, result, receipt):
                    self.final = (result, receipt)
                    return result

            ledger = ClaimLedger()
            authority = SimpleNamespace(validate=lambda _raw: (request, {}), grant={"maxActorSessions": 4},
                                        ledger=lambda *_args, **_kwargs: ledger)
            bound = {"executable": root / "unused.exe", "repository": root,
                     "sourceCommit": "a" * 40}
            with patch("classic_integration.bind_request", return_value=bound), \
                 patch.object(native_controller, "validate_native_fixture_grant"), \
                 patch("government_roles.preflight_authorization"), \
                 patch("dispatch.execution_sha", return_value="b" * 64):
                with self.assertRaises(FileExistsError):
                    native_controller.begin_classic_controller(request_path, authority, request_raw=raw)
            self.assertTrue(ledger.reserved)
            self.assertTrue(ledger.claimed)
            self.assertEqual(ledger.final[0]["status"], "incomplete")
            self.assertFalse(ledger.final[1]["productProcessStarted"])
            self.assertEqual(ledger.final[1]["controllerProcesses"], [])

    def test_native_fixture_source_grant_is_released_and_product_pin_bound(self):
        with tempfile.TemporaryDirectory(prefix="native-fixture-grant-") as temporary:
            root = Path(temporary)
            packet = classic.inspect_packet(classic_integration.PACKET)
            binary = Path(packet["binary"]["path"]).resolve(strict=True)
            grant = {"key": native_controller.NATIVE_FIXTURE_GRANT_KEY,
                     "realActorStartsAuthorized": 0, "providerCallsAuthorized": 0,
                     "studyCellsAuthorized": 0,
                     "perProduct": {"positiveCases": 1, "nativeCliOrControllerStartsMaximum": 8,
                                    "deterministicRoleStartsMaximum": 12, "maxParallel": 2,
                                    "totalProcessSecondsMaximum": 1200},
                     "products": [{"name": "Government", "sourceSha": "a" * 40,
                                   "binarySha256": classic.EXPECTED_BINARY_SHA256},
                                  {"name": "Classic", "sourceSha": classic.EXPECTED_HELD_SOURCE,
                                   "binarySha256": classic.EXPECTED_BINARY_SHA256}]}
            snapshot = {"threads": [{"name": "Scientist", "evidence": {
                "nativeIntegrationPreparationGrant": grant}}]}
            snapshot_raw = json.dumps(snapshot, sort_keys=True).encode()
            snapshot_path = root / "source-snapshot.json"
            snapshot_path.write_bytes(snapshot_raw)
            grant_doc = {"sourceThreadId": native_controller.NATIVE_FIXTURE_SOURCE_THREAD,
                         "sourceJsonPointer": "threads[name=Scientist].evidence.nativeIntegrationPreparationGrant",
                         "sourceCoordinationPath": str(snapshot_path),
                         "sourceCoordinationSha256": native_controller.digest(snapshot_raw),
                         "grant": grant}
            grant_raw = json.dumps(grant_doc, sort_keys=True).encode()
            grant_path = root / "grant.json"
            grant_path.write_bytes(grant_raw)
            entry = {"path": str(grant_path.resolve()), "sha256": native_controller.digest(grant_raw)}
            request = {"arm": "classic", "mode": "mechanical", "nativeFixtureGrant": {**entry,
                       "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY}, "releasedInputs": [entry],
                       "product": {"classic": {"executable": {"sha256": classic.EXPECTED_BINARY_SHA256}}}}
            bound = {"executable": binary, "sourceCommit": classic.EXPECTED_SOURCE}
            result = native_controller.validate_native_fixture_grant(
                request, {str(grant_path): grant_raw}, bound)
            self.assertEqual(result["sourceKey"], native_controller.NATIVE_FIXTURE_GRANT_KEY)
            self.assertEqual(result["binarySha256"], classic.EXPECTED_BINARY_SHA256)
            self.assertEqual(result["maxRoleParallel"], 2)
            correction_grant = {
                "key": CORRECTION_KEY,
                "basis": {"negativeHandoffSha": CORRECTION_BASIS, "previousAllocation": KEY,
                          "previousNativeStarts": {"government": 2, "classic": 1},
                          "previousReservedSessionSeconds": {"government": 300, "classic": 150},
                          "previousNativeWrapperAttempts": {"government": 1, "classic": 0},
                          "previousDelegateAttempts": 0},
                "limits": {"government": {"maxAdditionalNativeStarts": 2,
                                           "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                                           "maxAdditionalReservedControllerAndRoleSessionSeconds": 300,
                                           "cases": "One queue attempt; one directly related resume/replay only after the positive case succeeds. No repeated inspect."},
                           "classic": {"maxAdditionalNativeStarts": 5,
                                       "maxAdditionalRoleInvocationsIncludingFailedWrapperStarts": 6,
                                       "maxAdditionalReservedControllerAndRoleSessionSeconds": 750,
                                       "cases": "One Execute, guarded Apply after bound external review, fresh Verify, Audit, and stale Apply replay; later steps only after their prerequisites pass."},
                           "maxParallelRoles": 2, "productsRunSequentially": True,
                           "cumulativeNativeStartCeiling": {"government": 4, "classic": 6, "total": 10},
                           "cumulativeReservedSessionSecondsCeiling": {"government": 600, "classic": 900, "total": 1500},
                           "previousOverallNativeStartCeiling": 16,
                           "previousOverallReservedSessionSecondsCeiling": 2400,
                           "nativeProcessDeadlineSeconds": 38,
                           "controllerWindowSeconds": {"government": 38, "classic": 180},
                           "reservedSessionSecondsPerNativeStart": PROCESS_SECONDS_RESERVED},
                "realActorCalls": 0, "providerCalls": 0, "metadataAppServerTrees": 0,
                "studyCells": 0, "newPurchases": False, "productMutations": False}
            correction_snapshot = {"threads": [{"name": "Scientist", "evidence": {
                "correctedNativeIntegrationGrant": correction_grant}}]}
            correction_snapshot_raw = json.dumps(correction_snapshot, sort_keys=True).encode()
            correction_snapshot_path = root / "corrected-source-snapshot.json"
            correction_snapshot_path.write_bytes(correction_snapshot_raw)
            correction_doc = {"sourceThreadId": SOURCE_THREAD, "sourceJsonPointer": CORRECTION_POINTER,
                              "sourceCoordinationPath": str(correction_snapshot_path),
                              "sourceCoordinationSha256": native_controller.digest(correction_snapshot_raw),
                              "grant": correction_grant}
            correction_raw = json.dumps(correction_doc, sort_keys=True).encode()
            correction_path = root / "correction.json"
            correction_path.write_bytes(correction_raw)
            with patch.object(native_fixture_budget, "CORRECTION_SOURCE_PATH", str(correction_snapshot_path)), \
                 patch.object(native_fixture_budget, "CORRECTION_SOURCE_SHA256",
                              native_controller.digest(correction_snapshot_raw)):
                correction_binding = {"path": str(correction_path.resolve()),
                                      "sha256": native_controller.digest(correction_raw),
                                      "sourceKey": CORRECTION_KEY}
                request["nativeFixtureCorrection"] = correction_binding
                request["releasedInputs"].append({"path": correction_binding["path"],
                                                   "sha256": correction_binding["sha256"]})
                request["releasedInputs"].append({"path": str(correction_snapshot_path.resolve()),
                                                   "sha256": native_controller.digest(correction_snapshot_raw)})
                captured = {str(grant_path): grant_raw, str(correction_path): correction_raw}
                corrected = native_controller.validate_native_fixture_grant(request, captured, bound)
            self.assertEqual(corrected["maxRoleStarts"], 6)
            self.assertEqual(corrected["maxRoleParallel"], 2)
            self.assertEqual(corrected["maxRoleProcessSeconds"], 1200)
            request["nativeFixtureGrant"]["sha256"] = "0" * 64
            with self.assertRaisesRegex(ValueError, "digest mismatch"):
                native_controller.validate_native_fixture_grant(request, {str(grant_path): grant_raw}, bound)

    def test_five_step_flow_is_ordered_review_gated_and_process_accounted(self):
        with tempfile.TemporaryDirectory(prefix="classic-controller-api-") as temporary:
            root = Path(temporary)
            repo = root / "actor"
            repo.mkdir()
            runtime = root / "runtime.json"
            record_store = root / "state" / "records"
            runtime_raw = json.dumps({"recordStore": str(record_store)}).encode()
            runtime.write_bytes(runtime_raw)
            reports = root / "evidence" / "classic-native"
            reports.mkdir(parents=True)
            packet = classic.inspect_packet(classic_integration.PACKET)
            binary = Path(packet["binary"]["path"]).resolve(strict=True)
            grant_path = root / "source-grant.json"
            grant_path.write_text("synthetic source grant bound in Request", encoding="utf-8")
            request = {"dispatchId": "synthetic-classic-flow", "task": {"id": "task"},
                       "wallSeconds": 100, "baseCommit": "a" * 40,
                       "nativeFixtureGrant": {"path": str(grant_path.resolve()),
                                              "sha256": native_controller.digest(grant_path.read_bytes())},
                       "product": {"classic": {"packet": {"path": str(classic_integration.PACKET)},
                                                "projectConfig": {"path": "examples/canonical-projection/canonical.yaml"},
                                                "runtime": {"path": str(runtime.resolve()),
                                                            "sha256": native_controller.digest(runtime_raw)}}}}
            authority = SimpleNamespace(grant={"maxSessionWallSeconds": 100,
                                                 "expiresAt": time.time() + 300})
            context = SimpleNamespace(request=request, bootstrap_path=root / "bootstrap.json",
                                      bootstrap_sha256="a" * 64, authority=authority,
                                      authorization_raw=json.dumps({"expiresAt": time.time() + 200}).encode(),
                                      captured_inputs={str(runtime.resolve()): runtime_raw})
            ledger = _Ledger()
            bound = {"executable": binary, "repository": repo}
            session = native_controller.ClassicControllerSession(
                context, ledger, bound, root / "evidence", reports,
                native_controller.CLASSIC_ACTIONS)
            budget = _Budget(grant_path, binary)
            statuses = [
                {"status": "planned", "digest": "sha256:" + "1" * 64},
                {"status": "materialized-unverified"},
                {"status": "passed"},
                {"status": "complete", "findings": [], "nextSteps": []},
                {"status": "refused", "materialized": False},
            ]
            planned_calls = []
            record_store_checks = []
            original_record_store_check = classic_integration.assert_record_store_absent

            def checked_record_store(runtime_config):
                record_store_checks.append(runtime_config)
                return original_record_store_check(runtime_config)

            def plan(action, **kwargs):
                planned_calls.append((action, kwargs))
                return {"action": action, "argv": [str(binary), "canonical", "--action", action],
                        "cwd": str(repo), "expectedReport": "json", "timeoutSeconds": 40,
                        "stdoutPath": str((reports / f"{action}.json").resolve()),
                        "stderrPath": str((reports / f"{action}.stderr.txt").resolve())}

            def fake_bounded(argv, cwd, evidence, wall_seconds, **_kwargs):
                evidence = Path(evidence)
                evidence.mkdir(parents=True, exist_ok=False)
                status = statuses.pop(0)
                (evidence / "stdout.log").write_bytes(json.dumps(status, sort_keys=True).encode())
                (evidence / "stderr.log").write_bytes(b"")
                receipts = [{"path": str((evidence / name).resolve()), "sha256": "0" * 64,
                             "kind": name} for name in ("stdout.log", "stderr.log")]
                process_value = {"argv": argv, "cwd": cwd, "returnCode": 0, "wallSeconds": 0.01,
                                 "stopReason": None, "processTreeControl": "synthetic-test-double",
                                 "automaticRetries": 0, "receipts": receipts}
                (evidence / "process.json").write_text(json.dumps(process_value), encoding="utf-8")
                return process_value

            with patch.object(classic_integration, "plan_native_action", side_effect=plan), \
                 patch.object(classic_integration, "assert_record_store_absent", side_effect=checked_record_store), \
                 patch("process.bounded", side_effect=fake_bounded):
                execute = native_controller.run_classic_step(session, "execute", fixture_budget=budget)
                execute_raw = Path(execute["stdoutPath"]).read_bytes()
                review = {"status": "approved", "executeReportSha256": classic_integration.sha256(execute_raw),
                          "executeDigest": "sha256:" + "1" * 64,
                          "reviewerReference": "synthetic operator review"}
                native_controller.run_classic_step(session, "apply", fixture_budget=budget,
                                                   external_review=review)
                native_controller.run_classic_step(session, "verify", fixture_budget=budget)
                native_controller.run_classic_step(session, "audit", fixture_budget=budget)
                native_controller.run_classic_step(session, "apply-replay", fixture_budget=budget,
                                                   external_review=review)

            self.assertEqual([row[0] for row in planned_calls], list(native_controller.CLASSIC_ACTIONS))
            self.assertEqual([row["action"] for row in ledger.rows], list(native_controller.CLASSIC_ACTIONS))
            self.assertEqual(len(budget.reservations), 5)
            self.assertEqual(len(budget.receipts), 5)
            self.assertTrue(all(item[0] == "classic" for item in budget.reservations))
            self.assertEqual(session.review, review)
            self.assertTrue(all(Path(row["processPath"]).is_file() for row in session.captures))
            self.assertEqual(statuses, [])
            self.assertEqual(len(record_store_checks), 1)
            self.assertFalse(record_store.exists())

    def test_skipped_action_and_missing_budget_fail_before_process(self):
        with tempfile.TemporaryDirectory(prefix="classic-controller-order-") as temporary:
            root = Path(temporary)
            packet = classic.inspect_packet(classic_integration.PACKET)
            session = native_controller.ClassicControllerSession(
                SimpleNamespace(request={"dispatchId": "d", "task": {"id": "t"}}), _Ledger(),
                {"executable": Path(packet["binary"]["path"]).resolve(), "repository": root},
                root, root, native_controller.CLASSIC_ACTIONS)
            with self.assertRaisesRegex(ValueError, "action order"):
                native_controller.run_classic_step(session, "apply", fixture_budget=None,
                                                   external_review={})
            self.assertEqual(session.ledger.rows, [])


if __name__ == "__main__":
    unittest.main()
