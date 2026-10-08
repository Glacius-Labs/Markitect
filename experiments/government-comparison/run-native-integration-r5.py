"""One bounded Government scope-identity Queue and, only after full success, its Resume."""
from __future__ import annotations

import argparse
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT / "runtime"))
import dispatch
import government
import government_integration
import government_roles
import classic
import classic_integration
import native_controller
from native_fixture_budget import (DEADLINE, FixtureBudget,
                                   validate_r5_entry_gate,
                                   validate_r5_grant_binding)
from process import bounded

EXTERNAL = Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-government-scope-20261008-r5")
EVIDENCE = ROOT / "evidence/government-scope-native-20261008-r5"
LEGACY = Path(r"C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008")
SOURCE_GRANT = LEGACY / "released-native-grant.json"
SUCCESSOR_GRANT = EXTERNAL / "released-r5-grant.json"
COORDINATOR_SNAPSHOT = EXTERNAL / "coordinator-r5-preflight-snapshot.json"
R5_KEY = "government-scope-native-20261008-r5"
GRANT_SHA = "c80edc0e9864c1e332eaee81838dad0f33640aa96b226d30d14228375c493c90"
HISTORY = EVIDENCE / "historical-native-starts.sqlite"
EXTERNAL_HISTORY = EXTERNAL / "history-native-starts.sqlite"
LIVE_HISTORY = LEGACY / "native-starts.sqlite"
R1_GRANT_SHA = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"
COORDINATION = Path(r"C:\Users\Consiliari\Glacius Labs\Markitect\docs\design\government\coordination-state.json")


def prepare():
    """Bind the outer mechanical Request/Authority to the already prepared R5 fixture."""
    base = EXTERNAL / "government"
    released = base / "released"
    preparation = json.loads((base / "preparation-final.json").read_bytes())
    role = json.loads((released / "role-auth.json").read_bytes())
    runtime_binding = binding(released / "runtime.json")
    authorization_binding = binding(released / "role-auth.json")
    actor = Path(preparation["actorRepository"])
    # prepare-native-r5 creates the parent directory; the outer controller gets
    # one fresh child directory so its evidence writer cannot reuse old bytes.
    evidence = base / "controller-evidence" / "outer-run"
    if evidence.exists():
        raise ValueError("R5 outer controller evidence must be a fresh absent directory")
    outer_results = base / "outer-results"
    outer_results.mkdir(exist_ok=False)
    card = write_new(released / "task-card.txt",
        b"Public deterministic native protocol fixture only. Use the selected Government task and configured role slots. No model/provider, semantic or human acceptance claim.\n")
    mechanical = write_new(released / "mechanical_actor.py", (ROOT / "runtime/mechanical_actor.py").read_bytes())
    original = binding(SOURCE_GRANT)
    if original["sha256"] != R1_GRANT_SHA:
        raise ValueError("R5 requires the exact immutable original R1 source grant")
    r5 = {**binding(SUCCESSOR_GRANT), "sourceKey": R5_KEY}
    if r5["sha256"] != GRANT_SHA:
        raise ValueError("R5 envelope bytes differ from the exact released grant")
    product = preparation["productGovernment"]
    product, inputs = government_integration.product_binding(
        repository=actor, runtime_path=runtime_binding["path"],
        runtime_raw=Path(runtime_binding["path"]).read_bytes(),
        backlog_path=released / "backlog.json",
        backlog_raw=(released / "backlog.json").read_bytes(),
        authorization_path=authorization_binding["path"],
        authorization_raw=Path(authorization_binding["path"]).read_bytes(),
        queue_state_directory=product["queueStateDirectory"])
    common = json.loads((ROOT / "public/resource-proposal.json").read_bytes())["commonLimits"]
    released_inputs = inputs + [original, r5, mechanical, binding(COORDINATOR_SNAPSHOT),
                                binding(EXTERNAL_HISTORY), role["diagnostics"], card,
                                binding(Path("C:/Python313/python.exe"))]
    request = {
        "schemaVersion": 1, "mode": "mechanical", "operation": "run_task",
        "trialId": role["trialId"], "dispatchId": "government-native-scope-r5",
        "arm": "government", "condition": "brownfield",
        "actorRepository": str(actor.resolve()), "evidenceDirectory": str(evidence.resolve()),
        "limits": common, "wallSeconds": 38, "baseCommit": preparation["baseCommit"],
        "task": {"id": role["taskId"], "card": card}, "purpose": "task",
        "releasedInputs": released_inputs, "prompt": card,
        "mechanicalFixture": mechanical["path"],
        "nativeFixtureGrant": {**original, "sourceKey": "native-s1-integration-fixtures-20261008"},
        "fixtureAuthorization": dict(government_roles.NATIVE_FIXTURE_AUTH),
        "nativeFixtureR5Grant": r5, "product": {"government": product}}
    request_binding = write_new(released / "request.json", request)
    runner_pin = dispatch.digest(dispatch.encoded(dispatch.mechanical_pin()))
    protocol = {"status": "frozen", "mode": "mechanical", "commonLimits": common,
                "runnerPinSha256": runner_pin, "runtimeSourceSha256": dispatch.runtime_pins(),
                "wrapperPythonSha256": dispatch.digest(Path(sys.executable).read_bytes()),
                "fixtureSourceGrant": original, "fixtureR5SourceGrant": r5,
                "semanticAcceptance": False, "humanAcceptance": False}
    protocol_binding = write_new(base / "protocol.json", protocol)
    grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics",
             "mode": "mechanical", "trialId": role["trialId"], "notBefore": time.time() - 1,
             "expiresAt": role["expiresAt"], "protocolSha256": protocol_binding["sha256"],
             "profileSha256": dispatch.digest(dispatch.encoded(common)),
             "runnerPinSha256": runner_pin, "ledgerPath": role["ledgerPath"],
             "resultDirectory": str(outer_results.resolve()), "maxActorSessions": 6,
             "maxSessionWallSeconds": request["wallSeconds"], "retrospectiveTokenThreshold": 10000,
             "fixtureSourceGrant": original, "fixtureR5SourceGrant": r5,
             "authorizedRequests": [{"dispatchId": request["dispatchId"],
                                     "executionSha256": dispatch.execution_sha(request),
                                     "initialRequestSha256": request_binding["sha256"]}]}
    grant_binding = write_new(base / "grant.json", grant)
    authority_binding = write_new(base / "authority.json",
                                  {"request": request_binding, "grant": grant_binding,
                                   "protocol": protocol_binding})
    return {"authority": authority_binding, "request": request_binding,
            "grant": grant_binding, "protocol": protocol_binding}


