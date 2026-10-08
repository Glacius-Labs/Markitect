"""Exact R3 parent-owned native fixture driver. No real Actor/provider route.

Preparation and static validation do not launch Markitect. Every actual product
process consumes the original durable FixtureBudget before effects. Reservation
seconds count bounded controller and role sessions, including cleanup margin;
they are not an OS-wide sum of arbitrary compiler descendant wall durations.
"""
from __future__ import annotations
import argparse
import json
import os
from pathlib import Path
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
from native_fixture_budget import FixtureBudget, DEADLINE, validate_r3_grant_binding, validate_r3_entry_gate
from process import bounded

LEGACY = Path("C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008")
EXTERNAL = Path("C:/Users/Consiliari/Documents/Scientist-Probes/native-metadata-fixtures-20261008-r3")
EVIDENCE = ROOT / "evidence/native-integration/run-3"
SOURCE_GRANT = LEGACY / "released-native-grant.json"
SUCCESSOR_GRANT = EXTERNAL / "released-r3-a1-grant.json"
R3_KEY = "native-s1-contract-corrected-integration-20261008-r3"
GRANT_SHA = "b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e"


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


def budget(arm=None):
    request = json.loads((EXTERNAL / arm / "released/request.json").read_bytes()) if arm else None
    return FixtureBudget(LEGACY / "native-starts.sqlite", SOURCE_GRANT, GRANT_SHA, {
        "government": government.PIN["accepted"]["binary"]["path"],
        "classic": classic.inspect_packet(classic_integration.PACKET)["binary"]["path"]}, request=request)


