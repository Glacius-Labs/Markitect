"""Real-boundary offline R3 Classic Request/Grant/bootstrap preflight.

Creates only a disposable pinned fixture and authority/ledger tree. Exercises
production validators without patches and finalizes before any product process.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest

RUNTIME = Path(__file__).resolve().parent
sys.path.insert(0, str(RUNTIME))
import classic
import classic_integration
import dispatch
import government_roles
import native_controller
from measurement_profile import LEGACY

R3_ROOT = Path(os.environ.get(
    "MARKITECT_R3_NATIVE_ROOT",
    r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r3"))
ORIGINAL_ROOT = Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008")
ORIGINAL_GRANT = ORIGINAL_ROOT / "released-native-grant.json"
ORIGINAL_SNAPSHOT = ORIGINAL_ROOT / "coordinator-authority-snapshot.json"
R3_GRANT = R3_ROOT / "released-r3-a1-grant.json"
R3_SNAPSHOT = R3_ROOT / "coordinator-r3-a1-snapshot.json"
EXPECTED_ORIGINAL_GRANT_SHA = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"
EXPECTED_ORIGINAL_SNAPSHOT_SHA = "04484d4220dd7383af1d2675a95384554380dccc5a14ee8fc5908c3a474ede58"
EXPECTED_R3_GRANT_SHA = "b45a048923102feaa2040a769281cc2c258d66be09c2ce3561238b4b74c3f932"
EXPECTED_R3_SNAPSHOT_SHA = "29595284a6e4202129d9f014aee6e1fec545d19bdfe028411687d41ec6b8fa17"
R3_KEY = "native-s1-contract-corrected-integration-20261008-r3"


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def encoded(value) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


class R3ClassicPreflightTests(unittest.TestCase):
    def test_actual_r3_grant_and_classic_request_reach_bootstrap_without_product_start(self):
        # These are immutable operator-provided source envelopes, not test-generated approvals.
        for path, expected in ((ORIGINAL_GRANT, EXPECTED_ORIGINAL_GRANT_SHA),
                               (ORIGINAL_SNAPSHOT, EXPECTED_ORIGINAL_SNAPSHOT_SHA),
                               (R3_GRANT, EXPECTED_R3_GRANT_SHA),
                               (R3_SNAPSHOT, EXPECTED_R3_SNAPSHOT_SHA)):
            self.assertTrue(path.is_file(), f"required immutable R3 source input missing: {path}")
            self.assertEqual(digest(path.read_bytes()), expected, f"source input changed: {path}")

        with tempfile.TemporaryDirectory(prefix="r3-classic-real-preflight-") as temporary:
            root = Path(temporary).resolve()
            case = root / "case"
            request_path = case / "released" / "request.json"
            ledger_path = root / "authority" / "trial.sqlite"
            results = root / "results"
            results.mkdir()
            evidence = root / "evidence"
            trial_id = "native-fixture-classic-20261008-r3"
            dispatch_id = "classic-native-contract-corrected-r3"
            task_id = classic_integration.TASK_ID
            role_evidence = root / "role-evidence"

            prepared = classic_integration.prepare_fixture(
                classic_integration.PACKET, case,
                reports_directory=case / "results",
                runtime_path=case / "released" / "runtime.json",
                authorization_path=case / "released" / "role-auth.json",
                controller_state_directory=case / "controller-state",
                request_path=request_path,
                ledger_path=ledger_path,
                role_evidence_directory=role_evidence,
                trial_id=trial_id, dispatch_id=dispatch_id, task_id=task_id,
                expires_at=time.time() + 5400, authorization_status="approved")
            self.assertFalse(prepared["nativeOrActorProcessesStarted"])
            self.assertEqual(prepared["productSource"], classic.EXPECTED_SOURCE)
            self.assertEqual(prepared["heldSource"], classic.EXPECTED_HELD_SOURCE)

            card_path = root / "released" / "task-card.md"
            prompt_path = root / "released" / "prompt.md"
            card_path.parent.mkdir(parents=True, exist_ok=True)
            card_path.write_text("R3 offline Classic pinned-fixture preflight; no product execution.\n", encoding="utf-8")
            prompt_path.write_text("Check the exact pinned Classic contract and report only offline validation.\n", encoding="utf-8")
            runtime_path = Path(prepared["runtimePath"]).resolve(strict=True)
            authorization_path = Path(prepared["roleAuthorizationPath"]).resolve(strict=True)
            diagnostics_path = root / "released" / "wrapper-diagnostics.json"
            r3_binding = {"path": str(R3_GRANT.resolve()), "sha256": EXPECTED_R3_GRANT_SHA,
                          "sourceKey": R3_KEY}
            diagnostics = {"arm": "classic", "correction": r3_binding,
                           "requestPath": str(request_path.resolve())}
            diagnostics_raw = encoded(diagnostics)
            diagnostics_path.write_bytes(diagnostics_raw)
            authorization = json.loads(authorization_path.read_bytes())
            authorization["maxCalls"] = 6
            authorization["diagnostics"] = {"path": str(diagnostics_path.resolve()),
                                             "sha256": digest(diagnostics_raw)}
            authorization_raw = encoded(authorization) + b"\n"
            authorization_path.write_bytes(authorization_raw)
            runtime = json.loads(runtime_path.read_bytes())
            wrapper = Path(government_roles.__file__).resolve(strict=True)
            wrapper_args = government_roles.wrapper_arguments(
                str(wrapper), str(authorization_path), digest(authorization_raw),
                authorization["roleEvidenceDirectory"])
            wrapper_args.extend(["--diagnostics-config", str(diagnostics_path.resolve()),
                                 "--diagnostics-sha256", digest(diagnostics_raw)])
            wrapper_pins = runtime["executor"]["runtimeFiles"]
            auth_pin = classic_integration._runtime_file(authorization_path)
            diag_pin = classic_integration._runtime_file(diagnostics_path)
            wrapper_pins = [item for item in wrapper_pins
                            if Path(item["path"]).resolve() not in {authorization_path, diagnostics_path.resolve()}]
            wrapper_pins.extend([auth_pin, diag_pin])
            for role_name in ("executor", "verifier"):
                runtime[role_name]["args"] = wrapper_args
                runtime[role_name]["runtimeFiles"] = wrapper_pins
            runtime_path.write_bytes(encoded(runtime) + b"\n")
            actor = Path(prepared["fixtureRepo"]).resolve(strict=True)
            packet = classic.inspect_packet(classic_integration.PACKET)
            packet_path = Path(packet["packetPath"]).resolve(strict=True)
            executable = Path(packet["binary"]["path"]).resolve(strict=True)
            project_config = actor / "examples/canonical-projection/canonical.yaml"
            original_doc = json.loads(ORIGINAL_GRANT.read_bytes())
            original_snapshot_path = Path(original_doc["sourceCoordinationPath"]).resolve(strict=True)
            released_files = [
                card_path, prompt_path, runtime_path, authorization_path, diagnostics_path,
                ORIGINAL_GRANT.resolve(strict=True), original_snapshot_path,
                R3_GRANT.resolve(strict=True), R3_SNAPSHOT.resolve(strict=True),
            ]
            released_inputs = [{"path": str(path), "sha256": digest(path.read_bytes())}
                               for path in released_files]

            common = json.loads((RUNTIME.parent / "public/resource-proposal.json").read_bytes())["commonLimits"]
            pin = dispatch.digest(dispatch.encoded(dispatch.mechanical_pin()))
            protocol = {"schemaVersion": 1, "status": "frozen", "mode": "mechanical",
                        "commonLimits": common, "runnerPinSha256": pin,
                        "runtimeSourceSha256": dispatch.runtime_pins(),
                        "wrapperPythonSha256": digest(Path(sys.executable).resolve().read_bytes())}
            protocol_path = root / "authority" / "protocol.json"
            protocol_path.parent.mkdir(parents=True, exist_ok=True)
            protocol_raw = encoded(protocol)
            protocol_path.write_bytes(protocol_raw)
            request = {
                "schemaVersion": 1, "mode": "mechanical", "operation": "run_task",
                "trialId": trial_id, "dispatchId": dispatch_id, "arm": "classic",
                "condition": "brownfield", "actorRepository": str(actor),
                "evidenceDirectory": str(evidence), "limits": common, "wallSeconds": 180,
                "task": {"id": task_id, "card": {"path": str(card_path), "sha256": digest(card_path.read_bytes())}},
                "purpose": "task", "prompt": {"path": str(prompt_path), "sha256": digest(prompt_path.read_bytes())},
                "releasedInputs": released_inputs,
                "baseCommit": prepared["fixtureSourceCommit"],
                "nativeFixtureGrant": {"path": str(ORIGINAL_GRANT.resolve()),
                                        "sha256": EXPECTED_ORIGINAL_GRANT_SHA,
                                        "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY},
                "nativeFixtureR3Grant": r3_binding,
                "product": {"classic": {
                    "controllerActions": list(native_controller.CLASSIC_ACTIONS),
                    "packet": {"path": str(packet_path),
                               "manifestSha256": digest((packet_path / "checksums.sha256").read_bytes())},
                    "executable": {"path": str(executable), "sha256": digest(executable.read_bytes()),
                                   "sourceCommit": classic.EXPECTED_SOURCE,
                                   "heldSourceCommit": classic.EXPECTED_HELD_SOURCE},
                    "projectConfig": {"path": "examples/canonical-projection/canonical.yaml",
                                      "sha256": digest(project_config.read_bytes())},
                    "runtime": {"path": str(runtime_path), "sha256": digest(runtime_path.read_bytes())},
                    "roleAuthorization": {"path": str(authorization_path),
                                          "sha256": digest(authorization_path.read_bytes())}}}}
            request_raw = encoded(request)
            request_path.write_bytes(request_raw)

            # Construct a real Coordinator Grant/Protocol/Authority for this disposable test
            # identity; all source-grant evidence and product pins remain the immutable inputs above.
            authorized = [{"dispatchId": dispatch_id, "executionSha256": dispatch.execution_sha(request),
                           "initialRequestSha256": digest(request_raw)}]
            grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics",
                     "mode": "mechanical", "trialId": trial_id, "notBefore": time.time() - 1,
                     "expiresAt": time.time() + 5500, "protocolSha256": digest(protocol_raw),
                     "profileSha256": digest(encoded(common)), "runnerPinSha256": pin,
                     "ledgerPath": str(ledger_path), "resultDirectory": str(results),
                     "maxActorSessions": 6, "maxSessionWallSeconds": 180,
                     "retrospectiveTokenThreshold": 10000, "authorizedRequests": authorized}
            grant_path = root / "authority" / "grant.json"
            grant_raw = encoded(grant)
            grant_path.write_bytes(grant_raw)
            authority = dispatch.Authority(grant_path, digest(grant_raw), protocol_path, digest(protocol_raw))
            self.assertEqual(authority.validate(request_raw)[0]["nativeFixtureR3Grant"]["sourceKey"], R3_KEY)

            def rejected_request_variant(label, variant, message):
                raw = encoded(variant)
                request_path.write_bytes(raw)
                variant_grant = dict(grant)
                variant_grant["authorizedRequests"] = [{
                    "dispatchId": variant["dispatchId"],
                    "executionSha256": dispatch.execution_sha(variant),
                    "initialRequestSha256": digest(raw),
                }]
                variant_grant_path = root / "authority" / f"grant-{label}.json"
                variant_grant_raw = encoded(variant_grant)
                variant_grant_path.write_bytes(variant_grant_raw)
                variant_authority = dispatch.Authority(
                    variant_grant_path, digest(variant_grant_raw), protocol_path, digest(protocol_raw))
                if label == "missing-r3":
                    validated_variant, variant_captured = variant_authority.validate(raw)
                    with self.assertRaisesRegex(ValueError, "R3 dispatch requires its exact R3 grant"):
                        native_controller.validate_native_fixture_grant(validated_variant, variant_captured)
                with self.assertRaisesRegex(ValueError, message):
                    native_controller.begin_classic_controller(request_path, variant_authority, request_raw=raw)
                self.assertFalse(ledger_path.exists(), f"{label} rejection must precede ledger claim")
                request_path.write_bytes(request_raw)

            missing_r3 = json.loads(encoded(request))
            missing_r3.pop("nativeFixtureR3Grant")
            rejected_request_variant("missing-r3", missing_r3, "R3 dispatch requires its exact R3 grant")

            mixed_r2 = json.loads(encoded(request))
            mixed_r2["nativeFixtureCorrection"] = {
                "path": str(root / "released" / "synthetic-r2-correction.json"),
                "sha256": "a" * 64, "sourceKey": "native-s1-corrected-integration-20261008-r2"}
            rejected_request_variant("mixed-r2", mixed_r2, "cannot reuse the R2 correction binding")

            bad_auth_path = root / "released" / "role-auth-wrong-scope.json"
            bad_auth = json.loads(authorization_path.read_bytes())
            bad_auth["slots"][0]["scopeIds"] = ["unbound-scope"]
            bad_auth_raw = encoded(bad_auth) + b"\n"
            bad_auth_path.write_bytes(bad_auth_raw)
            wrong_scope = json.loads(encoded(request))
            role_binding = wrong_scope["product"]["classic"]["roleAuthorization"]
            role_binding.update(path=str(bad_auth_path.resolve()), sha256=digest(bad_auth_raw))
            wrong_scope["releasedInputs"] = [item for item in wrong_scope["releasedInputs"]
                                               if Path(item["path"]).resolve() != authorization_path]
            wrong_scope["releasedInputs"].append({"path": str(bad_auth_path.resolve()),
                                                   "sha256": digest(bad_auth_raw)})
            with self.assertRaisesRegex(ValueError, "exact native Projection subjects"):
                classic_integration.bind_request(wrong_scope)
            self.assertFalse(ledger_path.exists(), "scope mismatch rejection must precede ledger claim")
            bad_auth_path.unlink()

            # Direct calls make boundary ownership visible; begin then repeats the actual,
            # uncapped preflight path through Authority, Classic binder, R3 validator and bootstrap.
            checked_request, captured = authority.validate(request_raw)
            bound = classic_integration.bind_request(checked_request, request_path=request_path)
            self.assertEqual(bound["baseCommit"], request["baseCommit"])
            r3_check = native_controller.validate_native_fixture_grant(checked_request, captured, bound)
            self.assertEqual(r3_check["grantKey"], R3_KEY)
            self.assertEqual(r3_check["maxRoleStarts"], 6)
            self.assertEqual(r3_check["maxRoleParallel"], 2)

            session = native_controller.begin_classic_controller(
                request_path, authority, request_raw=request_raw, captured=captured)
            try:
                self.assertIsInstance(session.context.authority, dispatch.Authority)
                self.assertEqual(session.context.request_raw, request_raw)
                self.assertEqual(session.bound["baseCommit"], request["baseCommit"])
                self.assertEqual(session.context.request["nativeFixtureR3Grant"]["sourceKey"], R3_KEY)
                context_bounds = session.context.native_fixture_bounds
                self.assertEqual(context_bounds["grantKey"], R3_KEY)
                self.assertEqual(context_bounds["maxRoleStarts"], 6)
                self.assertEqual(context_bounds["maxDeterministicDelegates"], 6)
                self.assertEqual(context_bounds["maxNativeStarts"], 5)
                self.assertEqual(context_bounds["maxReservedSessionSeconds"], 750)
                self.assertEqual(context_bounds["r3Grant"]["grantKey"], R3_KEY)
                self.assertEqual(session.captures, [])
                with session.ledger.transaction() as db:
                    self.assertEqual(db.execute("SELECT COUNT(*) FROM controller_processes").fetchone()[0], 0)
                final = native_controller.finalize_classic_controller(session)
                self.assertEqual(final["status"], "incomplete")
                self.assertEqual(final["classic"]["semanticAcceptance"], False)
                self.assertEqual(final["candidateCommit"], None)
                self.assertEqual(final["controllerProcesses"], [])
                self.assertTrue(any("missing persisted native capture" in item for item in final["classic"]["errors"]))
            finally:
                if not session.finished:
                    session.ledger.finish_controller_dispatch(
                        dispatch_id,
                        {"status": "incomplete", "candidateCommit": None,
                         "gaps": ["test stopped before any Classic product process"]},
                        {"controllerProcesses": [], "productProcessStarted": False})
                    session.finished = True

            self.assertTrue((evidence / "controller-bootstrap.json").is_file())
            self.assertEqual(len(list((evidence / "classic-native").glob("*"))), 0)
            self.assertTrue((results / f"{dispatch_id}.json").is_file())
            # All mutable outputs, including the SQLite ledger, were confined to TemporaryDirectory.
            self.assertEqual(authority.ledger_path, ledger_path.resolve())
            self.assertTrue(str(ledger_path).startswith(str(root)))


if __name__ == "__main__":
    unittest.main()