def freeze():
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT):
        raise ValueError("R5 source must be committed before the final preflight freeze")
    validate()
    _, request, _, _, _ = authority()
    paths = {Path(__file__), ROOT / "prepare-native-r5.py", SUCCESSOR_GRANT,
             COORDINATOR_SNAPSHOT, EXTERNAL_HISTORY, EVIDENCE / "independent-preflight-review.md",
             EVIDENCE / "host-success-contract.md", HISTORY, SOURCE_GRANT,
             Path("C:/Python313/python.exe"), ROOT / "public/resource-proposal.json",
             ROOT / "runtime/mechanical_actor.py"}
    paths.update((EXTERNAL / "government").rglob("*.json"))
    paths.update(ROOT / "runtime" / name for name in dispatch.runtime_pins()
                 if (ROOT / "runtime" / name).is_file())
    paths.add(ROOT / "runtime/government_integration.py")
    runtime = json.loads(Path(request["product"]["government"]["runtime"]["path"]).read_bytes())
    for configured_role in government.configured_roles(runtime):
        paths.update(Path(item["path"]) for item in configured_role.get("runtimeFiles", []))
    paths.update(Path(item["path"]) for item in request["releasedInputs"] if isinstance(item, dict))
    paths.add(Path(request["product"]["government"]["executable"]["path"]))
    paths.add(Path(government_integration.FIXTURE_ROOT / "deterministic_delegate.py"))
    paths.update((EXTERNAL / "government/actor").rglob("*"))
    paths = {path for path in paths if path.is_file() and ".git" not in path.parts}
    result = {"grantKey": R5_KEY, "status": "independently-reviewed-mechanical-fixture",
              "sourceCommit": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
              "runtimeSourceSha256": dispatch.runtime_pins(),
              "inputs": [binding(path) for path in sorted(paths, key=str)],
              "nativeStarts": 0, "semanticAcceptance": False, "humanAcceptance": False}
    return write_new(EVIDENCE / "preflight-freeze.json", result)


