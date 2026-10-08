"""Cross-arm synthetic contract chain based on the frozen R2 slot declarations.

No native product or delegate is launched. ``government_roles.bounded`` is a
test double that records the already-persisted reservation and writes fixed
process receipts so the real response/failure translation can be inspected.
"""
import copy
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import time
import types
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import classic_integration
import government
import government_integration
import government_roles
import native_controller
from dispatch import Authority, digest, encoded, execution_sha, mechanical_pin, runtime_pins


R2 = Path(__file__).parents[1] / "evidence/native-integration/run-2/external-snapshots"
COMMON_LIMITS = {
    "taskWallSeconds": 1200, "trialWallSeconds": 7200,
    "taskActorCalls": 12, "trialActorCalls": 72,
    "taskProviderTurns": 80, "trialProviderTurns": 480,
    "taskProviderTokens": 120000, "trialProviderTokens": 720000,
    "maxParallelActors": 4, "maxTransportRetriesPerCall": 1,
    "maxSemanticRepairRoundsPerTask": 2, "trialActiveHumanSeconds": 600,
    "newPurchases": False,
}


def raw_json(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def dto_json(value):
    # Match encoding/json struct field order for agentexec.Request/Invocation.
    return json.dumps(value, separators=(",", ":"), ensure_ascii=False).encode()


def file_pin(path):
    path = Path(path).resolve(strict=True)
    mode = stat.S_IMODE(path.stat().st_mode)
    mode_text = ("0644" if os.name == "nt" and mode & stat.S_IWUSR else
                 "0444" if os.name == "nt" else f"{mode:04o}")
    return {"path": str(path), "mode": mode_text,
            "digest": "sha256:" + digest(path.read_bytes())}


class NativeAdapterContractTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="native-adapter-contracts-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()

    def _arm_fixture(self, arm, *, commandless=False):
        root = self.root / arm
        released = R2 / arm / "released"
        frozen_runtime = json.loads((released / "runtime.json").read_bytes())
        frozen_auth = json.loads((released / "role-auth.json").read_bytes())
        frozen_request = json.loads((released / "request.json").read_bytes())
        old_slots = frozen_auth["slots"]
        self.assertEqual(len(old_slots), 3)
        if arm == "government":
            delegate_path = Path(old_slots[0]["delegate"]["argv"][1]).resolve(strict=True)
            generated = government_integration.build_role_authorization(
                trial_id=frozen_request["trialId"], dispatch_id=frozen_request["dispatchId"],
                request_path=root / "request.json", ledger_path=root / "ledger.sqlite",
                runtime_path=root / "runtime.json", role_evidence_directory=root / "role-evidence",
                expires_at=time.time() + 1800, python_executable=sys.executable,
                delegate_path=delegate_path, status="approved",
                fixture_authorization=government_integration.FIXTURE_AUTHORIZATION)
        else:
            packet = frozen_request["product"]["classic"]["packet"]["path"]
            generated = classic_integration.build_role_authorization(
                trial_id=frozen_request["trialId"], dispatch_id=frozen_request["dispatchId"],
                task_id=frozen_request["task"]["id"], request_path=root / "request.json",
                ledger_path=root / "ledger.sqlite", runtime_path=root / "runtime.json",
                role_evidence_directory=root / "role-evidence", expires_at=time.time() + 1800,
                packet_path=packet, python_executable=sys.executable, status="approved")

        self.assertEqual([x["slotId"] for x in generated["slots"]], [x["slotId"] for x in old_slots])
        # The R2 role-auth snapshot predates the command field; this is the
        # source-derived corrected producer output under test.
        self.assertTrue(all("command" not in x["delegate"] for x in old_slots))
        self.assertTrue(all(x["delegate"]["command"] == x["delegate"]["argv"][0]
                            for x in generated["slots"]))

        root.mkdir(parents=True)
        (root / "actor").mkdir()
        (root / "outer-evidence").mkdir()
        (root / "role-evidence").mkdir()
        (root / "state").mkdir()
        request_path = root / "request.json"
        runtime_path = root / "runtime.json"
        auth_path = root / "role-auth.json"
        diagnostics_path = root / "wrapper-diagnostics-config.json"
        correction_path = root / "correction-grant.json"
        correction_source = R2 / "released-correction-grant.json"
        correction_raw = correction_source.read_bytes()
        correction_path.write_bytes(correction_raw)
        correction_value = json.loads(correction_raw)
        correction_binding = {"path": str(correction_path), "sha256": digest(correction_raw),
                              "sourceKey": correction_value["grant"]["key"]}
        diagnostics = {"arm": arm, "correction": correction_binding,
                       "requestPath": str(request_path)}
        diagnostics_raw = raw_json(diagnostics)
        diagnostics_path.write_bytes(diagnostics_raw)
        diagnostics_binding = {"path": str(diagnostics_path), "sha256": digest(diagnostics_raw)}

        auth = copy.deepcopy(generated)
        auth["maxCalls"] = 6
        if commandless:
            # Reproduce the frozen R2 slot wire shape: its argv[0] was pinned,
            # but the separate command binding had not yet been added.
            for role in auth["slots"]:
                role["delegate"].pop("command", None)
        auth["diagnostics"] = diagnostics_binding
        auth_raw = raw_json(auth)
        auth_path.write_bytes(auth_raw)
        auth_sha = digest(auth_raw)

        runtime = copy.deepcopy(frozen_runtime)
        bridge_names = ["government_roles.py", "native_controller.py", "government.py",
                        "dispatch.py", "identity_probe.py", "ledger.py", "process.py",
                        "runner.py", "measurement_profile.py", "context_allocation.py",
                        "context_tools_allocation.py", "diagnostic_history.py",
                        "native_fixture_budget.py"]
        bridge_paths = [Path(government_roles.__file__).with_name(name) for name in bridge_names]
        if arm == "classic":
            bridge_paths.extend([Path(classic_integration.__file__),
                                 Path(__file__).with_name("classic.py"),
                                 Path(__file__).with_name("classic-pin.json")])
        runtime_files = [file_pin(path) for path in bridge_paths]
        runtime_files.extend([file_pin(sys.executable), file_pin(auth_path), file_pin(diagnostics_path)])
        for role in auth["slots"]:
            runtime_files.extend(file_pin(item["path"]) for item in role["delegate"]["runtimeFiles"])
        by_path = {item["path"]: item for item in runtime_files}
        runtime_files = [by_path[path] for path in sorted(by_path)]

        def configure_runner(runner, role):
            runner.update({"command": role["wrapper"]["command"],
                           "args": government_roles.wrapper_arguments(
                               str(Path(government_roles.__file__).resolve()), str(auth_path), auth_sha,
                               str(root / "role-evidence"),
                               role["slotId"] if arm == "government" else None) +
                                  ["--diagnostics-config", str(diagnostics_path),
                                   "--diagnostics-sha256", diagnostics_binding["sha256"]],
                           "runtimeFiles": runtime_files,
                           "model": role["delegate"]["model"],
                           "modelOptions": role["delegate"]["modelOptions"],
                           "providerVersion": role["delegate"]["providerVersion"]})
            if arm == "government":
                runner["delegate"] = copy.deepcopy(role["delegate"])
        if arm == "government":
            runtime["executor"]["command"] = sys.executable
            for role in auth["slots"]:
                if role["phase"] == "execute":
                    configure_runner(runtime["executor"], role)
                elif role["phase"] == "review":
                    configure_runner(runtime["verifier"], role)
                else:
                    configure_runner(runtime["ressorts"][0]["runner"], role)
        else:
            execute = next(x for x in auth["slots"] if x["phase"] == "execute")
            review = next(x for x in auth["slots"] if x["slotId"].endswith("commerce-dotnet"))
            configure_runner(runtime["executor"], execute)
            configure_runner(runtime["verifier"], review)
            runtime["executor"]["delegate"] = copy.deepcopy(execute["delegate"])
            runtime["verifier"]["delegate"] = copy.deepcopy(review["delegate"])
        runtime_raw = raw_json(runtime)
        runtime_path.write_bytes(runtime_raw)

        request = copy.deepcopy(frozen_request)
        request.update({"actorRepository": str(root / "actor"),
                        "evidenceDirectory": str(root / "outer-evidence"),
                        "nativeFixtureCorrection": correction_binding,
                        "releasedInputs": []})
        card_path = root / "task-card.txt"
        card_raw = b"Source-derived synthetic task card for the offline R3 role contract.\n"
        card_path.write_bytes(card_raw)
        card_binding = {"path": str(card_path), "sha256": digest(card_raw)}
        request["task"]["card"] = card_binding
        request["prompt"] = card_binding
        product = request["product"][arm]
        product["runtime"] = {"path": str(runtime_path), "sha256": digest(runtime_raw)}
        product["roleAuthorization"] = {"path": str(auth_path), "sha256": auth_sha}
        released = ((runtime_path, runtime_raw), (auth_path, auth_raw),
                    (diagnostics_path, diagnostics_raw), (correction_path, correction_raw),
                    (card_path, card_raw))
        for path, value in released:
            request["releasedInputs"].append({"path": str(path), "sha256": digest(value)})
        request_raw = raw_json(request)
        request_path.write_bytes(request_raw)
        captured = {str(path): value for path, value in released}

        ledger_path = root / "ledger.sqlite"
        result_directory = root / "results"
        result_directory.mkdir()
        protocol_path, grant_path = root / "protocol.json", root / "grant.json"
        pin_sha = digest(encoded(mechanical_pin()))
        protocol = {"status": "frozen", "mode": "mechanical", "commonLimits": COMMON_LIMITS,
                    "runnerPinSha256": pin_sha, "runtimeSourceSha256": runtime_pins(),
                    "wrapperPythonSha256": digest(Path(sys.executable).read_bytes())}
        protocol_raw = encoded(protocol)
        protocol_path.write_bytes(protocol_raw)
        grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics",
                 "mode": "mechanical", "trialId": request["trialId"],
                 "notBefore": time.time() - 1, "expiresAt": time.time() + 1800,
                 "protocolSha256": digest(protocol_raw), "profileSha256": digest(encoded(COMMON_LIMITS)),
                 "runnerPinSha256": pin_sha, "ledgerPath": str(ledger_path),
                 "resultDirectory": str(result_directory), "maxActorSessions": 10,
                 "maxSessionWallSeconds": 180, "retrospectiveTokenThreshold": 10000,
                 "authorizedRequests": [{"dispatchId": request["dispatchId"],
                     "executionSha256": execution_sha(request), "initialRequestSha256": digest(request_raw)}]}
        grant_raw = encoded(grant)
        grant_path.write_bytes(grant_raw)
        authority = Authority(grant_path, digest(grant_raw), protocol_path, digest(protocol_raw))
        request, captured = authority.validate(request_raw)
        ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
        ledger.reserve_controller_dispatch(request["dispatchId"], execution_sha(request),
                                          request_raw, ["synthetic-controller-plan"],
                                          request["task"]["id"], "task", 10)
        self.assertTrue(ledger.claim_dispatch(request["dispatchId"], "launching"))
        bundle_path, bundle_sha = native_controller.write_bundle(
            root / "synthetic-controller-bootstrap.json", request_path=request_path,
            request_raw=request_raw, authority=authority, authorization_path=auth_path,
            authorization_sha256=auth_sha, allow_live=False)
        bounds = {"maxRoleStarts": 6, "maxRoleParallel": 2, "maxRoleProcessSeconds": 1200,
                  "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY, "product": arm.title()}
        with patch.object(native_controller, "validate_native_fixture_grant", return_value=bounds), \
             patch.object(classic_integration, "bind_request", return_value={"syntheticBound": True}):
            controller_context = native_controller.load_context(
                env={native_controller.BOOTSTRAP_PATH_ENV: str(bundle_path),
                     native_controller.BOOTSTRAP_SHA_ENV: bundle_sha})
        return {"arm": arm, "root": root, "request": request, "requestRaw": request_raw,
                "auth": auth, "authRaw": auth_raw, "authSha": auth_sha, "authPath": auth_path,
                "runtime": runtime, "runtimeRaw": runtime_raw, "runtimePath": runtime_path,
                "captured": captured, "diagnosticsPath": diagnostics_path,
                "authority": authority, "ledger": ledger, "controllerContext": controller_context,
                "frozenSlots": old_slots}

    @staticmethod
    def _invocation(arm, role, index, slot):
        projection = slot.get("projectionId") or f"government/{slot['phase']}/{slot['slotId']}"
        frozen_request = json.loads((R2 / arm / "released/request.json").read_bytes())
        if arm == "government":
            report_path = next((R2 / "government/results/run-state").glob("*/report.json"))
            report = json.loads(report_path.read_bytes())
            if slot["phase"] == "execute":
                actor_path = report_path.parent / "actor-0001.json"
                scope_ids = json.loads(actor_path.read_bytes())["scopes"]
            elif slot["phase"] == "review":
                work = report["plan"]["work"][0]
                identities = [work["area"], *work["subjects"]]
                scope_ids = [json.dumps([item[key] for key in
                    ("apiVersion", "kind", "namespace", "name")], separators=(",", ":"))
                    for item in identities]
            else:
                ressort = report["cabinet"][0]["ressort"]
                scope_ids = [json.dumps([ressort[key] for key in
                    ("apiVersion", "kind", "namespace", "name")], separators=(",", ":"))]
            context = {"phase": slot["phase"], "synthetic": True}
        else:
            scope_ids = list(slot["scopeIds"])
            definitions = []
            for identity_json in classic_integration.PROJECTION_SUBJECTS[projection]:
                api, kind, namespace, name = json.loads(identity_json)
                definitions.append({"apiVersion": api, "kind": kind,
                                    "metadata": {"namespace": namespace, "name": name}})
            context = {"model": {"projectionId": projection,
                                 "scopeIds": list(scope_ids),
                                 "definitions": definitions}}
        request = {"role": slot["responseRole"], "sourceRevision": frozen_request["baseCommit"],
                   "modelDigest": "sha256:" + "1" * 64, "modulePin": "sha256:" + "2" * 64,
                   "projectionId": projection, "scopeIds": scope_ids,
                   "policyIds": [], "context": context, "artifacts": []}
        invocation = {"apiVersion": government_roles.INVOCATION_API,
                      "runId": f"r3-{arm}-{index}", "nonce": f"synthetic-nonce-{arm}-{index}",
                      "inputDigest": "sha256:" + digest(dto_json(request)), "request": request}
        return raw_json(invocation)

    @staticmethod
    def _load_role_context(fixture, invocation_raw):
        context = fixture["controllerContext"]
        return native_controller.load_context(
            env={native_controller.BOOTSTRAP_PATH_ENV: str(context.bootstrap_path),
                 native_controller.BOOTSTRAP_SHA_ENV: context.bootstrap_sha256},
            invocation_raw=invocation_raw)

    def test_all_six_frozen_r2_roles_complete_the_offline_contract_chain(self):
        completed = []
        for arm in ("government", "classic"):
            with self.subTest(arm=arm):
                fixture = self._arm_fixture(arm)
                request, raw, auth_path = fixture["request"], fixture["requestRaw"], fixture["authPath"]
                authority, captured = fixture["authority"], fixture["captured"]
                # Static preflight and role resolution use the actual arm-specific
                # resolver. A source-derived synthetic grant/protocol binds this
                # temporary Request; actual Authority, bundle and load_context code
                # validate the offline bootstrap without launching a product.
                government_roles.preflight_authorization(
                    request, raw, authority, captured, str(auth_path),
                    fixture["authRaw"], fixture["authSha"])
                configured = (government.configured_roles(fixture["runtime"]) if arm == "government" else
                              classic_integration.configured_roles(fixture["runtime"]))
                self.assertEqual([role["slotId"] for role in configured],
                                 [slot["slotId"] for slot in fixture["auth"]["slots"]])
                role_resolver = classic_integration.resolve_role if arm == "classic" else None
                planned = []

                def fake_bounded(argv, cwd, evidence, timeout, **kwargs):
                    invocation = json.loads(kwargs["stdin"])
                    with fixture["ledger"].transaction() as db:
                        reserved = db.execute("SELECT status,attempt_id FROM government_role_calls "
                                               "WHERE run_id=?", (invocation["runId"],)).fetchone()
                        self.assertIsNotNone(reserved, "reservation must precede planned delegate effect")
                        self.assertEqual(reserved[0], "reserved")
                    planned.append((argv, invocation["runId"], timeout))
                    output = Path(evidence)
                    output.mkdir(parents=True, exist_ok=False)
                    failed = "failure" in invocation["runId"]
                    result = None
                    if not failed:
                        result = {"apiVersion": invocation["apiVersion"], "runId": invocation["runId"],
                                  "nonce": invocation["nonce"], "role": invocation["request"]["role"],
                                  "inputDigest": invocation["inputDigest"], "outcome": "proposed",
                                  "candidateFiles": [], "evidenceRefs": [], "verifierObservations": [],
                                  "uncertainty": []}
                    (output / "stdout.log").write_bytes(raw_json(result) if result else b"")
                    (output / "stderr.log").write_bytes(b"synthetic delegate receipt; no delegate ran\n")
                    (output / "process.json").write_bytes(raw_json({"argv": argv, "returnCode": 19 if failed else 0,
                        "wallSeconds": 0.01, "stopReason": None, "processTreeControl": "mocked-bounded"}))
                    return {"returnCode": 19 if failed else 0, "wallSeconds": 0.01,
                            "stopReason": None, "processTreeControl": "mocked-bounded"}

                bounds = {"maxRoleStarts": 6, "maxRoleParallel": 2,
                          "maxRoleProcessSeconds": 1200,
                          "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY,
                          "product": arm.title()}
                with patch.object(native_controller, "validate_native_fixture_grant", return_value=bounds), \
                     patch.object(classic_integration, "bind_request", return_value={"syntheticBound": True}), \
                     patch.object(government_roles, "bounded", side_effect=fake_bounded) as bounded:
                    # A source-shaped but altered native identity must fail in
                    # the resolver before a reservation or planned delegate call.
                    first_slot = fixture["auth"]["slots"][0]
                    altered = json.loads(self._invocation(arm, first_slot["responseRole"], "mutation", first_slot))
                    if arm == "government":
                        altered["request"]["projectionId"] = "government/review/unauthorized"
                    else:
                        altered["request"]["scopeIds"][0] = "[\"commerce.example.org/v1\",\"Handler\",\"commerce\",\"forged\"]"
                    altered["inputDigest"] = "sha256:" + digest(dto_json(altered["request"]))
                    with self.assertRaises(ValueError):
                        government_roles.run_role(
                            raw_json(altered), fixture["authRaw"], fixture["authSha"],
                            str(fixture["authPath"]), fixture["requestRaw"], authority,
                            expected_slot=first_slot["slotId"],
                            controller_context=self._load_role_context(fixture, raw_json(altered)),
                            role_resolver=role_resolver, cwd=request["actorRepository"])
                    self.assertEqual(bounded.call_count, 0)
                    with fixture["ledger"].transaction() as db:
                        tables = {row[0] for row in db.execute(
                            "SELECT name FROM sqlite_master WHERE type='table'").fetchall()}
                        role_rows = (db.execute("SELECT COUNT(*) FROM government_role_calls").fetchone()[0]
                                     if "government_role_calls" in tables else 0)
                        self.assertEqual(role_rows, 0)

                    for index, slot in enumerate(fixture["auth"]["slots"]):
                        invocation_raw = self._invocation(arm, slot["responseRole"], index, slot)
                        role_context = self._load_role_context(fixture, invocation_raw)
                        response, receipt = government_roles.run_role(
                            invocation_raw, fixture["authRaw"], fixture["authSha"],
                            str(fixture["authPath"]), fixture["requestRaw"], authority,
                            expected_slot=slot["slotId"], controller_context=role_context,
                            role_resolver=role_resolver, cwd=request["actorRepository"])
                        value = json.loads(response)
                        self.assertEqual(receipt["status"], "protocol-echo-valid")
                        self.assertEqual(value["runId"], f"r3-{arm}-{index}")
                        self.assertEqual(planned[-1][0], slot["delegate"]["argv"])
                        self.assertEqual(slot["delegate"]["command"], slot["delegate"]["argv"][0])
                        completed.append({"arm": arm, "slotId": slot["slotId"],
                                          "status": receipt["status"], "argv": slot["delegate"]["argv"]})

                    for index, failure_slot in enumerate(fixture["auth"]["slots"]):
                        failure_raw = self._invocation(
                            arm, failure_slot["responseRole"], f"failure-{index}", failure_slot)
                        failure_context = self._load_role_context(fixture, failure_raw)
                        with self.assertRaisesRegex(RuntimeError, "delegate process failed"):
                            government_roles.run_role(failure_raw, fixture["authRaw"], fixture["authSha"],
                                str(fixture["authPath"]), fixture["requestRaw"], authority,
                                expected_slot=failure_slot["slotId"], controller_context=failure_context,
                                role_resolver=role_resolver, cwd=request["actorRepository"])
                        self.assertEqual(planned[-1][0], failure_slot["delegate"]["argv"])
                    self.assertEqual(bounded.call_count, 6)
                with fixture["ledger"].transaction() as db:
                    rows = db.execute("SELECT slot_id,status FROM government_role_calls ORDER BY rowid").fetchall()
                self.assertEqual([x[0] for x in rows], [x["slotId"] for x in fixture["auth"]["slots"]] +
                                 [x["slotId"] for x in fixture["auth"]["slots"]])
                self.assertEqual([x[1] for x in rows], ["protocol-echo-valid"] * 3 + ["failed"] * 3)

        self.assertEqual(len(completed), 6)
        self.assertEqual(len({(x["arm"], x["slotId"]) for x in completed}), 6)

    def test_r2_commandless_shape_fails_in_pinned_predecessor_before_reservation(self):
        fixture = self._arm_fixture("government", commandless=True)
        predecessor = subprocess.run(
            ["git", "show", "419775a:experiments/government-comparison/runtime/government_roles.py"],
            cwd=Path(__file__).parents[3], capture_output=True, check=True, timeout=10).stdout
        module_name = "government_roles_r2_predecessor"
        predecessor_module = types.ModuleType(module_name)
        predecessor_module.__file__ = str(Path(government_roles.__file__).resolve())
        predecessor_module.__package__ = ""
        sys.modules[module_name] = predecessor_module
        try:
            exec(compile(predecessor, predecessor_module.__file__, "exec"), predecessor_module.__dict__)
            role = fixture["auth"]["slots"][0]
            invocation_raw = self._invocation("government", role["responseRole"], "r2-commandless", role)
            bounds = {"maxRoleStarts": 6, "maxRoleParallel": 2, "maxRoleProcessSeconds": 1200,
                      "sourceKey": native_controller.NATIVE_FIXTURE_GRANT_KEY, "product": "Government"}
            with patch.object(native_controller, "validate_native_fixture_grant", return_value=bounds), \
                 patch.object(predecessor_module, "bounded", side_effect=AssertionError("delegate must not run")):
                controller_context = self._load_role_context(fixture, invocation_raw)
                with self.assertRaisesRegex(ValueError, "delegate argv does not begin with its bound command"):
                    predecessor_module.run_role(
                        invocation_raw, fixture["authRaw"], fixture["authSha"],
                        str(fixture["authPath"]), fixture["requestRaw"], fixture["authority"],
                        expected_slot=role["slotId"], controller_context=controller_context,
                        cwd=fixture["request"]["actorRepository"])
            with fixture["ledger"].transaction() as db:
                tables = {row[0] for row in db.execute(
                    "SELECT name FROM sqlite_master WHERE type='table'").fetchall()}
                role_rows = (db.execute("SELECT COUNT(*) FROM government_role_calls").fetchone()[0]
                             if "government_role_calls" in tables else 0)
                self.assertEqual(role_rows, 0)
        finally:
            sys.modules.pop(module_name, None)


if __name__ == "__main__":
    unittest.main()