def prepare(arm):
    base = EXTERNAL / arm
    released = base / "released"
    auth = json.loads((released / "role-auth.json").read_bytes())
    runtime = binding(released / "runtime.json")
    auth_binding = binding(released / "role-auth.json")
    repo = base / "actor"
    evidence = base / ("controller-evidence" if arm == "government" else "outer-controller-evidence")
    if evidence.exists():
        raise ValueError("R3 controller evidence must be a fresh absent directory")
    results = base / "outer-results"
    results.mkdir(exist_ok=False)
    card = write_new(released / "task-card.txt", (
        "Public deterministic native protocol fixture only. Use the selected "
        "product task and configured independent role slots. No model/provider, "
        "semantic or human acceptance claim.\n").encode())
    mechanical = write_new(released / "mechanical_actor.py", (ROOT / "runtime/mechanical_actor.py").read_bytes())
    source = binding(SOURCE_GRANT)
    correction = {**binding(SUCCESSOR_GRANT), "sourceKey": R3_KEY}
    if source["sha256"] != GRANT_SHA:
        raise ValueError("original source grant bytes changed")
    if arm == "government":
        prep = json.loads((base / "preparation-final.json").read_bytes())
        product, inputs = government_integration.product_binding(
            repository=repo, runtime_path=runtime["path"], runtime_raw=Path(runtime["path"]).read_bytes(),
            backlog_path=released / "backlog.json", backlog_raw=(released / "backlog.json").read_bytes(),
            authorization_path=auth_binding["path"], authorization_raw=Path(auth_binding["path"]).read_bytes(),
            queue_state_directory=prep["productGovernment"]["queueStateDirectory"])
        revision = prep["baseCommit"]
    else:
        prep = json.loads((base / "preparation-final.json").read_bytes())
        packet = classic.inspect_packet(classic_integration.PACKET)
        product = {
            "packet": {"path": str(classic_integration.PACKET.resolve()),
                       "manifestSha256": dispatch.digest((classic_integration.PACKET / "checksums.sha256").read_bytes())},
            "executable": {**binding(packet["binary"]["path"]), "sourceCommit": classic.EXPECTED_SOURCE,
                           "heldSourceCommit": classic.EXPECTED_HELD_SOURCE},
            "projectConfig": {"path": "examples/canonical-projection/canonical.yaml",
                              "sha256": dispatch.digest((repo / "examples/canonical-projection/canonical.yaml").read_bytes())},
            "runtime": runtime, "roleAuthorization": auth_binding,
            "controllerActions": list(native_controller.CLASSIC_ACTIONS)}
        inputs = [runtime, auth_binding]
        revision = prep.get("fixtureSourceCommit", prep.get("baseCommit"))
    common = json.loads((ROOT / "public/resource-proposal.json").read_bytes())["commonLimits"]
    request = {
        "schemaVersion": 1, "mode": "mechanical", "operation": "run_task",
        "trialId": auth["trialId"], "dispatchId": auth["dispatchId"], "arm": arm,
        "condition": "brownfield", "actorRepository": str(repo.resolve()),
        "evidenceDirectory": str(evidence.resolve()),
        "limits": common, "wallSeconds": 38 if arm == "government" else 180,
        "baseCommit": revision, "task": {"id": auth["taskId"], "card": card},
        "purpose": "task", "releasedInputs": inputs + [source, binding(SUCCESSOR_GRANT),
            binding(EXTERNAL / "coordinator-r3-a1-snapshot.json"), binding(EXTERNAL / "history.json"),
            binding(EXTERNAL / "coordinator-slot-activation-snapshot.json"),
            auth["diagnostics"], card], "prompt": card,
        "mechanicalFixture": mechanical["path"], "nativeFixtureGrant": {**source,
            "sourceKey": "native-s1-integration-fixtures-20261008"},
        "fixtureAuthorization": dict(government_roles.NATIVE_FIXTURE_AUTH),
        "nativeFixtureR3Grant": correction,
        "product": {arm: product}}
    request_binding = write_new(released / "request.json", request)
    pin = dispatch.digest(dispatch.encoded(dispatch.mechanical_pin()))
    protocol = {"status": "frozen", "mode": "mechanical", "commonLimits": common,
                "runnerPinSha256": pin, "runtimeSourceSha256": dispatch.runtime_pins(),
                "wrapperPythonSha256": dispatch.digest(Path(sys.executable).read_bytes()),
                "fixtureSourceGrant": source, "fixtureR3SourceGrant": correction,
                "semanticAcceptance": False, "humanAcceptance": False}
    protocol_binding = write_new(base / "protocol.json", protocol)
    grant = {"schemaVersion": 1, "status": "approved", "purpose": "s1-mechanics", "mode": "mechanical",
             "trialId": auth["trialId"], "notBefore": time.time() - 1, "expiresAt": auth["expiresAt"] + 60,
             "protocolSha256": protocol_binding["sha256"], "profileSha256": dispatch.digest(dispatch.encoded(common)),
             "runnerPinSha256": pin, "ledgerPath": auth["ledgerPath"], "resultDirectory": str(results.resolve()),
             "maxActorSessions": 6, "maxSessionWallSeconds": request["wallSeconds"],
             "retrospectiveTokenThreshold": 10000, "fixtureSourceGrant": source,
             "fixtureR3SourceGrant": correction,
             "authorizedRequests": [{"dispatchId": request["dispatchId"],
                 "executionSha256": dispatch.execution_sha(request), "initialRequestSha256": request_binding["sha256"]}]}
    grant_binding = write_new(base / "grant.json", grant)
    return write_new(base / "authority.json", {"request": request_binding, "grant": grant_binding,
                                               "protocol": protocol_binding})


def authority(arm):
    doc = json.loads((EXTERNAL / arm / "authority.json").read_bytes())
    for item in doc.values():
        if binding(item["path"]) != item:
            raise ValueError("frozen authority entry changed")
    a = dispatch.Authority(doc["grant"]["path"], doc["grant"]["sha256"],
                           doc["protocol"]["path"], doc["protocol"]["sha256"])
    raw = Path(doc["request"]["path"]).read_bytes()
    r, captured = a.validate(raw)
    # These are compiled mechanical-role bindings under the original allocation,
    # never an independent grant of actual Actor/provider/study sessions.
    source = {key: r["nativeFixtureGrant"][key] for key in ("path", "sha256")}
    if (a.grant.get("fixtureSourceGrant") != source or a.protocol.get("fixtureSourceGrant") != source or
            a.grant.get("fixtureR3SourceGrant") != r["nativeFixtureR3Grant"] or
            a.protocol.get("fixtureR3SourceGrant") != r["nativeFixtureR3Grant"] or
            a.grant.get("maxActorSessions") != 6 or a.mode != "mechanical" or
            source["sha256"] != GRANT_SHA):
        raise ValueError("compiled mechanical Authority exceeds original source allocation")
    return a, r, raw, captured, Path(doc["request"]["path"])