def write_new(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = value if isinstance(value, bytes) else dispatch.encoded(value) + b"\n"
    with path.open("xb") as stream:
        stream.write(raw)
    return {"path": str(path.resolve()), "sha256": dispatch.digest(raw)}


def binding(path):
    path = Path(path).resolve(strict=True)
    return {"path": str(path), "sha256": dispatch.digest(path.read_bytes())}


def authority():
    doc = json.loads((EXTERNAL / "government/authority.json").read_bytes())
    for item in doc.values():
        if binding(item["path"]) != item:
            raise ValueError("frozen R5 authority entry changed")
    result_authority = dispatch.Authority(
        doc["grant"]["path"], doc["grant"]["sha256"],
        doc["protocol"]["path"], doc["protocol"]["sha256"])
    raw = Path(doc["request"]["path"]).read_bytes()
    request, captured = result_authority.validate(raw)
    original = {key: request["nativeFixtureGrant"][key] for key in ("path", "sha256")}
    r5_binding = request.get("nativeFixtureR5Grant")
    if (original != {"path": str(SOURCE_GRANT.resolve(strict=True)),
                     "sha256": dispatch.digest(SOURCE_GRANT.read_bytes())} or
            result_authority.grant.get("fixtureSourceGrant") != original or
            result_authority.protocol.get("fixtureSourceGrant") != original or
            result_authority.grant.get("fixtureR5SourceGrant") != r5_binding or
            result_authority.protocol.get("fixtureR5SourceGrant") != r5_binding or
            r5_binding != {**binding(SUCCESSOR_GRANT), "sourceKey": R5_KEY} or
            request.get("dispatchId") != "government-native-scope-r5" or
            request.get("arm") != "government" or request.get("operation") != "run_task" or
            result_authority.mode != "mechanical" or
            result_authority.grant.get("maxActorSessions") != 6):
        raise ValueError("compiled Authority differs from the exact R5 Government-only grant")
    return result_authority, request, raw, captured, Path(doc["request"]["path"])


def validate():
    auth, request, raw, captured, _ = authority()
    validated = validate_r5_grant_binding(request, SOURCE_GRANT, dispatch.digest(SOURCE_GRANT.read_bytes()))
    validate_r5_entry_gate(validated)
    bound = government.bind_request(request)
    native_controller.validate_native_fixture_grant(request, captured, bound)
    role = request["product"]["government"]["roleAuthorization"]
    government_roles.preflight_authorization(
        request, raw, auth, captured, role["path"], Path(role["path"]).read_bytes(), role["sha256"])
    if bound["backlogValue"]["limits"]["maxParallelism"] > 2:
        raise ValueError("R5 Government native parallelism exceeds its grant")
    ledger = _fixture_budget(request).preflight_snapshot()
    if ledger.get("entryGate", {}).get("ready") is not True:
        raise ValueError("R5 live slot assignment is not ready for a native reservation")
    starts = ledger.get("starts")
    if not isinstance(starts, list):
        raise ValueError("R5 immutable native-start ledger snapshot is malformed")
    r5_labels = [row.get("label") for row in starts
                 if isinstance(row, dict) and row.get("label") in {
                     "government-native-scope-r5/queue", "government-native-scope-r5/resume"}]
    if (r5_labels not in ([], ["government-native-scope-r5/queue"],
                          ["government-native-scope-r5/queue", "government-native-scope-r5/resume"]) or
            len(starts) != 12 + len(r5_labels)):
        raise ValueError("R5 historical ledger prefix or sequential additive consumption is malformed")
    return {"arm": "government", "requestSha256": dispatch.digest(raw),
            "validated": True, "nativeStarts": 0, "providerCalls": 0,
            "sourcePins": dispatch.runtime_pins(),
            "ledgerRows": len(starts), "r5LedgerRows": len(r5_labels),
            "entryGate": ledger["entryGate"]}


def require_freeze():
    changes = subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).splitlines()
    allowed = "experiments/government-comparison/evidence/government-scope-native-20261008-r5/"
    if any(not line[3:].replace("\\", "/").startswith(allowed) for line in changes):
        raise ValueError("R5 source and non-R5 evidence must remain clean")
    validate()
    freeze_path = EVIDENCE / "preflight-freeze.json"
    freeze = json.loads(freeze_path.read_bytes())
    committed = subprocess.check_output([
        "git", "show", "HEAD:experiments/government-comparison/evidence/government-scope-native-20261008-r5/preflight-freeze.json"], cwd=ROOT)
    if committed != freeze_path.read_bytes():
        raise ValueError("R5 freeze must be committed byte-identically")
    if freeze.get("status") != "independently-reviewed-mechanical-fixture":
        raise ValueError("independent final R5 preflight review is missing")
    if (freeze.get("grantKey") != R5_KEY or
            freeze.get("runtimeSourceSha256") != dispatch.runtime_pins()):
        raise ValueError("R5 source/grant freeze differs from the admitted candidate")
    source_commit = freeze.get("sourceCommit")
    if not isinstance(source_commit, str) or len(source_commit) != 40:
        raise ValueError("R5 freeze lacks its exact source commit")
    subprocess.check_call(["git", "merge-base", "--is-ancestor", source_commit, "HEAD"], cwd=ROOT)
    expected = {str(Path(item["path"]).resolve()): item for item in freeze.get("inputs", [])}
    required = [Path(__file__), ROOT / "prepare-native-r5.py", EVIDENCE / "independent-preflight-review.md",
                EVIDENCE / "host-success-contract.md", SUCCESSOR_GRANT, COORDINATOR_SNAPSHOT,
                EXTERNAL_HISTORY, HISTORY, SOURCE_GRANT, Path("C:/Python313/python.exe"),
                ROOT / "public/resource-proposal.json", ROOT / "runtime/mechanical_actor.py"]
    required += [EXTERNAL / "government" / name for name in
                 ("authority.json", "grant.json", "protocol.json", "released/request.json")]
    for path in required:
        if str(path.resolve(strict=True)) not in expected:
            raise ValueError(f"required final R5 input is absent from the freeze: {path.name}")
    for item in expected.values():
        if binding(item["path"]) != item:
            raise ValueError("frozen R5 input changed before native start")


