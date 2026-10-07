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
from native_fixture_budget import FixtureBudget


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
                     "products": [{"name": "Classic", "sourceSha": classic.EXPECTED_HELD_SOURCE,
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
            request["nativeFixtureGrant"]["sha256"] = "0" * 64
            with self.assertRaisesRegex(ValueError, "digest mismatch"):
                native_controller.validate_native_fixture_grant(request, {str(grant_path): grant_raw}, bound)

    def test_five_step_flow_is_ordered_review_gated_and_process_accounted(self):
        with tempfile.TemporaryDirectory(prefix="classic-controller-api-") as temporary:
            root = Path(temporary)
            repo = root / "actor"
            repo.mkdir()
            runtime = root / "runtime.json"
            runtime.write_text("{}", encoding="utf-8")
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
                                                "runtime": {"path": str(runtime)}}}}
            authority = SimpleNamespace(grant={"maxSessionWallSeconds": 100,
                                                 "expiresAt": time.time() + 300})
            context = SimpleNamespace(request=request, bootstrap_path=root / "bootstrap.json",
                                      bootstrap_sha256="a" * 64, authority=authority,
                                      authorization_raw=json.dumps({"expiresAt": time.time() + 200}).encode())
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
