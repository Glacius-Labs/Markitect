"""Pure preparation tests; no Markitect executable or provider is started."""
from __future__ import annotations

import base64
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import government
import government_integration as integration
from fixtures.government_positive import deterministic_delegate


def invocation(phase, role, *, context=None, artifacts=None):
    return {
        "apiVersion": deterministic_delegate.API,
        "runId": f"fixture-{phase}",
        "nonce": f"nonce-{phase}",
        "inputDigest": "sha256:" + "a" * 64,
        "request": {
            "role": role,
            "sourceRevision": "b" * 40,
            "modelDigest": "sha256:" + "c" * 64,
            "modulePin": "sha256:" + "d" * 64,
            "projectionId": f"government/{phase}/root" if phase != "vote" else "government/vote/inventory-correctness",
            "scopeIds": ["requirement/inventory/safe-release"],
            "policyIds": [],
            "context": context or {},
            "artifacts": artifacts or [],
        },
    }


def artifacts(source: str, test: str):
    return [
        {"path": "inventory/reservation.go", "mode": "0644", "digest": "sha256:" + "1" * 64,
         "content": base64.b64encode(source.encode()).decode()},
        {"path": "inventory/reservation_test.go", "mode": "0644", "digest": "sha256:" + "2" * 64,
         "content": base64.b64encode(test.encode()).decode()},
    ]