def _check_report_checks(report, runtime):
    expected = runtime.get("checks")
    observed = report.get("checks")
    if (not isinstance(expected, list) or not expected or not isinstance(observed, list) or
            len(expected) != len(observed)):
        raise ValueError("fresh configured Government checks are missing")
    by_key = {}
    for definition in expected:
        if not isinstance(definition, dict) or not isinstance(definition.get("name"), str) or not definition.get("name"):
            raise ValueError("configured fresh check definition is malformed")
        command = definition.get("run")
        tool = definition.get("tool")
        if tool is None and isinstance(command, list) and command:
            tool = command[0]
        timeout_seconds = definition.get("timeoutSeconds", 30)
        if (not isinstance(tool, str) or not tool or type(timeout_seconds) is not int or
                timeout_seconds <= 0):
            raise ValueError("configured fresh check tool or timeout is malformed")
        key = (definition["name"], tool)
        if key in by_key:
            raise ValueError("configured fresh check identity is duplicated")
        by_key[key] = definition
    seen = set()
    for item in observed:
        if not isinstance(item, dict):
            raise ValueError("fresh native GateResult is malformed")
        key = (item.get("Name"), item.get("Tool"))
        definition = by_key.get(key)
        if definition is None or key in seen:
            raise ValueError("fresh native check names/tools differ from the exact configured set")
        timeout_ms = definition.get("timeoutSeconds", 30) * 1000
        if (type(item.get("ExitCode")) is not int or item["ExitCode"] != 0 or
                type(item.get("Milliseconds")) is not int or item["Milliseconds"] < 0 or
                type(item.get("TimeoutMilliseconds")) is not int or
                item["TimeoutMilliseconds"] != timeout_ms or
                item["Milliseconds"] > timeout_ms):
            raise ValueError("fresh native check exit/time receipt is missing or outside its configured bound")
        seen.add(key)
    if seen != set(by_key):
        raise ValueError("fresh native check coverage is incomplete")


def _controller_elapsed(auth, request, process):
    runs = [item for item in auth.ledger().snapshot()["controllerRuns"]
            if item.get("dispatch_id") == request["dispatchId"]]
    if len(runs) != 1 or runs[0].get("end") is None or runs[0].get("status") != "completed":
        raise ValueError("completed Queue requires its exact terminal controller ledger receipt")
    raw_receipt = runs[0].get("process_receipt")
    if not isinstance(raw_receipt, str):
        raise ValueError("terminal controller receipt is missing")
    receipt = json.loads(raw_receipt)
    elapsed = receipt.get("elapsedSeconds") if isinstance(receipt, dict) else None
    if (not isinstance(receipt, dict) or receipt.get("process") != process or
            type(elapsed) not in (int, float) or elapsed < 0 or elapsed > 38 or
            process.get("stopReason") is not None):
        raise ValueError("completed Queue controller receipt is invalid or exceeds the 38-second deadline")
    return elapsed


