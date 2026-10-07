"""Pure Government v1.2 translation tests; no native process or Actor starts."""
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import government


def write(path, value):
    raw = value if isinstance(value, bytes) else json.dumps(value, sort_keys=True).encode()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(raw)
    return hashlib.sha256(raw).hexdigest()


class GovernmentTranslationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="government-adapter-mechanics-")
        self.real_pin = government.PIN
        self.root = Path(self.temp.name)
        self.repo = self.root / "actor"
        self.repo.mkdir()
        self.evidence = self.root / "evidence"
        self.evidence.mkdir()
        self.state = self.root / "queue-state"
        self.run_state = self.root / "run-state"
        self.run_state.mkdir()
        self.outside = self.root / "inputs"
        self.binary = self.root / "markitect.exe"
        self.binary_sha = write(self.binary, b"pinned provisional test binary")
        self.handoff = self.outside / "handoff.json"
        self.handoff_sha = write(self.handoff, {"phase": "G5", "sourceSHA": "c" * 40,
                                                "status": "provisional-pending-review",
                                                "binary": {"path": str(self.binary), "sha256": self.binary_sha}})
        government.PIN = {"accepted": {"sourceCommit": "c" * 40,
                                        "handoff": {"path": str(self.handoff), "sha256": self.handoff_sha},
                                        "binary": {"path": str(self.binary), "sha256": self.binary_sha},
                                        "acceptance": {"status": "accepted"}}}
        self.config = self.repo / "government.yaml"
        self.config_sha = write(self.config, b"apiVersion: markitect.government/v1alpha1\n")
        self.order = self.repo / "orders/task.yaml"
        self.order_sha = write(self.order, b"apiVersion: markitect.government/v1alpha1\n")
        self.runtime = self.outside / "runtime.json"
        runtime_value = {"apiVersion": government.RUN_API, "stateDirectory": str(self.run_state),
                         "timeoutSeconds": 300,
                         "executor": {"slotId": "writer", "command": "C:/model.exe", "model": "model-x"},
                         "verifier": {"slotId": "reviewer", "command": "C:/model.exe", "model": "model-x"},
                         "recursion": {"areas": [{"area": {"apiVersion": "markitect.government/v1alpha1",
                                                               "kind": "Area", "namespace": "orders", "name": "child"},
                                                     "executor": {"slotId": "child-writer", "command": "C:/model.exe", "model": "model-x"},
                                                     "verifier": {"slotId": "child-reviewer", "command": "C:/model.exe", "model": "model-x"}}]},
                         "ressorts": [{"ressort": {"namespace": "orders", "name": "finance"},
                                       "runner": {"slotId": "vote-finance", "command": "C:/model.exe", "model": "model-x"}}]}
        self.runtime_sha = write(self.runtime, runtime_value)
        self.backlog = self.outside / "backlog.json"
        self.backlog_value = {"apiVersion": government.QUEUE_API, "stateDirectory": str(self.state),
                              "limits": {"actorStarts": 4, "maxRepairs": 1,
                                         "maxWallTimeSeconds": 600, "maxParallelism": 1},
                              "jobs": [{"id": "task-1", "dependsOn": [], "configPath": "government.yaml",
                                        "orderPath": "orders/task.yaml", "runtimePath": str(self.runtime)}]}
        self.backlog_sha = write(self.backlog, self.backlog_value)
        self.request = {"schemaVersion": 1, "trialId": "trial", "dispatchId": "task-1-dispatch",
                        "operation": "run_task", "mode": "live", "arm": "government",
                        "condition": "greenfield", "actorRepository": str(self.repo),
                        "evidenceDirectory": str(self.evidence),
                        "task": {"id": "task-1"},
                        "limits": {"taskActorCalls": 12, "maxSemanticRepairRoundsPerTask": 2,
                                   "taskWallSeconds": 1200, "maxParallelActors": 4},
                        "releasedInputs": [{"path": str(self.handoff), "sha256": self.handoff_sha},
                                           {"path": str(self.runtime), "sha256": self.runtime_sha},
                                           {"path": str(self.backlog), "sha256": self.backlog_sha}],
                        "product": {"government": {
                            "handoff": {"path": str(self.handoff), "sha256": self.handoff_sha},
                            "executable": {"path": str(self.binary), "sha256": self.binary_sha,
                                            "sourceCommit": "c" * 40},
                            "projectConfig": {"path": "government.yaml", "sha256": self.config_sha},
                            "order": {"path": "orders/task.yaml", "sha256": self.order_sha},
                            "runtime": {"path": str(self.runtime), "sha256": self.runtime_sha},
                            "backlog": {"path": str(self.backlog), "sha256": self.backlog_sha},
                            "queueStateDirectory": str(self.state), "pinStatus": "provisional"}}}

    def tearDown(self):
        government.PIN = self.real_pin
        self.temp.cleanup()

    def test_request_binding_and_review_only_queue_plan(self):
        bound = government.bind_request(self.request)
        self.assertEqual([role["slotId"] for role in bound["roles"]],
                         ["writer", "reviewer", "child-writer", "child-reviewer", "vote-finance"])
        plan = government.plan_request(self.request)
        self.assertFalse(plan["dispatchable"])
        self.assertEqual(plan["argv"], [str(self.binary.resolve()), "government", "--repo", str(self.repo.resolve()),
                                         "--action", "queue", "--backlog", str(self.backlog.resolve()), "--write"])
        self.assertTrue(any("accepted for preparation" in gap for gap in plan["gaps"]))
        self.assertTrue(any("not yet wired" in gap for gap in plan["gaps"]))

    def test_static_binding_allows_controller_to_create_evidence_and_run_state(self):
        self.evidence.rmdir()
        self.run_state.rmdir()
        bound = government.bind_request(self.request)
        self.assertEqual(bound["runStateDirectory"], self.run_state.resolve())
        self.assertFalse(self.evidence.exists())
        self.assertFalse(self.run_state.exists())

    def test_native_fixture_backlog_cannot_exceed_source_parallel_limit(self):
        self.request["nativeFixtureGrant"] = {"path": str(self.root / "grant.json"),
                                               "sha256": "0" * 64,
                                               "sourceKey": "native-s1-integration-fixtures-20261008"}
        self.backlog_value["limits"]["maxParallelism"] = 3
        self.backlog_sha = write(self.backlog, self.backlog_value)
        self.request["releasedInputs"][-1]["sha256"] = self.backlog_sha
        self.request["product"]["government"]["backlog"]["sha256"] = self.backlog_sha
        with self.assertRaisesRegex(ValueError, "source-grant parallel limit"):
            government.bind_request(self.request)

    def test_native_fixture_backlog_allows_zero_semantic_repairs(self):
        self.request["nativeFixtureGrant"] = {"path": str(self.root / "grant.json"),
                                               "sha256": "0" * 64,
                                               "sourceKey": "native-s1-integration-fixtures-20261008"}
        self.backlog_value["limits"]["maxRepairs"] = 0
        self.backlog_sha = write(self.backlog, self.backlog_value)
        self.request["releasedInputs"][-1]["sha256"] = self.backlog_sha
        self.request["product"]["government"]["backlog"]["sha256"] = self.backlog_sha
        bound = government.bind_request(self.request)
        self.assertEqual(bound["backlogValue"]["limits"]["maxRepairs"], 0)

    def test_pin_preserves_accepted_source_and_superseded_provisional_history(self):
        accepted = self.real_pin["accepted"]
        history = self.real_pin["history"]["priorProvisional"]
        self.assertEqual(accepted["sourceCommit"], "04e225d5caee78c2a198607143863fca1e829750")
        self.assertEqual(accepted["handoff"]["sha256"], "b83103dc1f0c9d9985bfe540692869754abbd40249809e55c6aebc4a9d24417b")
        self.assertEqual(accepted["binary"]["sha256"], "12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f")
        self.assertEqual(accepted["acceptance"]["sourceThreadId"], "01a11367-a781-7683-a20f-46e12614dcb4")
        self.assertEqual(history["sourceCommit"], "ce021cef2d23002c3eca0b6a81e0913b3fc5a4f5")
        self.assertEqual(history["status"], "superseded-provisional")

    def test_request_cannot_substitute_arbitrary_government_pin(self):
        test_pin = government.PIN
        government.PIN = self.real_pin
        try:
            with self.assertRaisesRegex(ValueError, "exact accepted Government G5 pin"):
                government.bind_request(self.request)
        finally:
            government.PIN = test_pin

    def test_resume_argv_binds_same_queue(self):
        queue = self.state / "queue--abc"
        queue.mkdir(parents=True)
        self.request["operation"] = "resume"
        self.request["product"]["government"]["queueDirectory"] = str(queue)
        plan = government.plan_request(self.request)
        self.assertEqual(plan["argv"], [str(self.binary.resolve()), "government", "--repo", str(self.repo.resolve()),
                                         "--action", "resume", "--backlog", str(self.backlog.resolve()),
                                         "--queue", str(queue.resolve()), "--write"])
        self.assertFalse(plan["dispatchable"])

    def _report(self, run_id="government-run-abc"):
        candidate, evidence = "sha256:" + "1" * 64, "sha256:" + "2" * 64
        mandate = {"apiVersion": "markitect.government/v1alpha1", "kind": "Mandate",
                   "namespace": "orders", "name": "prior"}
        return {"apiVersion": government.RUN_API, "runId": run_id, "status": "accepted-scoped",
                "priorConstitution": "sha256:" + "9" * 64,
                "plan": {"integrationReviews": [{"namespace": "orders", "name": "root"}]},
                "evidence": {"id": evidence, "materialCandidateId": candidate, "round": 1},
                "cabinet": [{"ressort": {"apiVersion": "markitect.government/v1alpha1", "kind": "Ressort",
                                           "namespace": "orders", "name": "finance"},
                             "priorMandate": mandate, "mandateDigest": "sha256:" + "4" * 64,
                             "slotId": "vote-finance"}],
                "actors": [{"phase": "execute", "slotId": "writer", "result": {
                    "Response": {"runId": "actor-1", "role": "executor", "inputDigest": "sha256:" + "3" * 64},
                    "Receipt": {"runId": "actor-1", "inputDigest": "sha256:" + "3" * 64}}},
                           {"phase": "execute", "slotId": "child-writer", "result": {
                    "Response": {"runId": "actor-3", "role": "executor", "inputDigest": "sha256:" + "7" * 64},
                    "Receipt": {"runId": "actor-3", "inputDigest": "sha256:" + "7" * 64}}},
                           {"phase": "review", "slotId": "reviewer", "scopes": ["orders/root"], "result": {
                    "Response": {"runId": "actor-review", "role": "verifier", "inputDigest": "sha256:" + "8" * 64,
                                 "outcome": "passed", "uncertainty": [],
                                 "verifierObservations": [{"subject": "orders/root", "outcome": "passed",
                                                           "detail": "reviewed planned integration scope"}]},
                    "Receipt": {"runId": "actor-review", "inputDigest": "sha256:" + "8" * 64}}},
                           {"phase": "vote", "slotId": "vote-finance", "result": {
                    "Response": {"runId": "actor-2", "role": "verifier", "inputDigest": "sha256:" + "5" * 64,
                                 "outcome": "passed", "uncertainty": [],
                                 "verifierObservations": [{"subject": "government-vote", "outcome": "passed",
                                     "detail": json.dumps({"outcome": "assent-unaffected", "reason": "reviewed exact evidence",
                                                           "materialCandidateId": candidate, "evidenceId": evidence,
                                                           "round": 1})}]},
                    "Receipt": {"runId": "actor-2", "inputDigest": "sha256:" + "5" * 64}}}],
                "votes": [{"ressort": {"namespace": "orders", "name": "finance"},
                           "id": "sha256:" + "6" * 64,
                           "priorMandate": mandate, "mandateDigest": "sha256:" + "4" * 64,
                           "materialCandidateId": candidate, "evidenceId": evidence,
                           "round": 1, "outcome": "assent-unaffected",
                           "provenance": {"runId": "actor-2", "slotId": "vote-finance"}}],
                "decision": {"id": "sha256:" + "a" * 64, "priorAuthorityDigest": "sha256:" + "9" * 64,
                             "materialCandidateId": candidate, "evidenceId": evidence, "round": 1,
                             "voteIds": ["sha256:" + "6" * 64]},
                "promotion": {"status": "promoted", "intentPath": "", "completionPath": ""}}

    def _request_bytes(self):
        return json.dumps(self.request, sort_keys=True).encode()

    def _queue(self, status="complete"):
        queue_id = "queue--abc"
        queue = self.state / queue_id
        run_id = "government-run-abc"
        run_dir = self.run_state / run_id
        run_path = run_dir / "report.json"
        report = self._report(run_id)
        intent_name = "promotion-" + hashlib.sha256(run_id.encode()).hexdigest() + ".json"
        completion_name = "promotion-completion-" + hashlib.sha256(run_id.encode()).hexdigest() + ".json"
        write(run_dir / intent_name, b'{"synthetic":"promotion intent"}')
        write(run_dir / completion_name, b'{"synthetic":"promotion completion"}')
        report["promotion"].update({"intentPath": str(run_dir / intent_name),
                                    "completionPath": str(run_dir / completion_name)})
        report_sha = write(run_path, report)
        job = {"id": "task-1", "state": "accepted-scoped", "runId": run_id,
               "reportPath": str(run_path), "reportDigest": report_sha}
        value = {"apiVersion": government.QUEUE_API, "queueId": queue_id, "queueDirectory": str(queue),
                 "status": status, "backlogDigest": self.backlog_sha,
                 "limits": {"actorStarts": 4, "maxParallelism": 1}, "actorStarts": 1,
                 "inFlightActors": 0, "journalSequence": 1, "journalDigest": "a" * 64,
                 "usage": {"inputTokens": {"known": True, "total": 10, "unknown": False},
                           "outputTokens": {"known": True, "total": 5, "unknown": False},
                           "cachedTokens": {"known": False, "total": 0, "unknown": True}},
                 "jobs": [job]}
        queue.mkdir(parents=True, exist_ok=True)
        write(queue / "queue-report-00000001.json", value)
        write(queue / "events.jsonl", b'{"sequence":1}\n')
        stdout = self.evidence / "queue.stdout.json"
        write(stdout, value)
        return stdout, value

    def test_result_collects_bound_roles_votes_and_promotion_without_acceptance_claim(self):
        stdout, _ = self._queue()
        result = government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)
        self.assertEqual(result["status"], "completed")
        self.assertEqual(result["requestSha256"], hashlib.sha256(self._request_bytes()).hexdigest())
        self.assertIsNone(result["candidateCommit"])
        self.assertIsNone(result["usage"]["providerRequests"])
        self.assertEqual(result["usage"]["reportedInputPlusOutputTokens"], 15)
        kinds = {receipt["kind"] for receipt in result["receipts"]}
        self.assertIn("government-role:execute:writer:executor", kinds)
        self.assertIn("government-role:execute:child-writer:executor", kinds)
        self.assertIn("government-vote:orders/finance:assent-unaffected", kinds)
        self.assertIn("government-promotion:promoted", kinds)
        self.assertTrue(any("not independent task acceptance" in gap for gap in result["gaps"]))
        with self.assertRaisesRegex(ValueError, "differs from exact Request"):
            government.translate_queue_result(self.request, b"{}", stdout, 0)

    def test_native_blocked_status_wins_over_zero_exit_and_bad_report_binding_fails(self):
        stdout, _ = self._queue("blocked")
        request_raw = self._request_bytes()
        result = government.translate_queue_result(self.request, request_raw, stdout, 0)
        self.assertEqual(result["status"], "blocked")
        self.assertEqual(government.translate_queue_result(self.request, request_raw, stdout, 1)["status"], "failed")
        value = json.loads(stdout.read_bytes())
        value["jobs"][0]["reportDigest"] = "0" * 64
        queue_dir = Path(value["queueDirectory"])
        write(queue_dir / "queue-report-00000001.json", value)
        write(stdout, value)
        with self.assertRaisesRegex(ValueError, "report path/digest mismatch"):
            government.translate_queue_result(self.request, request_raw, stdout, 0)
        stdout, value = self._queue("complete")
        for key in ("inputTokens", "outputTokens"):
            value["usage"][key] = {"known": False, "total": 0, "unknown": True}
        queue_dir = Path(value["queueDirectory"])
        write(queue_dir / "queue-report-00000001.json", value)
        write(stdout, value)
        unknown = government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)
        self.assertIsNone(unknown["usage"]["reportedInputPlusOutputTokens"])
        self.assertIsNone(unknown["usage"]["providerRequests"])

    def test_backlog_future_job_is_rejected(self):
        self.backlog_value["jobs"].append(dict(self.backlog_value["jobs"][0], id="future"))
        self.backlog_sha = write(self.backlog, self.backlog_value)
        self.request["product"]["government"]["backlog"]["sha256"] = self.backlog_sha
        self.request["releasedInputs"][2]["sha256"] = self.backlog_sha
        with self.assertRaisesRegex(ValueError, "one Request-bound Government job"):
            government.bind_request(self.request)

    def test_external_run_report_path_is_rejected(self):
        stdout, value = self._queue()
        outside = self.root / "outside-report.json"
        outside_digest = write(outside, self._report())
        value["jobs"][0]["reportPath"] = str(outside)
        value["jobs"][0]["reportDigest"] = outside_digest
        queue_dir = Path(value["queueDirectory"])
        write(queue_dir / "queue-report-00000001.json", value)
        write(stdout, value)
        with self.assertRaisesRegex(ValueError, "exact native run directory"):
            government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)

    def test_native_run_report_and_promotion_layout_is_sibling_to_queue(self):
        stdout, value = self._queue()
        queue_dir = Path(value["queueDirectory"])
        report_path = Path(value["jobs"][0]["reportPath"])
        self.assertNotIn(queue_dir, report_path.parents)
        result = government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)
        self.assertIn("government-promotion-intent", {item["kind"] for item in result["receipts"]})
        self.assertIn("government-promotion-completion", {item["kind"] for item in result["receipts"]})

    def test_complete_queue_requires_accepted_job_report_and_event_log(self):
        stdout, value = self._queue()
        value["jobs"][0].pop("reportPath")
        value["jobs"][0].pop("reportDigest")
        value["jobs"][0].pop("runId")
        queue_dir = Path(value["queueDirectory"])
        write(queue_dir / "queue-report-00000001.json", value)
        write(stdout, value)
        with self.assertRaisesRegex(ValueError, "accepted Government job must bind"):
            government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)
        stdout, value = self._queue()
        (Path(value["queueDirectory"]) / "events.jsonl").unlink()
        with self.assertRaisesRegex(ValueError, "event log is required"):
            government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)

    def test_accepted_queue_job_without_evidence_identity_is_incomplete_with_bound_receipts(self):
        stdout, value = self._queue("complete")
        report_path = Path(value["jobs"][0]["reportPath"])
        report = json.loads(report_path.read_bytes())
        report.pop("evidence")
        report_sha = write(report_path, report)
        value["jobs"][0]["reportDigest"] = report_sha
        queue_dir = Path(value["queueDirectory"])
        write(queue_dir / "queue-report-00000001.json", value)
        write(stdout, value)

        result = government.translate_queue_result(self.request, self._request_bytes(), stdout, 0)
        self.assertEqual(result["status"], "incomplete")
        self.assertIsNone(result["candidateCommit"])
        self.assertEqual(result["usage"]["nativeRolesObserved"], [])
        kinds = {receipt["kind"] for receipt in result["receipts"]}
        self.assertTrue({"government-queue-stdout", "government-queue-report",
                         "government-queue-events", "government-run-report"}.issubset(kinds))
        self.assertNotIn("government-decision", kinds)
        self.assertTrue(any("lacks evidence identity" in gap for gap in result["gaps"]))

    def test_decision_round_must_match_evidence_round(self):
        report = self._report()
        report["decision"]["round"] = 2
        with self.assertRaisesRegex(ValueError, "bound to report evidence"):
            government._validate_run_report(report, "government-run-abc", {
                "writer": ("execute", "executor"),
                "vote-finance": ("vote", "verifier")})

    def test_accepted_report_requires_positive_votes_and_passing_planned_reviews(self):
        report = self._report()
        configured = {"writer": ("execute", "executor"), "reviewer": ("review", "verifier"),
                      "child-writer": ("execute", "executor"), "vote-finance": ("vote", "verifier")}
        government._validate_run_report(report, "government-run-abc", configured,
                                        require_acceptance=True, root_review_slots={"reviewer"})
        no_decision = json.loads(json.dumps(report))
        no_decision["decision"] = None
        with self.assertRaisesRegex(ValueError, "requires a complete native AcceptanceDecision"):
            government._validate_run_report(no_decision, "government-run-abc", configured,
                                            require_acceptance=True, root_review_slots={"reviewer"})
        objection = json.loads(json.dumps(report))
        objection["votes"][0]["outcome"] = "objection"
        with self.assertRaisesRegex(ValueError, "positive final votes"):
            government._validate_run_report(objection, "government-run-abc", configured,
                                            require_acceptance=True, root_review_slots={"reviewer"})
        no_review = json.loads(json.dumps(report))
        no_review["actors"] = [actor for actor in no_review["actors"] if actor["phase"] != "review"]
        with self.assertRaisesRegex(ValueError, "root-review actor receipt"):
            government._validate_run_report(no_review, "government-run-abc", configured,
                                            require_acceptance=True, root_review_slots={"reviewer"})


if __name__ == "__main__":
    unittest.main(verbosity=2)
