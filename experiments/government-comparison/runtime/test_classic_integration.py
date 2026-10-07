"""Focused Classic fixture mapping/plan/normalization checks; no product or provider calls."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import classic_integration as integration
import government_roles
from ledger import Ledger
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile_sha

PACKET = Path(integration.PACKET)


class ClassicIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="classic-integration-test-")
        self.root = Path(self.temp.name).resolve()

    def tearDown(self):
        self.temp.cleanup()

    @staticmethod
    def invocation(role, projection, scope):
        return {"apiVersion": "markitect.example.org/agent-execution/v1alpha1", "runId": "fixture-run",
                "nonce": "fixture-nonce", "inputDigest": "sha256:" + "a" * 64,
                "request": {"role": role, "sourceRevision": "b" * 40, "modelDigest": "sha256:" + "c" * 64,
                             "modulePin": {"name": "markitect-dotnet", "version": "1.0.0", "digest": "sha256:" + "d" * 64},
                             "projectionId": projection, "scopeIds": [scope], "policyIds": [],
                             "context": {"nativeContext": "opaque; preserve exactly"}, "artifacts": []}}

    def test_native_projection_to_fixed_classic_slot_preserves_invocation(self):
        cases = (("executor", integration.DOTNET_PROJECTION, "commerce-dotnet",
                  "execute", "classic/execute/commerce-dotnet"),
                 ("verifier", integration.DOTNET_PROJECTION, "commerce-dotnet",
                  "review", "classic/review/commerce-dotnet"),
                 ("verifier", integration.MARKDOWN_PROJECTION, "commerce-markdown",
                  "review", "classic/review/commerce-markdown"))
        for role, projection, scope, phase, slot in cases:
            with self.subTest(slot=slot):
                invocation = self.invocation(role, projection, scope)
                original = json.dumps(invocation, sort_keys=True)
                resolved = integration.resolve_role(invocation)
                self.assertEqual(set(resolved), {"slotId", "phase", "responseRole"})
                self.assertEqual((resolved["phase"], resolved["slotId"], resolved["responseRole"]),
                                 (phase, slot, role))
                self.assertEqual(json.dumps(invocation, sort_keys=True), original)
        for invocation in (
            self.invocation("executor", integration.MARKDOWN_PROJECTION, "commerce-markdown"),
            self.invocation("infer", integration.DOTNET_PROJECTION, "commerce-dotnet"),
        ):
            with self.assertRaisesRegex(ValueError, "not an authorized fixture slot"):
                integration.resolve_role(invocation)
        with self.assertRaisesRegex(ValueError, "missing the exact configured scope identity"):
            integration.resolve_role(self.invocation("verifier", integration.DOTNET_PROJECTION,
                                                      "commerce-markdown"))

    def test_invocation_replay_identity_is_stable_and_nonce_bound(self):
        invocation = self.invocation("executor", integration.DOTNET_PROJECTION, "commerce-dotnet")
        first = integration.invocation_call_id("dispatch-1", invocation)
        self.assertEqual(first, integration.invocation_call_id("dispatch-1", invocation))
        changed = json.loads(json.dumps(invocation))
        changed["nonce"] = "different-nonce"
        self.assertNotEqual(first, integration.invocation_call_id("dispatch-1", changed))
        self.assertNotEqual(first, integration.invocation_call_id("dispatch-2", invocation))

    def test_shared_role_ledger_refuses_exact_replay_before_delegate_or_second_attempt(self):
        limits = {"taskWallSeconds": 1200, "trialWallSeconds": 7200, "taskActorCalls": 12,
                  "trialActorCalls": 72, "taskProviderTurns": 80, "trialProviderTurns": 480,
                  "taskProviderTokens": 120000, "trialProviderTokens": 720000,
                  "maxParallelActors": 4, "maxTransportRetriesPerCall": 1,
                  "maxSemanticRepairRoundsPerTask": 2, "trialActiveHumanSeconds": 600,
                  "newPurchases": False}
        path = self.root / "replay-ledger.sqlite"
        ledger = Ledger(path, "classic-replay-fixture", limits, profile_id=LEGACY)
        approval = {"status": "approved", "trialId": "classic-replay-fixture", "ledgerPath": str(path.resolve()),
                    "fromProfileSha256": profile_sha(LEGACY), "toProfileSha256": profile_sha(OBSERVED),
                    "limitsSha256": limits_sha(limits), "decisionRef": "decision://synthetic-classic-replay"}
        ledger.adopt_profile(OBSERVED, approval)
        authority = {"grantId": "synthetic-classic-replay-grant", "protocolSha256": "a" * 64,
                     "profileSha256": profile_sha(OBSERVED), "limitsSha256": limits_sha(limits)}
        ledger.bind_dispatch(authority, maximum=2)
        dispatch_id = "synthetic-classic-dispatch"
        ledger.reserve_dispatch(dispatch_id, "b" * 64, b'{"synthetic":true}', ["fake-adapter", "--never-run"],
                                "classic-task", "task", 2)
        self.assertTrue(ledger.claim_dispatch(dispatch_id, "launching"))
        invocation = self.invocation("executor", integration.DOTNET_PROJECTION, "commerce-dotnet")
        resolved = integration.resolve_role(invocation)
        auth = {"dispatchId": dispatch_id, "maxCalls": 3, "taskId": "classic-task"}
        call_args = (ledger, auth, invocation, "c" * 64, "d" * 64, resolved,
                     self.root / "role-evidence" / "one", 10000)
        call_id, first_attempt = government_roles._reserve(*call_args)
        self.assertEqual(call_id, integration.invocation_call_id(dispatch_id, invocation))
        with self.assertRaisesRegex(ValueError, "agentexec invocation replay is already reserved; no delegate relaunch"):
            government_roles._reserve(*call_args)
        with ledger.transaction() as db:
            self.assertEqual(db.execute("SELECT COUNT(*) FROM attempts").fetchone()[0], 2,
                             "the repeated Invocation must not reserve a second role attempt")
            self.assertEqual(db.execute("SELECT COUNT(*) FROM government_role_calls WHERE dispatch_id=?",
                                        (dispatch_id,)).fetchone()[0], 1)
            self.assertEqual(db.execute("SELECT attempt_id,status FROM government_role_calls WHERE call_id=?",
                                        (call_id,)).fetchone(), (first_attempt, "reserved"))
        self.assertFalse((self.root / "role-evidence").exists(), "replay refusal happens before evidence/delegate launch")

    def test_prepare_real_pinned_disposable_fixture_and_exact_initial_native_plan(self):
        destination = self.root / "prepared"
        result = integration.prepare_fixture(PACKET, destination)
        self.assertFalse(result["nativeOrActorProcessesStarted"])
        self.assertEqual(result["productSource"], integration.CLASSIC_SOURCE)
        self.assertEqual(len(result["fixtureSourceCommit"]), 40)
        repo = Path(result["fixtureRepo"])
        runtime = json.loads(Path(result["runtimePath"]).read_text(encoding="utf-8"))
        self.assertFalse(Path(runtime["recordStore"]).exists(),
                         "fixture preparation must leave native RecordStore initialization to Classic")
        authorization_raw = Path(result["roleAuthorizationPath"]).read_bytes()
        authorization = json.loads(authorization_raw)
        self.assertEqual(authorization["status"], "approved")
        self.assertEqual(authorization["runtimePath"], result["runtimePath"])
        self.assertEqual(authorization["dispatchId"], "classic-native-positive")
        self.assertEqual(authorization["trialId"], "native-fixture-classic-20261008")
        self.assertEqual(authorization["taskId"], "native-classic-positive")
        self.assertEqual(authorization["fixtureAuthorization"], {
            "sourceGrantKey": "native-s1-integration-fixtures-20261008",
            "classification": "nativeFixture", "providerUse": "noProvider"})
        self.assertEqual(runtime["executor"]["model"], "protocol-test-double")
        self.assertEqual(runtime["executor"]["providerVersion"], "protocol-test-double/1")
        self.assertEqual(runtime["executor"]["command"], sys.executable)
        self.assertIn("government_roles.py", runtime["executor"]["args"][0])
        self.assertIn("--authorization-sha256", runtime["executor"]["args"])
        self.assertEqual(runtime["executor"]["args"][runtime["executor"]["args"].index("--authorization-sha256") + 1],
                         integration.sha256(authorization_raw))
        self.assertNotIn("protocol_test_double.py", runtime["executor"]["args"],
                         "the packet double must be behind the shared wrapper")
        pinned_runtime_files = {Path(item["path"]).name for item in runtime["executor"]["runtimeFiles"]}
        self.assertIn("native_fixture_budget.py", pinned_runtime_files)
        self.assertIn("classic_integration.py", pinned_runtime_files)
        self.assertEqual(authorization["slots"][0]["delegate"]["argv"][1].endswith("protocol_test_double.py"), True)
        self.assertTrue(runtime["auditAll"])
        self.assertEqual(runtime["assuranceRoots"], ["commerce-markdown"])
        self.assertEqual([scope["id"] for scope in runtime["assuranceScopes"]],
                         ["commerce-markdown", "commerce-dotnet"])
        roles = integration.configured_roles(runtime)
        self.assertEqual([(role["phase"], role["slotId"]) for role in roles], [
            ("execute", "classic/execute/commerce-dotnet"),
            ("review", "classic/review/commerce-dotnet"),
            ("review", "classic/review/commerce-markdown"),
        ])
        self.assertEqual(roles[1]["args"], roles[2]["args"],
                         "one native Verifier config serves both scopes; slot comes from the Invocation")
        self.assertTrue(result["preparation"]["qualification"].startswith("Explicit existing test-fixture setup"))
        self.assertEqual([call["action"] for call in json.loads(Path(result["planPath"]).read_text(encoding="utf-8"))["calls"]],
                         ["execute"])
        plan = json.loads(Path(result["planPath"]).read_text(encoding="utf-8"))
        execute = plan["calls"][-1]
        self.assertEqual(execute["argv"][execute["argv"].index("--action") + 1], "controller-execute")
        self.assertEqual(execute["argv"][execute["argv"].index("--base") + 1], result["fixtureSourceCommit"])
        self.assertEqual(Path(execute["stdoutPath"]), Path(result["reportsDirectory"]) / "execute.json")
        self.assertFalse(Path(execute["stdoutPath"]).exists())
        status = __import__("subprocess").run(["git", "-C", str(repo), "status", "--porcelain"],
                                              capture_output=True, check=True).stdout
        self.assertEqual(status, b"")

    def test_prepare_runtime_leaves_record_store_absent_and_preserves_preexisting_empty_root(self):
        prepared = integration.prepare_fixture(PACKET, self.root / "record-store-check")
        runtime = json.loads(Path(prepared["runtimePath"]).read_text(encoding="utf-8"))
        record_store = Path(runtime["recordStore"])
        self.assertFalse(record_store.exists())
        self.assertEqual(integration.assert_record_store_absent(runtime), record_store)

        state = self.root / "preexisting-controller-state"
        root = state / "ledger"
        root.mkdir(parents=True)
        self.assertEqual(list(root.iterdir()), [])
        rejected_runtime = Path(prepared["runtimePath"]).with_name("must-not-be-written.json")
        with self.assertRaisesRegex(FileExistsError, "must be absent before native initialization"):
            integration.prepare_protocol_runtime(
                PACKET, self.root / "record-store-check", prepared["fixtureRepo"],
                authorization_path=prepared["roleAuthorizationPath"],
                authorization_sha256=prepared["roleAuthorizationSha256"],
                role_evidence_directory=self.root / "record-store-check" / "role-evidence",
                state_directory=state, runtime_path=rejected_runtime)
        self.assertTrue(root.is_dir(), "the preexisting root must be preserved")
        self.assertEqual(list(root.iterdir()), [], "the helper must not alter or populate an old empty root")
        self.assertFalse((state / "private-logs").exists(), "reject before creating sibling controller state")
        self.assertFalse(rejected_runtime.exists(), "reject before writing a runtime bound to old state")
        with self.assertRaisesRegex(FileExistsError, "must be absent before native initialization"):
            integration.assert_record_store_absent({"recordStore": str(root)})

    def test_bind_request_validates_external_auth_runtime_pins_and_clean_actor_base(self):
        prepared = integration.prepare_fixture(PACKET, self.root / "bound")
        repo = Path(prepared["fixtureRepo"])
        released = Path(prepared["runtimePath"]).parent
        request_path = released / "request.json"
        auth_path = Path(prepared["roleAuthorizationPath"])
        runtime_path = Path(prepared["runtimePath"])
        pin = json.loads(Path(__file__).with_name("classic-pin.json").read_text(encoding="utf-8"))
        request = {"schemaVersion": 1, "trialId": "native-fixture-classic-20261008",
                   "dispatchId": "classic-native-positive", "operation": "run_task",
                   "mode": "mechanical", "arm": "classic", "baseCommit": prepared["fixtureSourceCommit"],
                   "task": {"id": "native-classic-positive"}, "actorRepository": str(repo),
                   "evidenceDirectory": str(self.root / "bound-evidence"),
                   "releasedInputs": [{"path": str(auth_path), "sha256": integration.sha256(auth_path.read_bytes())},
                                      {"path": str(runtime_path), "sha256": integration.sha256(runtime_path.read_bytes())}],
                   "product": {"classic": {
                       "packet": {"path": str(PACKET), "manifestSha256": pin["packetManifestSha256"]},
                       "executable": {"path": str(PACKET / "artifacts" / "markitect-v0.14.1-windows-amd64.exe"),
                                      "sha256": integration.classic.EXPECTED_BINARY_SHA256,
                                      "sourceCommit": integration.classic.EXPECTED_SOURCE,
                                      "heldSourceCommit": integration.classic.EXPECTED_HELD_SOURCE},
                       "projectConfig": {"path": "examples/canonical-projection/canonical.yaml",
                                         "sha256": integration.sha256((repo / "examples/canonical-projection/canonical.yaml").read_bytes())},
                       "controllerActions": list(integration.CONTROLLER_ACTIONS),
                       "runtime": {"path": str(runtime_path), "sha256": integration.sha256(runtime_path.read_bytes())},
                       "roleAuthorization": {"path": str(auth_path), "sha256": integration.sha256(auth_path.read_bytes())}}}}
        request_path.write_bytes(json.dumps(request, sort_keys=True).encode())
        bound = integration.bind_request(request, request_path)
        self.assertEqual(bound["baseCommit"], request["baseCommit"])
        self.assertEqual(len(bound["roles"]), 3)
        # Simulate only the workspace-dirty state after a same-session Apply;
        # no product command is run and this file is not treated as a candidate.
        (repo / "materialized-workspace-test-only.txt").write_text("synthetic dirty-workspace sentinel\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "must be clean"):
            integration.bind_request(request, request_path)
        materialized = integration.bind_request(request, request_path, allow_materialized_workspace=True)
        self.assertEqual(materialized["baseCommit"], request["baseCommit"])
        with self.assertRaisesRegex(ValueError, "explicit materialized-workspace boolean"):
            integration.bind_request(request, request_path, allow_materialized_workspace=1)
        changed = json.loads(json.dumps(request))
        changed["product"]["classic"]["executable"]["sourceCommit"] = "0" * 40
        with self.assertRaisesRegex(ValueError, "source/binary binding"):
            integration.bind_request(changed, request_path, allow_materialized_workspace=True)

    def test_exact_review_gates_apply_and_fresh_verify_then_records_result_captures(self):
        root = self.root
        repo = root / "repo"
        repo.mkdir()
        runtime = root / "runtime.json"
        runtime.write_text("{}", encoding="utf-8")
        reports = root / "reports"
        reports.mkdir()
        execute_path = reports / "execute.json"
        execute_report = {"status": "planned", "digest": "sha256:" + "e" * 64}
        execute_raw = json.dumps(execute_report, separators=(",", ":")).encode()
        execute_path.write_bytes(execute_raw)
        bad_review = {"status": "approved", "executeReportSha256": "0" * 64,
                      "executeDigest": execute_report["digest"], "reviewerReference": "review://fixture"}
        with self.assertRaisesRegex(ValueError, "exact Execute report"):
            integration.review_execute(execute_path, bad_review)
        review = {"status": "approved", "executeReportSha256": integration.sha256(execute_raw),
                  "executeDigest": execute_report["digest"], "reviewerReference": "review://external-fixture-review"}
        plan = integration.plan_guarded_flow(PACKET, repo, runtime, reports, "f" * 40, execute_path, review)
        self.assertEqual([call["action"] for call in plan["calls"]], ["apply", "verify", "audit"])
        apply, verify, audit = plan["calls"]
        self.assertIn(execute_report["digest"], apply["argv"])
        self.assertEqual(verify["argv"][-3:], ["--apply-result", str(reports / "apply.json"), "--write"])
        self.assertNotIn("--write", audit["argv"])
        replay = plan["targetedReplay"]
        self.assertTrue(replay["notStarted"])
        self.assertEqual(replay["argv"], apply["argv"])

        reports_by_action = {
            "execute": {"status": "planned"},
            "apply": {"status": "materialized-unverified"}, "verify": {"status": "passed"},
            "audit": {"status": "complete", "findings": [], "nextSteps": []},
            "apply-replay": {"status": "stale", "materialized": False},
        }
        expected = {key: "json" for key in reports_by_action}
        captures = []
        for action, report_type in expected.items():
            stdout_path, stderr_path = reports / (action + ".out"), reports / (action + ".err")
            if report_type == "json":
                raw = json.dumps(reports_by_action[action], separators=(",", ":")).encode()
            elif report_type == "yaml":
                raw = b"fixture inspection report\n"
            else:
                raw = b"Markitect 0.14.1 (windows/amd64)\n"
            planned = {"argv": ["pinned-classic-fixture", action], "cwd": str(repo),
                       "expectedReport": report_type, "stdoutPath": str(stdout_path),
                       "stderrPath": str(stderr_path)}
            captured = {"argv": planned["argv"], "cwd": planned["cwd"],
                        "returnCode": 7 if action == "apply-replay" else 0,
                        "stdout": raw, "stderr": b"", "wallSeconds": 0.01, "timedOut": False,
                        "processTreeControl": "synthetic-test-only", "stopReason": None,
                        "productSourceSha": integration.CLASSIC_SOURCE,
                        "runtimeSha256": __import__("classic").EXPECTED_BINARY_SHA256}
            captures.append(integration.persist_native_capture(action, planned, captured))
        request = json.dumps({"schemaVersion": 1, "arm": "classic", "trialId": "synthetic-classic",
                              "operation": "run_task", "mode": "fixture"}, separators=(",", ":")).encode()
        result = integration.normalize_result(request, captures, inference_performed=False)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["candidateCommit"], None)
        self.assertFalse(result["classic"]["semanticAcceptance"])
        self.assertFalse(result["classic"]["humanAcceptance"])
        self.assertIsNone(result["usage"])
        self.assertEqual(result["requestSha256"], integration.sha256(request))
        self.assertEqual(len(result["receipts"]), 15)


if __name__ == "__main__":
    unittest.main()