def positive_checkpoint(process, queue, report, runtime):
    """Require full native acceptance and its exact positive completion checkpoint."""
    if (process.get("returnCode") != 0 or process.get("stopReason") is not None or
            type(process.get("wallSeconds")) not in (int, float) or
            process["wallSeconds"] > 38 or
            type(process.get("controllerElapsedSeconds")) not in (int, float) or
            process["controllerElapsedSeconds"] > 38 or
            queue.get("status") != "complete" or queue.get("nextStep") != "none" or
            queue.get("error") or queue.get("inFlightActors") != 0):
        raise ValueError("Queue/process is not completely positive")
    jobs = queue.get("jobs", [])
    if (len(jobs) != 1 or jobs[0].get("id") != government_integration.TASK_ID or
            jobs[0].get("state") != "accepted-scoped" or jobs[0].get("nextStep") != "complete" or
            jobs[0].get("resultStatus") not in (None, "completed")):
        raise ValueError("Queue job/result is not accepted-scoped and complete")
    if (report.get("status") != "accepted-scoped" or report.get("stage") != "complete" or
            report.get("error") or not report.get("candidateCommit") or not report.get("candidateTree")):
        raise ValueError("run result is not accepted-scoped/complete")
    _check_report_checks(report, runtime)
    promotion = report.get("promotion", {})
    if (promotion.get("status") != "promoted" or
            promotion.get("expectedOld") != runtime["expectedBase"] or
            promotion.get("newCommit") != report["candidateCommit"] or
            promotion.get("actualActive") != report["candidateCommit"] or
            not promotion.get("intentPath") or not promotion.get("completionPath")):
        raise ValueError("exact promotion and durable completion are required")
    roles = government.configured_roles(runtime)
    government._validate_run_report(
        report, report["runId"],
        {role["slotId"]: (role["phase"], role["responseRole"]) for role in roles},
        require_acceptance=True,
        root_review_slots={role["slotId"] for role in roles
                           if role["phase"] == "review" and not role.get("area")})
    return {"candidateCommit": report["candidateCommit"],
            "candidateTree": report["candidateTree"],
            "materialCandidateId": report["evidence"]["materialCandidateId"],
            "evidenceId": report["evidence"]["id"],
            "decisionId": report["decision"]["id"],
            "round": report["evidence"]["round"],
            "scopeIdentities": report["plan"]["integrationReviews"]}


def _fixture_budget(request):
    # The unchanged Classic packet is read only to preserve the original shared
    # allocation's binary map; this helper never launches the Classic product.
    classic_binary = classic.inspect_packet(classic_integration.PACKET)["binary"]["path"]
    return FixtureBudget(LIVE_HISTORY, SOURCE_GRANT, R1_GRANT_SHA,
                         {"government": government.PIN["accepted"]["binary"]["path"],
                          "classic": classic_binary}, request=request)


def _promotion_receipts(result):
    receipts = result.get("receipts") if isinstance(result, dict) else None
    if not isinstance(receipts, list):
        return None
    output = {}
    for kind in ("government-promotion-intent", "government-promotion-completion"):
        selected = [(item.get("path"), item.get("sha256")) for item in receipts
                    if isinstance(item, dict) and item.get("kind") == kind]
        if len(selected) != 1:
            return None
        output[kind] = selected
    return output


def r5_resume_within_controller_deadline(elapsed_seconds):
    """Pure terminal predicate for the complete Resume controller window."""
    return (type(elapsed_seconds) in (int, float) and
            0 <= elapsed_seconds <= 38)


def r5_resume_work_window(elapsed_seconds):
    """Reserve cleanup time inside 38s and bound Resume work to at most 32s."""
    if type(elapsed_seconds) not in (int, float) or elapsed_seconds < 0:
        raise ValueError("R5 Resume elapsed time must be finite and nonnegative")
    available = min(DEADLINE - 6, 38 - elapsed_seconds - 6)
    if available <= 0:
        raise ValueError("R5 Resume controller window leaves no work time after cleanup reserve")
    return available