def validate(arm):
    a, r, raw, captured, request_path = authority(arm)
    bound = government.bind_request(r) if arm == "government" else classic_integration.bind_request(r, request_path)
    native_controller.validate_native_fixture_grant(r, captured, bound)
    role = r["product"][arm]["roleAuthorization"]
    government_roles.preflight_authorization(r, raw, a, captured, role["path"],
                                             Path(role["path"]).read_bytes(), role["sha256"])
    if arm == "government" and bound["backlogValue"]["limits"]["maxParallelism"] > 2:
        raise ValueError("Government fixture native parallelism exceeds source grant")
    return {"arm": arm, "requestSha256": dispatch.digest(raw), "validated": True,
            "nativeStarts": 0, "providerCalls": 0, "sourcePins": dispatch.runtime_pins()}


def require_freeze():
    # Live shared-slot authority is an independent pre-effect gate, never inferred from a queue.
    _, request, _, _, _ = authority("government")
    admitted = validate_r3_grant_binding(request, SOURCE_GRANT, GRANT_SHA)
    validate_r3_entry_gate(admitted)
    freeze = json.loads((EVIDENCE / "preflight-freeze.json").read_bytes())
    if freeze.get("status") != "independently-reviewed-mechanical-fixture":
        raise ValueError("independent final fixture preflight missing")
    if freeze.get("grantKey") != R3_KEY or freeze.get("runtimeSourceSha256") != dispatch.runtime_pins():
        raise ValueError("R3 source/grant freeze differs from the admitted candidate")
    expected = {str(Path(item["path"]).resolve()): item for item in freeze["inputs"]}
    required = [Path(__file__), ROOT / "public/native-integration-r3-native-preflight-review.md", SUCCESSOR_GRANT]
    required += [EXTERNAL / arm / name for arm in ("government", "classic")
                 for name in ("authority.json", "grant.json", "protocol.json", "released/request.json")]
    for path in required:
        if str(path.resolve()) not in expected:
            raise ValueError("required final source/authority/preflight absent from freeze")
    for item in expected.values():
        if binding(item["path"]) != item:
            raise ValueError("frozen native fixture input changed before start")


def government_queue():
    require_freeze()
    validate("government")
    a, r, raw, captured, request_path = authority("government")
    argv = government.plan_request(r)["argv"]
    b = budget("government")
    b.reserve("government", "government-native-contract-corrected-r3/queue", argv)
    result = dispatch.dispatch(request_path, a.result_directory / "government-native-positive.json", a)
    process = Path(r["evidenceDirectory"]) / "process/process.json"
    if process.exists():
        b.finish("government", "government-native-contract-corrected-r3/queue", json.loads(process.read_bytes()))
    write_new(EVIDENCE / "government/queue-result.json", result)
    snapshots = []
    for index, item in enumerate(result.get("receipts", [])):
        content = Path(item["path"]).read_bytes()
        if dispatch.digest(content) != item["sha256"]:
            raise ValueError("Government checkpoint receipt changed before snapshot")
        saved = write_new(EVIDENCE / "government/queue-checkpoint" / str(index), content)
        snapshots.append({"original": item, "snapshot": saved})
    write_new(EVIDENCE / "government/queue-checkpoint.json", snapshots)
    return result