class GovernmentIntegrationPreparationTests(unittest.TestCase):
    def test_disposable_repo_requires_real_inspection_binding_before_runtime(self):
        with tempfile.TemporaryDirectory(prefix="government-positive-fixture-") as temporary:
            repo = Path(temporary) / "repo"
            created = integration.create_disposable_repository(repo)
            self.assertIsNone(created["constitutionDigest"])
            self.assertEqual(created["managedRef"], integration.MANAGED_REF)
            self.assertIn(integration.PENDING_CONSTITUTION, (repo / "order.yaml").read_text())

            bound = integration.bind_order_to_inspection(repo, "sha256:" + "1" * 64)
            self.assertEqual(bound["constitutionDigest"], "sha256:" + "1" * 64)
            self.assertNotEqual(bound["baseCommit"], created["baseCommit"])
            self.assertEqual((repo / "order.yaml").read_text().count("sha256:" + "1" * 64), 1)
            self.assertEqual(integration._run_git(repo, "rev-parse", "--verify", f"{integration.MANAGED_REF}^{{commit}}"),
                             bound["baseCommit"])
            self.assertEqual(integration._run_git(repo, "status", "--porcelain"), "")
            with self.assertRaisesRegex(ValueError, "already bound"):
                integration.bind_order_to_inspection(repo, "sha256:" + "2" * 64)

    def test_three_native_slots_have_actual_prefixed_runtime_file_digests(self):
        with tempfile.TemporaryDirectory(prefix="government-positive-runtime-") as temporary:
            root = Path(temporary)
            repo = root / "repo"
            marker = integration.create_disposable_repository(repo)
            marker = integration.bind_order_to_inspection(repo, "sha256:" + "3" * 64)
            auth_path = root / "role-authorization.json"
            runtime_path = root / "runtime.json"
            for name in ("queue", "runs", "temporary"):
                (root / name).mkdir()
            auth = integration.build_role_authorization(
                trial_id="test-trial", dispatch_id="test-dispatch",
                request_path=root / "request.json", ledger_path=root / "dispatch.sqlite",
                runtime_path=runtime_path, role_evidence_directory=root / "role-evidence",
                expires_at=2_000_000_000, python_executable=sys.executable,
                delegate_path=integration.FIXTURE_ROOT / "deterministic_delegate.py",
                status="approved", fixture_authorization=integration.FIXTURE_AUTHORIZATION)
            auth_raw = integration.canonical_json(auth)
            auth_path.write_bytes(auth_raw)
            runtime = integration.build_runtime(
                repository=repo, base_commit=marker["baseCommit"],
                queue_state_directory=root / "queue", run_state_directory=root / "runs",
                temporary_directory=root / "temporary", runtime_path=runtime_path,
                authorization_path=auth_path, authorization_raw=auth_raw,
                python_executable=sys.executable,
                delegate_path=integration.FIXTURE_ROOT / "deterministic_delegate.py",
                expected_constitution_digest="sha256:" + "3" * 64)

            configured = government.configured_roles(runtime)
            self.assertEqual([item["slotId"] for item in configured],
                             ["positive-root-executor", "positive-root-verifier", "positive-ressort-correctness"])
            self.assertEqual([item["phase"] for item in configured], ["execute", "review", "vote"])
            self.assertEqual(len(runtime["ressorts"]), 1)
            self.assertEqual(auth["fixtureAuthorization"], {
                "sourceGrantKey": "native-s1-integration-fixtures-20261008",
                "classification": "nativeFixture", "providerUse": "noProvider"})
            self.assertEqual(runtime["timeoutSeconds"], 40)
            self.assertEqual(runtime["executor"]["timeoutSeconds"], 30)
            self.assertEqual(runtime["checks"][0]["timeoutSeconds"], 30)
            for slot in (runtime["executor"], runtime["verifier"], runtime["ressorts"][0]["runner"]):
                files = {item["path"]: item for item in slot["runtimeFiles"]}
                self.assertIn(str(Path(integration.government_roles.__file__).resolve()), files)
                self.assertIn(str((Path(integration.government_roles.__file__).parent / "native_controller.py").resolve()), files)
                self.assertIn(str((Path(integration.government_roles.__file__).parent / "native_fixture_budget.py").resolve()), files)
                self.assertIn(str(auth_path.resolve()), files)
                self.assertIn(str(Path(sys.executable).resolve()), files)
                self.assertIn(str((integration.FIXTURE_ROOT / "deterministic_delegate.py").resolve()), files)
                self.assertTrue(all(len(item["digest"]) == 71 and item["digest"].startswith("sha256:")
                                    and all(ch in "0123456789abcdef" for ch in item["digest"][7:])
                                    for item in files.values()))
                self.assertEqual(files[str(Path(sys.executable).resolve())], integration.runtime_file(sys.executable))
                self.assertEqual(files[str((integration.FIXTURE_ROOT / "deterministic_delegate.py").resolve())],
                                 integration.runtime_file(integration.FIXTURE_ROOT / "deterministic_delegate.py"))
            self.assertEqual(integration.expected_queue_role_calls()["queueTotal"], 3)

    def test_pending_role_authorization_cannot_build_native_runtime(self):
        with tempfile.TemporaryDirectory(prefix="government-positive-pending-") as temporary:
            root = Path(temporary)
            repo = root / "repo"
            marker = integration.create_disposable_repository(repo)
            marker = integration.bind_order_to_inspection(repo, "sha256:" + "4" * 64)
            auth_path = root / "role-authorization.json"
            runtime_path = root / "runtime.json"
            auth = integration.build_role_authorization(
                trial_id="test-trial", dispatch_id="test-dispatch",
                request_path=root / "request.json", ledger_path=root / "dispatch.sqlite",
                runtime_path=runtime_path, role_evidence_directory=root / "role-evidence",
                expires_at=2_000_000_000, python_executable=sys.executable,
                delegate_path=integration.FIXTURE_ROOT / "deterministic_delegate.py")
            auth_raw = integration.canonical_json(auth)
            auth_path.write_bytes(auth_raw)
            with self.assertRaisesRegex(ValueError, "actual approved operator"):
                integration.build_runtime(
                    repository=repo, base_commit=marker["baseCommit"],
                    queue_state_directory=root / "queue", run_state_directory=root / "runs",
                    temporary_directory=root / "temporary", runtime_path=runtime_path,
                    authorization_path=auth_path, authorization_raw=auth_raw,
                    python_executable=sys.executable,
                    delegate_path=integration.FIXTURE_ROOT / "deterministic_delegate.py",
                    expected_constitution_digest="sha256:" + "4" * 64)

    def test_approved_fixture_authorization_requires_exact_parent_source_grant(self):
        with tempfile.TemporaryDirectory(prefix="government-positive-grant-") as temporary:
            root = Path(temporary)
            common = dict(
                trial_id="test-trial", dispatch_id="test-dispatch",
                request_path=root / "request.json", ledger_path=root / "dispatch.sqlite",
                runtime_path=root / "runtime.json", role_evidence_directory=root / "role-evidence",
                expires_at=2_000_000_000, python_executable=sys.executable,
                delegate_path=integration.FIXTURE_ROOT / "deterministic_delegate.py", status="approved")
            with self.assertRaisesRegex(ValueError, "exact parent source grant"):
                integration.build_role_authorization(**common)
            with self.assertRaisesRegex(ValueError, "exact parent source grant"):
                integration.build_role_authorization(**common, fixture_authorization={
                    **integration.FIXTURE_AUTHORIZATION, "providerUse": "provider"})
            approved = integration.build_role_authorization(
                **common, fixture_authorization=integration.FIXTURE_AUTHORIZATION)
            self.assertEqual(approved["fixtureAuthorization"], integration.FIXTURE_AUTHORIZATION)

    def test_deterministic_delegate_uses_native_invocation_and_candidate_bindings(self):
        executor = invocation("execute", "executor")
        proposed = deterministic_delegate.response(executor, "execute")
        self.assertEqual(proposed["outcome"], "proposed")
        self.assertEqual({item["path"] for item in proposed["candidateFiles"]},
                         {"inventory/reservation.go", "inventory/reservation_test.go"})
        final_source = next(item["content"] for item in proposed["candidateFiles"]
                            if item["path"] == "inventory/reservation.go")
        final_test = next(item["content"] for item in proposed["candidateFiles"]
                          if item["path"] == "inventory/reservation_test.go")
        review = invocation("review", "verifier", artifacts=artifacts(final_source, final_test))
        reviewed = deterministic_delegate.response(review, "review")
        self.assertEqual(reviewed["outcome"], "passed")
        self.assertEqual(reviewed["verifierObservations"][0]["outcome"], "passed")

        vote = invocation("vote", "verifier", context={
            "candidate": {"id": "sha256:" + "5" * 64},
            "evidence": {"id": "sha256:" + "6" * 64, "round": 1},
        }, artifacts=artifacts(final_source, final_test))
        voted = deterministic_delegate.response(vote, "vote")
        detail = json.loads(voted["verifierObservations"][0]["detail"])
        self.assertEqual(voted["verifierObservations"][0]["subject"], "government-vote")
        self.assertEqual(detail["outcome"], "assent")
        self.assertEqual(detail["materialCandidateId"], "sha256:" + "5" * 64)
        self.assertEqual(detail["evidenceId"], "sha256:" + "6" * 64)

    def test_failed_candidate_never_gets_positive_review_or_vote(self):
        bad = artifacts("package inventory\nfunc Release(a, b int64) (int64, error) { return a+b, nil }\n", "package inventory\n")
        review = deterministic_delegate.response(invocation("review", "verifier", artifacts=bad), "review")
        self.assertEqual(review["outcome"], "failed")
        vote = deterministic_delegate.response(invocation("vote", "verifier", context={
            "candidate": {"id": "sha256:" + "5" * 64},
            "evidence": {"id": "sha256:" + "6" * 64, "round": 1},
        }, artifacts=bad), "vote")
        self.assertEqual(vote["outcome"], "failed")
        self.assertEqual(json.loads(vote["verifierObservations"][0]["detail"])["outcome"], "objection")

    def test_role_call_summary_and_native_command_order_are_explicit(self):
        calls = integration.expected_queue_role_calls()
        self.assertEqual((calls["executor"], calls["independentVerifier"], calls["selectedRessortVotes"],
                          calls["queueTotal"], calls["resumeExpectedAdditional"]), (1, 1, 1, 3, 0))
        backlog = integration.build_backlog(runtime_path=Path("C:/fixture/runtime.json"),
                                            queue_state_directory=Path("C:/fixture/queue"))
        self.assertEqual(backlog["limits"]["maxWallTimeSeconds"], 40)

    def test_product_binding_uses_the_actual_accepted_worker04e_files(self):
        with tempfile.TemporaryDirectory(prefix="government-positive-bindings-") as temporary:
            root = Path(temporary)
            repo = root / "repo"
            marker = integration.create_disposable_repository(repo)
            integration.bind_order_to_inspection(repo, "sha256:" + "7" * 64)
            paths = {name: root / name for name in ("runtime.json", "backlog.json", "authorization.json")}
            raw = {name: f"{name}-fixture-bytes".encode() for name in paths}
            for name, path in paths.items():
                path.write_bytes(raw[name])
            product, released = integration.product_binding(
                repository=repo, runtime_path=paths["runtime.json"], runtime_raw=raw["runtime.json"],
                backlog_path=paths["backlog.json"], backlog_raw=raw["backlog.json"],
                authorization_path=paths["authorization.json"], authorization_raw=raw["authorization.json"],
                queue_state_directory=root / "queue")
            accepted = government.PIN["accepted"]
            self.assertEqual(product["executable"]["sourceCommit"], "04e225d5caee78c2a198607143863fca1e829750")
            self.assertEqual(product["executable"]["sha256"], "12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f")
            self.assertEqual(product["handoff"]["sha256"], "b83103dc1f0c9d9985bfe540692869754abbd40249809e55c6aebc4a9d24417b")
            self.assertEqual({item["path"] for item in released}, {
                str(Path(accepted["handoff"]["path"]).resolve()), str(paths["runtime.json"].resolve()),
                str(paths["backlog.json"].resolve()), str(paths["authorization.json"].resolve()),
            })


if __name__ == "__main__":
    unittest.main()