def require_positive_queue():
    auth, request, raw, _, _ = authority()
    evidence = Path(request["evidenceDirectory"])
    process = json.loads((evidence / "process/process.json").read_bytes())
    process["controllerElapsedSeconds"] = _controller_elapsed(auth, request, process)
    translated = government.translate_queue_result(
        request, raw, evidence / "process/stdout.log", process["returnCode"])
    if translated.get("status") != "completed":
        raise ValueError("strict Queue Result translation is not completed")
    queue = json.loads((evidence / "process/stdout.log").read_bytes())
    job = queue["jobs"][0]
    report = json.loads(Path(job["reportPath"]).read_bytes())
    runtime = json.loads(Path(request["product"]["government"]["runtime"]["path"]).read_bytes())
    checkpoint = positive_checkpoint(process, queue, report, runtime)
    for key in ("intentPath", "completionPath"):
        record = json.loads(Path(report["promotion"][key]).read_bytes())
        intent = record.get("intent", {}) if key == "completionPath" else record
        if key == "completionPath" and record.get("observedRef") != report["candidateCommit"]:
            raise ValueError("promotion completion readback does not bind the accepted candidate")
        expected = {"expectedOld": runtime["expectedBase"], "newCommit": report["candidateCommit"],
                    "expectedTreeId": report["candidateTree"],
                    "materialCandidateId": checkpoint["materialCandidateId"],
                    "evidenceId": checkpoint["evidenceId"], "decisionId": checkpoint["decisionId"],
                    "activeRef": runtime["activeRef"], "idempotencyKey": report["runId"]}
        if any(intent.get(name) != value for name, value in expected.items()):
            raise ValueError("promotion intent/completion binding mismatch")
    active = subprocess.check_output(
        ["git", "-C", request["actorRepository"], "rev-parse", runtime["activeRef"]], text=True).strip()
    if active != report["candidateCommit"]:
        raise ValueError("promoted active ref readback mismatch")
    return {"checkpoint": checkpoint, "queue": queue,
            "translatedResult": translated, "runReport": binding(job["reportPath"]),
            "runtime": runtime}


def government_queue():
    require_freeze()
    validate()
    auth, request, raw, captured, request_path = authority()
    argv = government.plan_request(request)["argv"]
    budget = _fixture_budget(request)
    validate_r5_entry_gate(budget.r5)
    budget.reserve("government", "government-native-scope-r5/queue", argv)
    result = dispatch.dispatch(request_path, auth.result_directory / "government-scope-r5-queue.json", auth)
    process_path = Path(request["evidenceDirectory"]) / "process/process.json"
    if process_path.exists():
        budget.finish("government", "government-native-scope-r5/queue",
                      json.loads(process_path.read_bytes()))
    write_new(EVIDENCE / "government/queue-result.json", result)
    snapshots = []
    for index, item in enumerate(result.get("receipts", [])):
        content = Path(item["path"]).read_bytes()
        if dispatch.digest(content) != item["sha256"]:
            raise ValueError("R5 Queue receipt changed before archival snapshot")
        saved = write_new(EVIDENCE / "government/queue-checkpoint" / str(index), content)
        snapshots.append({"original": item, "snapshot": saved})
    write_new(EVIDENCE / "government/queue-checkpoint.json", snapshots)
    return result