def government_resume():
    require_freeze()
    a, r, raw, captured, _ = authority("government")
    original = Path(r["evidenceDirectory"]) / "process/stdout.log"
    value = json.loads(original.read_bytes())
    prior = json.loads((EVIDENCE / "government/queue-result.json").read_bytes())
    if prior.get("status") != "completed":
        raise ValueError("resume is only authorized after this positive case completed")
    queue = Path(value["queueDirectory"]).resolve(strict=True)
    if queue.parent != Path(r["product"]["government"]["queueStateDirectory"]).resolve():
        raise ValueError("resume queue escaped approved parent")
    argv = government.resume_argv(r["product"]["government"]["executable"]["path"],
                                  r["actorRepository"], r["product"]["government"]["backlog"]["path"], queue)
    ledger = a.ledger()
    before = ledger.snapshot()
    b = budget("government")
    b.reserve("government", "government-native-contract-corrected-r3/resume", argv)
    process = bounded(argv, r["actorRepository"], Path(r["evidenceDirectory"]) / "resume-process", DEADLINE,
                      env=native_controller.strip_bootstrap_environment(os.environ))
    b.finish("government", "government-native-contract-corrected-r3/resume", process)
    resumed = json.loads((Path(r["evidenceDirectory"]) / "resume-process/stdout.log").read_bytes())
    after = ledger.snapshot()
    result = {"process": process, "nativeResult": resumed,
              "roleStartsBefore": before["actorSessions"], "roleStartsAfter": after["actorSessions"],
              "sameQueue": resumed.get("queueDirectory") == value["queueDirectory"],
              "sameNativeRoleStarts": resumed.get("actorStarts") == value.get("actorStarts"),
              "noAdditionalRoles": before["actorSessions"] == after["actorSessions"],
              "semanticAcceptance": False, "humanAcceptance": False}
    write_new(EVIDENCE / "government/resume-result.json", result)
    return result


def classic_flow():
    require_freeze()
    validate("classic")
    a, r, _, _, request_path = authority("classic")
    classic_integration.assert_record_store_absent(json.loads(Path(r["product"]["classic"]["runtime"]["path"]).read_bytes()))
    b = budget("classic")
    session = native_controller.begin_classic_controller(request_path, a)
    try:
        execute = native_controller.run_classic_step(session, "execute", fixture_budget=b)
        if execute.get("returnCode") != 0:
            raise RuntimeError("Classic Execute failed; stop R3 case without later actions")
        print(json.dumps({"awaitingExactExecuteReview": execute,
                      "reviewPath": str(EXTERNAL / "classic/execute-review.json")}), flush=True)
    # Keep the same active controller Request/runtime across independent review.
        review_path = EXTERNAL / "classic/execute-review.json"
        deadline = time.monotonic() + 100
        while not review_path.exists() and time.monotonic() < deadline:
            time.sleep(0.2)
        if not review_path.exists():
            raise ValueError("exact Execute review absent; no Apply or automatic continuation")
        review = json.loads(review_path.read_bytes())
        for action in ("apply", "verify", "audit", "apply-replay"):
            capture = native_controller.run_classic_step(session, action, fixture_budget=b,
                external_review=review if action in {"apply", "apply-replay"} else None)
            if action != "apply-replay" and capture.get("returnCode") != 0:
                break
    finally:
        result = native_controller.finalize_classic_controller(session)
        write_new(EVIDENCE / "classic/flow-result.json", result)
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["prepare", "validate", "government-queue", "government-resume", "classic-flow", "snapshot"])
    parser.add_argument("--arm", choices=["government", "classic"])
    args = parser.parse_args()
    if args.action in {"prepare", "validate"} and not args.arm:
        parser.error("--arm required")
    result = (prepare(args.arm) if args.action == "prepare" else validate(args.arm) if args.action == "validate" else
              government_queue() if args.action == "government-queue" else government_resume() if args.action == "government-resume" else
              classic_flow() if args.action == "classic-flow" else budget().snapshot())
    print(json.dumps({key: result[key] for key in ("status", "gaps", "sameQueue", "sameNativeRoleStarts", "noAdditionalRoles")
                      if key in result} if "status" in result or "sameQueue" in result else result,
                     sort_keys=True), flush=True)
