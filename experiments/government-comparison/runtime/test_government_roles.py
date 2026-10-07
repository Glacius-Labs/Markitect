"""Synthetic agentexec wrapper-boundary tests; no Markitect/provider process."""
import hashlib
import json
import os
from pathlib import Path
import stat
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import government
import government_roles
from dispatch import Authority, digest as dispatch_digest, encoded as dispatch_encoded
from ledger import Ledger, LimitReached
from measurement_profile import LEGACY, OBSERVED, limits_sha, profile_sha


def raw_json(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def go_json(value):
    """Compact insertion-order encoding for the documented Go struct fixture."""
    return json.dumps(value, separators=(",", ":"), ensure_ascii=False).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def write(path, raw):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(raw)
    return sha(raw)


def native_mode(path):
    mode = stat.S_IMODE(Path(path).stat().st_mode)
    if os.name == "nt":
        return "0644" if mode & stat.S_IWUSR else "0444"
    return f"{mode:04o}"


class GovernmentRoleBridgeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="government-role-bridge-")
        self.root = Path(self.temp.name).resolve()
        self.actor = self.root / "actor"
        self.actor.mkdir()
        self.evidence = self.root / "outer-evidence"
        self.evidence.mkdir()
        self.queue = self.root / "queue-state"
        self.run_state = self.root / "run-state"
        self.run_state.mkdir()
        self.role_evidence = self.root / "role-evidence"
        self.inputs = self.root / "released"
        self.inputs.mkdir()
        self.request_path = self.inputs / "request.json"
        self.auth_path = self.inputs / "role-auth.json"
        self.runtime_path = self.inputs / "runtime.json"
        self.grant_path = self.inputs / "grant.json"
        self.protocol_path = self.inputs / "protocol.json"
        self.delegate_path = self.inputs / "delegate.py"
        self.marker_path = self.root / "delegate-effects.txt"
        self.ledger_path = self.root / "ledger.sqlite"
        self.grant_sha = write(self.grant_path, b"synthetic already validated grant")
        self.protocol_sha = write(self.protocol_path, b"synthetic already validated protocol")

        delegate_source = f'''import json, pathlib, sys
inv=json.load(sys.stdin)
pathlib.Path({str(self.marker_path)!r}).open("a", encoding="utf-8").write(inv["runId"] + "\\n")
r=inv["request"]
response={{"apiVersion":inv["apiVersion"],"runId":inv["runId"],"nonce":inv["nonce"],
 "role":r["role"],"inputDigest":inv["inputDigest"],"outcome":"proposed",
 "candidateFiles":[],"evidenceRefs":[],"verifierObservations":[],"uncertainty":[],
 "usage":{{"source":"provider-reported","inputTokens":11,"outputTokens":7,
 "cachedTokens":400,"toolCalls":2}}}}
sys.stdout.write(json.dumps(response, separators=(",", ":")))
'''.encode()
        self.delegate_sha = write(self.delegate_path, delegate_source)
        self.python_digest = "sha256:" + sha(Path(sys.executable).read_bytes())
        self.delegate_file_digest = "sha256:" + self.delegate_sha
        self.native_python_digest = self.python_digest
        self.native_delegate_script_digest = self.delegate_file_digest
        self.model_config = {"model": "synthetic-model", "modelOptions": {"temperature": 0},
                             "providerVersion": "synthetic-provider/1"}
        self.max_calls = 4

        slot_specs = []
        for slot, phase, response_role in (("root-writer", "execute", "executor"),
                                           ("root-reviewer", "review", "verifier")):
            slot_specs.append({"slotId": slot, "phase": phase, "responseRole": response_role,
                               "wrapper": {"command": sys.executable},
                               "delegate": {"argv": [sys.executable, str(self.delegate_path)],
                                            "command": sys.executable,
                                            "commandDigest": self.native_python_digest,
                                            **self.model_config, "timeoutSeconds": 5,
                                            "maxStdoutBytes": 1_000_000, "maxStderrBytes": 1_000_000,
                                            "runtimeFiles": [
                                                {"path": sys.executable, "mode": native_mode(sys.executable), "digest": self.native_python_digest},
                                                {"path": str(self.delegate_path), "mode": native_mode(self.delegate_path), "digest": self.native_delegate_script_digest}]}})

        auth_value = {"apiVersion": government_roles.ROLE_AUTH_API, "status": "approved",
                      "trialId": "trial-bridge", "dispatchId": "dispatch-bridge", "taskId": "task-bridge",
                      "requestPath": str(self.request_path), "ledgerPath": str(self.ledger_path),
                      "runtimePath": str(self.runtime_path), "roleEvidenceDirectory": str(self.role_evidence),
                      "expiresAt": time.time() + 180, "maxCalls": self.max_calls, "slots": slot_specs}
        auth_raw_without_runtime = raw_json(auth_value)
        self.auth_sha = sha(auth_raw_without_runtime)
        self.auth_raw = auth_raw_without_runtime
        self.auth_sha = write(self.auth_path, self.auth_raw)

        wrapper_args = {slot["slotId"]: government_roles.wrapper_arguments(
            str(Path(government_roles.__file__).resolve()), str(self.auth_path), self.auth_sha,
            str(self.role_evidence), slot["slotId"])
            for slot in slot_specs}
        native_runtime_files = [
            {"path": str(Path(government_roles.__file__).resolve()), "mode": native_mode(government_roles.__file__),
             "digest": "sha256:" + sha(Path(government_roles.__file__).read_bytes())},
            {"path": str(self.auth_path), "mode": native_mode(self.auth_path), "digest": "sha256:" + self.auth_sha},
            {"path": sys.executable, "mode": native_mode(sys.executable), "digest": self.native_python_digest},
            {"path": str(self.delegate_path), "mode": native_mode(self.delegate_path), "digest": self.native_delegate_script_digest}]
        runtime_value = {"apiVersion": government.RUN_API, "timeoutSeconds": 30,
                         "stateDirectory": str(self.run_state),
                         "executor": {"slotId": "root-writer", "command": sys.executable,
                                      "args": wrapper_args["root-writer"], **self.model_config,
                                      "runtimeFiles": native_runtime_files},
                         "verifier": {"slotId": "root-reviewer", "command": sys.executable,
                                      "args": wrapper_args["root-reviewer"], **self.model_config,
                                      "runtimeFiles": native_runtime_files},
                         "ressorts": []}
        runtime_raw = raw_json(runtime_value)
        self.runtime_sha = write(self.runtime_path, runtime_raw)
        self.limits = json.loads((Path(__file__).parents[1] / "public" / "resource-proposal.json").read_bytes())["commonLimits"]
        self.request = {"schemaVersion": 1, "trialId": "trial-bridge", "dispatchId": "dispatch-bridge",
                        "operation": "run_task", "mode": "mechanical", "arm": "government",
                        "condition": "greenfield", "purpose": "task", "wallSeconds": 5,
                        "measurementProfileId": OBSERVED,
                        "measurementProfileSha256": profile_sha(OBSERVED),
                        "task": {"id": "task-bridge"}, "actorRepository": str(self.actor),
                        "evidenceDirectory": str(self.evidence), "limits": self.limits,
                        "releasedInputs": [{"path": str(self.runtime_path), "sha256": self.runtime_sha}],
                        "product": {"government": {"runtime": {"path": str(self.runtime_path),
                                                                   "sha256": self.runtime_sha},
                                                    "queueStateDirectory": str(self.queue)}}}
        self.card_path = self.inputs / "task-card.json"
        self.card_sha = write(self.card_path, b'{"synthetic":"public task card"}')
        card_binding = {"path": str(self.card_path), "sha256": self.card_sha}
        self.request["task"]["card"] = card_binding
        self.request["prompt"] = card_binding
        self.request["releasedInputs"].append(card_binding)
        self.request_raw = raw_json(self.request)
        write(self.request_path, self.request_raw)

        self._create_real_authority()
        self.invocation = self._invocation()

    def tearDown(self):
        self.temp.cleanup()

    def _create_real_authority(self):
        from dispatch import mechanical_pin, runtime_pins, execution_sha
        common = self.limits
        pin_sha = dispatch_digest(dispatch_encoded(mechanical_pin()))
        adoption_path = self.inputs / "profile-adoption.json"
        adoption = {"status": "approved", "trialId": self.request["trialId"],
                    "ledgerPath": str(self.ledger_path.resolve()),
                    "fromProfileSha256": profile_sha(LEGACY), "toProfileSha256": profile_sha(OBSERVED),
                    "limitsSha256": limits_sha(common), "decisionRef": "synthetic-role-bridge-test"}
        adoption_raw = dispatch_encoded(adoption)
        write(adoption_path, adoption_raw)
        protocol = {"status": "frozen", "mode": "mechanical", "commonLimits": common,
                    "runnerPinSha256": pin_sha, "runtimeSourceSha256": runtime_pins(),
                    "wrapperPythonSha256": dispatch_digest(Path(sys.executable).read_bytes()),
                    "measurementProfileId": OBSERVED,
                    "measurementProfileSha256": profile_sha(OBSERVED),
                    "profileAdoption": {"path": str(adoption_path),
                                        "sha256": dispatch_digest(adoption_raw)}}
        protocol_raw = dispatch_encoded(protocol)
        write(self.protocol_path, protocol_raw)
        grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics",
                 "mode": "mechanical", "trialId": self.request["trialId"],
                 "notBefore": time.time() - 1, "expiresAt": time.time() + 300,
                 "protocolSha256": dispatch_digest(protocol_raw), "profileSha256": dispatch_digest(dispatch_encoded(common)),
                 "measurementProfileId": OBSERVED,
                 "measurementProfileSha256": profile_sha(OBSERVED),
                 "runnerPinSha256": pin_sha, "ledgerPath": str(self.ledger_path),
                 "resultDirectory": str(self.root / "results"), "maxActorSessions": self.max_calls,
                 "maxSessionWallSeconds": 60, "retrospectiveTokenThreshold": 10000,
                 "authorizedRequests": [{"dispatchId": self.request["dispatchId"],
                     "executionSha256": execution_sha(self.request),
                     "initialRequestSha256": dispatch_digest(self.request_raw)}]}
        Path(grant["resultDirectory"]).mkdir()
        grant_raw = dispatch_encoded(grant)
        write(self.grant_path, grant_raw)
        self.authority = Authority(self.grant_path, dispatch_digest(grant_raw), self.protocol_path,
                                   dispatch_digest(protocol_raw))
        self.ledger = self.authority.ledger()
        self.ledger.reserve_dispatch(self.request["dispatchId"], execution_sha(self.request), self.request_raw,
                                    ["synthetic-outer"], "task-bridge", "task", self.max_calls)
        self.ledger.claim_dispatch(self.request["dispatchId"], "launching")

    def _rebind_real_authority_with_token_threshold(self, threshold):
        grant = json.loads(self.grant_path.read_bytes())
        grant["retrospectiveTokenThreshold"] = threshold
        grant_raw = dispatch_encoded(grant)
        write(self.grant_path, grant_raw)
        self.ledger_path.unlink()
        self.authority = Authority(self.grant_path, dispatch_digest(grant_raw), self.protocol_path,
                                   dispatch_digest(self.protocol_path.read_bytes()))
        self.ledger = self.authority.ledger()
        from dispatch import execution_sha
        self.ledger.reserve_dispatch(self.request["dispatchId"], execution_sha(self.request), self.request_raw,
                                    ["synthetic-outer"], "task-bridge", "task", self.max_calls)
        self.ledger.claim_dispatch(self.request["dispatchId"], "launching")

    @staticmethod
    def execution_sha(request):
        return sha(government_roles.encoded({key: value for key, value in request.items() if key != "operation"}))

    def _invocation(self, *, run_id="run-1", nonce="nonce-1", phase="execute", slot="root-writer", role="executor"):
        request = {"role": role, "sourceRevision": "commit-a", "modelDigest": "sha256:" + "1" * 64,
                   "modulePin": "sha256:" + "2" * 64, "projectionId": f"government/{phase}/{slot}",
                   "scopeIds": ["scope-1"], "policyIds": ["policy-1"],
                   "context": {"phase": phase}, "artifacts": []}
        value = {"apiVersion": government_roles.INVOCATION_API, "runId": run_id, "nonce": nonce,
                 "inputDigest": "sha256:" + sha(go_json(request)), "request": request}
        return raw_json(value)

    def _call(self, raw=None):
        return government_roles.run_role(raw or self.invocation, self.auth_raw, self.auth_sha,
                                         str(self.auth_path), self.request_raw, self.authority,
                                         expected_slot="root-writer",
                                         cwd=str(self.actor))

    def test_invocation_uses_source_wire_digest_and_returns_unchanged_response(self):
        invocation = government_roles.parse_invocation(self.invocation)
        self.assertTrue(invocation["inputDigest"].startswith("sha256:"))
        response_raw, receipt = self._call()
        response = json.loads(response_raw)
        self.assertEqual(response["runId"], "run-1")
        self.assertEqual(response["nonce"], "nonce-1")
        self.assertEqual(receipt["inputDigest"], invocation["inputDigest"])
        self.assertEqual(receipt["reportedInputPlusOutputTokens"], 18)
        self.assertIsNone(receipt["providerTurns"])
        self.assertEqual(len(self.marker_path.read_text().splitlines()), 1)
        attempts = self.ledger.snapshot()["attempts"]
        self.assertEqual(len(attempts), 2)  # existing outer dispatch + one pre-effect role reservation
        self.assertEqual(attempts[-1]["tokens"], 18)  # cached tokens are not added

    def test_duplicate_invocation_never_relaunches_delegate(self):
        self._call()
        marker_count = len(self.marker_path.read_text().splitlines())
        with self.assertRaisesRegex(ValueError, "replay"):
            self._call()
        self.assertEqual(len(self.marker_path.read_text().splitlines()), marker_count)
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 2)

    def test_existing_grant_session_ceiling_counts_outer_booking_and_nested_calls(self):
        self._call()
        self._call(self._invocation(run_id="run-2", nonce="nonce-2"))
        self._call(self._invocation(run_id="run-3", nonce="nonce-3"))
        with self.assertRaisesRegex(LimitReached, "grant session limit"):
            self._call(self._invocation(run_id="run-4", nonce="nonce-4"))
        self.assertEqual(len(self.marker_path.read_text().splitlines()), 3)
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 4)

    def test_coordinator_token_threshold_blocks_following_role_atomically(self):
        self._rebind_real_authority_with_token_threshold(18)
        self.assertLess(18, self.ledger.limits["trialProviderTokens"])
        prior_attempt = self.ledger.reserve_dispatch("prior-dispatch", sha(b"prior-request"), b"prior-request",
                                                     ["synthetic-prior"], "prior-task", "task", self.max_calls)
        self.ledger.claim_dispatch("prior-dispatch", "launching")
        self.ledger.complete_dispatch("prior-dispatch", {"status": "completed", "receipts": []}, 1, 1000)
        self.assertIsNotNone(prior_attempt)
        self._call()
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 3)
        self.assertLess(3, self.authority.grant["maxActorSessions"])
        next_invocation = self._invocation(run_id="run-after-threshold", nonce="nonce-after-threshold")
        with self.assertRaisesRegex(LimitReached, "retrospective token threshold"):
            self._call(next_invocation)
        self.assertEqual(len(self.marker_path.read_text().splitlines()), 1)
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 3)

    def test_delegate_timeout_is_capped_by_remaining_outer_request_wall(self):
        record = self.ledger.dispatch_record(self.request["dispatchId"])
        with self.ledger.transaction() as db:
            db.execute("UPDATE attempts SET start=? WHERE id=?",
                       (time.time() - self.request["wallSeconds"] + 0.8, record["attempt"]))
        observed = []

        def bounded_spy(argv, cwd, evidence, wall_seconds, **kwargs):
            observed.append(wall_seconds)
            output = Path(evidence)
            output.mkdir(parents=True)
            invocation = json.loads(kwargs["stdin"])
            request = invocation["request"]
            response = {"apiVersion": invocation["apiVersion"], "runId": invocation["runId"],
                        "nonce": invocation["nonce"], "role": request["role"],
                        "inputDigest": invocation["inputDigest"], "usage": {"source": "provider-reported",
                        "inputTokens": 1, "outputTokens": 1}}
            (output / "stdout.log").write_bytes(raw_json(response))
            (output / "stderr.log").write_bytes(b"")
            (output / "process.json").write_bytes(b"{}")
            return {"returnCode": 0}

        with patch.object(government_roles, "bounded", side_effect=bounded_spy):
            government_roles.run_role(self.invocation, self.auth_raw, self.auth_sha, str(self.auth_path),
                                      self.request_raw, self.authority, expected_slot="root-writer",
                                      cwd=str(self.actor))
        self.assertEqual(len(observed), 1)
        self.assertGreater(observed[0], 0)
        self.assertLessEqual(observed[0], 0.81)
        self.assertFalse(self.marker_path.exists())

    def test_changed_authorization_digest_fails_before_reservation_or_effect(self):
        with self.assertRaisesRegex(ValueError, "authorization digest"):
            government_roles.run_role(self.invocation, self.auth_raw, "f" * 64, str(self.auth_path),
                                      self.request_raw, self.authority,
                                      expected_slot="root-writer",
                                      cwd=str(self.actor))
        self.assertFalse(self.marker_path.exists())
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 1)

    def test_authorization_raw_must_match_runtime_pinned_path_bytes(self):
        changed = json.loads(self.auth_raw)
        changed["expiresAt"] += 1
        changed_raw = raw_json(changed)
        with self.assertRaisesRegex(ValueError, "path bytes differ"):
            government_roles.run_role(self.invocation, changed_raw, sha(changed_raw), str(self.auth_path),
                                      self.request_raw, self.authority,
                                      expected_slot="root-writer", cwd=str(self.actor))
        self.assertFalse(self.marker_path.exists())
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 1)

    def test_mismatched_native_phase_is_rejected_before_reservation(self):
        invalid = self._invocation(phase="vote", slot="root-writer", role="verifier")
        with self.assertRaisesRegex(ValueError, "role/phase"):
            self._call(invalid)
        self.assertFalse(self.marker_path.exists())
        self.assertEqual(len(self.ledger.snapshot()["attempts"]), 1)

    def test_recursive_role_inventory_and_vote_mapping_are_explicit(self):
        runtime = {"executor": {"slotId": "root", "command": "wrapper"},
                   "verifier": {"slotId": "review", "command": "wrapper"},
                   "recursion": {"areas": [{"area": {"name": "child"},
                                               "executor": {"slotId": "child-exec", "command": "wrapper"},
                                               "verifier": {"slotId": "child-review", "command": "wrapper"}}]},
                   "ressorts": [{"ressort": {"name": "budget"},
                                  "runner": {"slotId": "budget-vote", "command": "wrapper"}}]}
        roles = government.configured_roles(runtime)
        self.assertEqual([(role["slotId"], role["phase"], role["responseRole"]) for role in roles],
                         [("root", "execute", "executor"), ("review", "review", "verifier"),
                          ("child-exec", "execute", "executor"), ("child-review", "review", "verifier"),
                          ("budget-vote", "vote", "verifier")])

    def test_native_cli_remains_closed_without_acyclic_authority_bootstrap(self):
        with self.assertRaisesRegex(RuntimeError, "not integrated"):
            government_roles.main([])


if __name__ == "__main__":
    unittest.main(verbosity=2)