def government_resume():
    require_freeze()
    auth, request, raw, captured, _ = authority()
    original_stdout = Path(request["evidenceDirectory"]) / "process/stdout.log"
    queue_value = json.loads(original_stdout.read_bytes())
    positive = require_positive_queue()
    queue_directory = Path(queue_value["queueDirectory"]).resolve(strict=True)
    if queue_directory.parent != Path(request["product"]["government"]["queueStateDirectory"]).resolve():
        raise ValueError("R5 Resume queue escaped its exact bound parent")
    argv = government.resume_argv(
        request["product"]["government"]["executable"]["path"], request["actorRepository"],
        request["product"]["government"]["backlog"]["path"], queue_directory)
    ledger = auth.ledger()
    before = ledger.snapshot()
    budget = _fixture_budget(request)
    controller_started = time.monotonic()
    budget.reserve("government", "government-native-scope-r5/resume", argv)
    fresh, fresh_captured = auth.validate(raw)
    native_controller.validate_r5_entry_for_launch(fresh, fresh_captured, argv)
    launch_gate_elapsed = time.monotonic() - controller_started
    work_window = r5_resume_work_window(launch_gate_elapsed)
    process = bounded(argv, request["actorRepository"],
                      Path(request["evidenceDirectory"]) / "resume-process", work_window,
                      env=native_controller.strip_bootstrap_environment(os.environ))
    budget.finish("government", "government-native-scope-r5/resume", process)
    resumed_path = Path(request["evidenceDirectory"]) / "resume-process/stdout.log"
    resumed_queue = json.loads(resumed_path.read_bytes())
    after = ledger.snapshot()
    if type(process.get("wallSeconds")) not in (int, float) or process["wallSeconds"] > DEADLINE:
        raise ValueError("R5 Resume exceeded its hard process wall deadline")
    replay_request = copy.deepcopy(request)
    replay_request["operation"] = "resume"
    replay_request["product"]["government"]["queueDirectory"] = str(queue_directory)
    replay_raw = dispatch.encoded(replay_request)
    translated = government.translate_queue_result(
        replay_request, replay_raw, resumed_path, process["returnCode"])
    same_report = binding(positive["runReport"]["path"]) == positive["runReport"]
    replay_promotions = _promotion_receipts(translated)
    queue_promotions = _promotion_receipts(positive["translatedResult"])
    result = {
        "process": process, "nativeResult": resumed_queue,
        "translatedResult": translated,
        "roleStartsBefore": before["actorSessions"], "roleStartsAfter": after["actorSessions"],
        "sameQueue": resumed_queue.get("queueDirectory") == queue_value.get("queueDirectory"),
        "sameNativeRoleStarts": resumed_queue.get("actorStarts") == queue_value.get("actorStarts"),
        "noAdditionalRoles": before["attempts"] == after["attempts"],
        "samePersistedQueue": resumed_queue == queue_value,
        "sameCandidateEvidenceDecision": all(
            resumed_queue.get("jobs", [{}])[0].get(key) == queue_value.get("jobs", [{}])[0].get(key)
            for key in ("runId", "reportPath", "reportDigest")),
        "sameBoundRunReport": same_report,
        "samePromotionReceipts": (replay_promotions is not None and queue_promotions is not None and
                                  replay_promotions == queue_promotions),
        "semanticAcceptance": False, "humanAcceptance": False}
    active = subprocess.check_output(
        ["git", "-C", request["actorRepository"], "rev-parse", positive["runtime"]["activeRef"]],
        text=True).strip()
    result["samePromotionReadback"] = active == positive["checkpoint"]["candidateCommit"]
    controller_elapsed = time.monotonic() - controller_started
    result["controllerElapsedSeconds"] = controller_elapsed
    result["workWindowSeconds"] = work_window
    result["controllerDeadlineSatisfied"] = r5_resume_within_controller_deadline(controller_elapsed)
    result["passed"] = (
        process["returnCode"] == 0 and process["stopReason"] is None and
        translated.get("status") == "completed" and
        result["controllerDeadlineSatisfied"] and
        all(result[key] for key in ("sameQueue", "sameNativeRoleStarts", "noAdditionalRoles",
                                    "samePersistedQueue", "sameCandidateEvidenceDecision",
                                    "sameBoundRunReport", "samePromotionReceipts",
                                    "samePromotionReadback")))
    write_new(EVIDENCE / "government/resume-result.json", result)
    return result


def run_once():
    require_freeze()
    write_new(EXTERNAL / "case-claimed.json", {"grantKey": R5_KEY, "claimedUtc": time.time()})
    outcome = {"grantKey": R5_KEY, "status": "incomplete", "noRetry": True,
               "semanticAcceptance": False, "humanAcceptance": False, "s1Complete": False}
    try:
        write_new(EVIDENCE / "coordinator-start-snapshot.json", COORDINATION.read_bytes())
        queue_result = government_queue()
        outcome["outerQueueStatus"] = queue_result.get("status")
        if queue_result.get("status") == "completed":
            outcome["positiveQueue"] = require_positive_queue()
            resume = government_resume()
            outcome["resumePassed"] = resume["passed"]
            if resume["passed"]:
                outcome["status"] = "bounded-mechanics-passed"
    except Exception as exc:
        outcome["error"] = f"{type(exc).__name__}: {exc}"
    finally:
        write_new(EVIDENCE / "terminal-result.json", outcome)
    return outcome


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("prepare", "validate", "freeze", "run-once"))
    args = parser.parse_args()
    print(json.dumps({"prepare": prepare, "validate": validate, "freeze": freeze,
                      "run-once": run_once}[args.action](), sort_keys=True))
