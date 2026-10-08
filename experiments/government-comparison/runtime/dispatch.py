"""Finite S1 dispatcher. Explicit operator authority; no mode-only live activation."""
from __future__ import annotations
import hashlib
import json
import os
from pathlib import Path
import sys
import time

from identity_probe import events, summarize
from ledger import Ledger, LimitReached
from process import bounded
import runner
from measurement_profile import LEGACY, OBSERVED, FILES, profile, profile_sha

CHECKOUT = Path(__file__).resolve().parents[3]
LIVE_GAPS = ["provider-requests-and-internal-retries-unknown", "retrospective-token-overshoot",
             "unobserved-native-descendants", "same-user-filesystem-access", "serving-model-may-be-null",
             "effective-actor-file-tools-unproven-by-local-flags"]
R5_CONTROLLER_DEADLINE_SECONDS = 38
R5_CLEANUP_MARGIN_SECONDS = 6


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()


def execution_sha(request):
    return digest(encoded({key: value for key, value in request.items() if key != "operation"}))


def r5_process_timeout(existing_wall, controller_elapsed):
    """Reserve bounded-process cleanup time inside the R5 38-second controller deadline."""
    return min(float(existing_wall), R5_CONTROLLER_DEADLINE_SECONDS -
               float(controller_elapsed) - R5_CLEANUP_MARGIN_SECONDS)


def validate_r5_terminal_deadline(request, controller_elapsed):
    """Keep positive R5 qualification inside the complete controller window."""
    if (request.get("nativeFixtureR5Grant") is not None and
            float(controller_elapsed) > R5_CONTROLLER_DEADLINE_SECONDS):
        raise ValueError("R5 controller deadline exceeded before positive result finalization")


def external(path, *, existing=False):
    path = Path(path)
    if not path.is_absolute():
        raise ValueError("absolute external path required")
    path = path.resolve(strict=existing)
    if path == CHECKOUT or CHECKOUT in path.parents or path in CHECKOUT.parents:
        raise ValueError("path must be outside checkout hierarchy")
    return path


def positive(value, maximum):
    if type(value) not in (int, float) or not 0 < value <= maximum:
        raise ValueError("finite positive bound required")
    return value


def mechanical_pin():
    return {"executable": str(Path(sys.executable).resolve()),
            "executableSha256": digest(Path(sys.executable).read_bytes()),
            "fixtureSha256": digest(Path(__file__).with_name("mechanical_actor.py").read_bytes())}


def runtime_pins():
    pins = {name: digest(Path(__file__).with_name(name).read_bytes()) for name in
            ("dispatch.py", "adapter.py", "ledger.py", "process.py", "runner.py", "mechanical_actor.py", "identity_probe.py", "classic.py", "classic_integration.py", "government.py", "government_roles.py", "native_controller.py", "native_fixture_budget.py", "government-pin.json", "classic-pin.json", "measurement_profile.py", "context_allocation.py", "context_tools_allocation.py", "diagnostic_history.py")}
    pins["harness.py"] = digest(Path(__file__).parents[1].joinpath("harness.py").read_bytes())
    pins["prepare.py"] = digest(Path(__file__).parents[1].joinpath("prepare.py").read_bytes())
    for name in FILES.values():
        pins["public/" + name] = digest(Path(__file__).parents[1].joinpath("public", name).read_bytes())
    pins["public/context-additional-decision.json"] = digest(Path(__file__).parents[1].joinpath("public/context-additional-decision.json").read_bytes())
    pins["public/context-tools-decision.json"] = digest(Path(__file__).parents[1].joinpath("public/context-tools-decision.json").read_bytes())
    return pins


def validate_listing(listing):
    """Validate the frozen metadata result; no fresh RPC or serving identity inference."""
    raw = Path(listing["path"]).read_bytes()
    if digest(raw) != listing["sha256"]:
        raise ValueError("model-listing digest mismatch")
    value = json.loads(raw)
    matches = [item for item in value.get("targetEntries", []) if item.get("id") == runner.MODEL
               and item.get("model") == runner.MODEL and any(e.get("reasoningEffort") == runner.REASONING
               for e in item.get("supportedReasoningEfforts", []))]
    if (value.get("status") != "advertised" or value.get("accountType") != "chatgpt" or len(matches) != 1
            or value.get("requestedHighAdvertised") is not True or value.get("rpcErrors") != []
            or value.get("processReturnCode") != 0 or value.get("processStopReason") is not None):
        raise ValueError("successful authenticated exact model/high listing required")


class Authority:
    """Paths/digests come from the operator, never from Request-provided authority."""
    def __init__(self, grant_path, grant_sha256, protocol_path, protocol_sha256, *, allow_live=False):
        raw_grant = Path(grant_path).read_bytes()
        raw_protocol = Path(protocol_path).read_bytes()
        if digest(raw_grant) != grant_sha256 or digest(raw_protocol) != protocol_sha256:
            raise ValueError("operator authority digest mismatch")
        self.grant, self.protocol = json.loads(raw_grant), json.loads(raw_protocol)
        self.grant_sha, self.protocol_sha = grant_sha256, protocol_sha256
        g, p = self.grant, self.protocol
        if g.get("schemaVersion") != 1 or g.get("status") != "approved" or p.get("status") != "frozen":
            raise ValueError("approved Coordinator grant and frozen protocol required")
        if any("cumulativeLedgerBinding" in item for item in (g, p)):
            raise ValueError("superseded cumulative migration binding; fixed additional context allocation required")
        self.mode = g["mode"]
        if self.mode not in {"mechanical", "live"} or p.get("mode") != self.mode:
            raise ValueError("authority mode mismatch")
        if g.get("purpose") != ("s1-public-smoke" if self.mode == "live" else "s1-mechanics"):
            raise ValueError("finite S1 grant purpose required")
        if self.mode == "live" and (allow_live is not True or g.get("acceptedObservabilityGaps") != LIVE_GAPS):
            raise ValueError("explicit operator live switch and accepted observable limits required")
        common = json.loads(Path(__file__).parents[1].joinpath("public/resource-proposal.json").read_bytes())["commonLimits"]
        if p.get("commonLimits") != common or g.get("profileSha256") != digest(encoded(common)):
            raise ValueError("unchanged shared commonLimits required")
        self.profile_id = p.get("measurementProfileId", LEGACY)
        profile(self.profile_id)
        if self.profile_id != LEGACY and any(
                item.get("measurementProfileId") != self.profile_id or item.get("measurementProfileSha256") != profile_sha(self.profile_id)
                for item in (g, p)):
            raise ValueError("same explicit versioned measurement profile required in Grant and Protocol")
        self.context_allocation = None
        maximum_threshold = 10000
        if (self.mode == "live" and self.profile_id == OBSERVED) or any("contextAllocation" in item for item in (g, p)):
            from context_allocation import validate_allocation, ContextAllocationLedger
            self.context_ledger_class = ContextAllocationLedger
            from context_tools_allocation import ALLOCATION_ID, ContextToolsAllocationLedger
            if g.get("contextAllocation", {}).get("decisionId") == ALLOCATION_ID:
                from context_tools_allocation import validate_allocation
                self.context_ledger_class = ContextToolsAllocationLedger
                maximum_threshold = 50000
            self.context_allocation = validate_allocation(g, p)
        if g.get("protocolSha256") != protocol_sha256 or p.get("runtimeSourceSha256") != runtime_pins():
            raise ValueError("protocol/runtime source binding mismatch")
        if p.get("wrapperPythonSha256") != digest(Path(sys.executable).read_bytes()):
            raise ValueError("wrapper Python runtime binding mismatch")
        if type(g["maxActorSessions"]) is not int:
            raise ValueError("integer session bound required")
        positive(g["maxActorSessions"], common["trialActorCalls"])
        positive(g["maxSessionWallSeconds"], 180)
        # Only the exact new fixed decision can select the 50k draft window.
        positive(g["retrospectiveTokenThreshold"], maximum_threshold)
        if not (type(g.get("notBefore")) in (int, float) and type(g.get("expiresAt")) in (int, float)
                and float('-inf') < g["notBefore"] < g["expiresAt"] < float('inf')):
            raise ValueError("finite Coordinator grant validity interval required")
        if self.mode == "mechanical":
            pin = mechanical_pin()
        else:
            pin = runner.inspect(p["runnerExecutable"])
            if p.get("requestedModel") != runner.MODEL or p.get("requestedReasoning") != runner.REASONING:
                raise ValueError("exact selected model/high required; aliases forbidden")
            listing = p["authenticatedModelListing"]
            validate_listing(listing)
        if p.get("runnerPinSha256") != digest(encoded(pin)) or g.get("runnerPinSha256") != p["runnerPinSha256"]:
            raise ValueError("runner pin mismatch")
        self.pin, self.limits = pin, common
        self.ledger_path = external(g["ledgerPath"])
        self.result_directory = external(g["resultDirectory"], existing=True)
        if not self.result_directory.is_dir():
            raise ValueError("dedicated existing Result directory required")
        self.paths = [Path(grant_path).resolve(), Path(protocol_path).resolve(), self.ledger_path]
        self.adoption = None
        if self.profile_id == OBSERVED:
            adoption = p["profileAdoption"]
            content = Path(adoption["path"]).read_bytes()
            if digest(content) != adoption["sha256"]:
                raise ValueError("profile adoption digest mismatch")
            self.adoption = json.loads(content)
            self.paths.append(Path(adoption["path"]).resolve())
        if any(path == self.result_directory or self.result_directory in path.parents for path in self.paths):
            raise ValueError("Result directory cannot contain authority/ledger")

    def validate(self, raw):
        r = json.loads(raw)
        if r.get("operation") == "run_task" and not self.grant["notBefore"] <= time.time() < self.grant["expiresAt"]:
            raise ValueError("Coordinator grant is not currently valid for a new start")
        if (r.get("schemaVersion") != 1 or r.get("mode") != self.mode or r.get("trialId") != self.grant["trialId"]
                or r.get("operation") not in {"run_task", "resume", "stop"}
                or r.get("arm") not in {"conventional", "classic", "government"}
                or r.get("condition") not in {"greenfield", "brownfield"}):
            raise ValueError("Request identity/operation mismatch")
        authorization = [item for item in self.grant["authorizedRequests"] if item["dispatchId"] == r["dispatchId"]]
        if len(authorization) != 1 or authorization[0]["executionSha256"] != execution_sha(r):
            raise ValueError("Request execution is not granted")
        if r["operation"] == "run_task" and authorization[0]["initialRequestSha256"] != digest(raw):
            raise ValueError("exact initial Request bytes are not granted")
        root, evidence = external(r["actorRepository"], existing=True), external(r["evidenceDirectory"])
        if not root.is_dir() or evidence == root or root in evidence.parents or evidence in root.parents:
            raise ValueError("separate external Actor/evidence roots required")
        if (self.result_directory == root or root in self.result_directory.parents or self.result_directory in root.parents
                or self.result_directory == evidence or evidence in self.result_directory.parents or self.result_directory in evidence.parents):
            raise ValueError("Result directory must be separate from Actor and evidence")
        if any(path == root or root in path.parents for path in self.paths):
            raise ValueError("authority/ledger cannot reside in Actor root")
        if r.get("limits") != self.limits:
            raise ValueError("Request commonLimits mismatch")
        if (r.get("measurementProfileId", LEGACY) != self.profile_id
                or (self.profile_id == OBSERVED and r.get("measurementProfileSha256") != profile_sha(OBSERVED))):
            raise ValueError("Request measurement profile mismatch; arm-specific profiles forbidden")
        if self.context_allocation:
            if (r.get("contextAllocationDecisionId") != self.context_allocation["decisionId"] or
                    r.get("purpose") != "context-access" or r.get("smokeKind") != "effective-context-only" or
                    r.get("arm") != "conventional" or r.get("condition") != "greenfield" or
                    len(r.get("allowedReadPaths", [])) != 2 or r["allowedReadPaths"] != self.protocol.get("allowedReadPaths") or
                    r["allowedReadPaths"] != self.grant.get("allowedReadPaths")):
                raise ValueError("fixed additional allocation requires the exact two-file context request")
        positive(r["wallSeconds"], self.grant["maxSessionWallSeconds"])
        if r.get("purpose") not in {"setup", "context-access", "task", "review", "child", "repair"}:
            raise ValueError("explicit counted role required")
        if self.mode == "live":
            runner.validate_work_config()
            if r["purpose"] not in {"setup", "context-access"}:
                raise ValueError("this finite live route allows public S1 smokes only")
            if r.get("toolPolicy") != "ordinary-tools" or r["toolPolicy"] != self.protocol.get("toolPolicy"):
                raise ValueError("live work requires ordinary-tools; use metadata-only route for identity diagnostics")
            if r.get("sandbox", "workspace-write") != self.protocol.get("sandbox", "workspace-write"):
                raise ValueError("frozen common sandbox request required")
            if self.profile_id == OBSERVED and r["purpose"] == "context-access" and r.get("sandbox") != "read-only":
                raise ValueError("v2 context sentinel requires requested read-only sandbox")
        if not isinstance(r["task"].get("id"), str) or not r["task"]["id"]:
            raise ValueError("explicit task identity required")
        released = r["releasedInputs"]
        captured = {}
        for item in released:
            source = external(item["path"], existing=True)
            if (any(source == directory or directory in source.parents for directory in (root, evidence, self.result_directory))
                    or source in self.paths):
                raise ValueError("released inputs must be separate from mutable Actor/evidence/results/authority")
            content = source.read_bytes()
            if digest(content) != item["sha256"]:
                raise ValueError("released input mismatch")
            captured[item["path"]] = content
        if r["task"]["card"] not in released or r["prompt"] not in released:
            raise ValueError("public task card and explicit prompt must be released inputs")
        for entry in r.get("actorOwnedInputs", []):
            owned = external(entry["path"], existing=True)
            if root not in owned.parents or digest(owned.read_bytes()) != entry["sha256"]:
                raise ValueError("explicit Actor-owned input binding mismatch")
        return r, captured

    def ledger(self, operation="run_task", *, dispatch_id=None):
        def open_ledger(profile_id):
            if self.context_allocation:
                return self.context_ledger_class(self.ledger_path, self.grant["trialId"], self.limits,
                                               binding=self.context_allocation, profile_id=profile_id,
                                               resume_dispatch_id=dispatch_id)
            return Ledger(self.ledger_path, self.grant["trialId"], self.limits, profile_id=profile_id)
        binding = {"grantSha256": self.grant_sha, "protocolSha256": self.protocol_sha,
                   "mode": self.mode, "ledgerPath": str(self.ledger_path)}
        if operation in {"resume", "stop"}:
            ledger = open_ledger(None)
        else:
            try:
                ledger = open_ledger(self.profile_id)
            except ValueError as exc:
                if self.profile_id != OBSERVED or "profile binding mismatch" not in str(exc):
                    raise
                ledger = open_ledger(LEGACY)
        transition = {key: self.grant.get(key) for key in
                      ("priorAuthoritySha256", "priorActorSessions", "additionalActorSessions")}
        ledger.bind_dispatch(binding, maximum=self.grant["maxActorSessions"], transition=transition,
                             adoption=self.adoption if ledger.profile_id == LEGACY and self.profile_id == OBSERVED else None)
        return ledger


def command(request, authority):
    if request["mode"] == "mechanical":
        if request["arm"] != "conventional":
            raise ValueError("mechanics are not native product-arm simulations")
        fixture = external(request["mechanicalFixture"], existing=True)
        if digest(fixture.read_bytes()) != authority.pin["fixtureSha256"]:
            raise ValueError("only known deterministic fixture bytes allowed")
        case = request["mechanicalCase"]
        if case not in {"success", "unknown_requests", "no_usage", "sleep", "child"}:
            raise ValueError("unknown deterministic fixture case")
        return [authority.pin["executable"], str(fixture), case, request["dispatchId"]]
    return runner.prospective_argv(authority.pin["path"], request["actorRepository"], sandbox=request.get("sandbox", "workspace-write"))


def usage(directory, request):
    values = events(directory / "stdout.log")
    if request["mode"] == "live":
        observed = summarize(values)
        observed.update(tool_observations(values, request["toolPolicy"]))
        return observed
    counters = [v for v in values if v.get("type") == "mechanical.usage"]
    valid = len(counters) == 1 and all(type(counters[0].get(k)) is int and counters[0][k] >= 0
                                       for k in ("input_tokens", "output_tokens"))
    requests = counters[0].get("providerRequests") if len(counters) == 1 else None
    requests = requests if type(requests) is int and requests >= 0 else None
    return {"providerRequests": requests,
            "reportedInputPlusOutputTokens": counters[0]["input_tokens"] + counters[0]["output_tokens"] if valid else None,
            "rawUsage": counters, "counterScope": "synthetic-only", "resolvedProviderModel": None,
            "agentTurnsStarted": 0, "agentTurnsCompleted": 0}


def tool_observations(values, policy):
    tools = [v.get("item", {}).get("type") for v in values if v.get("type") in {"item.started", "item.completed"}
             and v.get("item", {}).get("type") not in {"agent_message", "reasoning"}]
    return {"toolEventCount": len(tools), "toolEventTypes": tools,
            "unexpectedToolActivity": bool(tools) and (policy == "forbidden" or any("collab" in str(t) for t in tools))}


def base_result(request, raw):
    return {"schemaVersion": 1, "trialId": request["trialId"], "operation": request["operation"],
            "mode": request["mode"], "requestSha256": digest(raw), "executionSha256": execution_sha(request),
            "dispatchId": request["dispatchId"], "status": "blocked", "candidateCommit": None,
            "measurementProfileId": request.get("measurementProfileId", LEGACY),
            "measurementProfileSha256": profile_sha(request.get("measurementProfileId", LEGACY)),
            "capabilities": [], "gaps": [], "receipts": [], "usage": None,
            "inferencePerformed": False}


def write_result(path, result):
    path = Path(path)
    # Derived copy only: the durable authoritative result resides in SQLite.
    temporary = path.with_name(path.name + "." + os.urandom(8).hex() + ".tmp")
    with temporary.open("wb") as out:
        out.write(encoded(result) + b"\n")
        out.flush()
        os.fsync(out.fileno())
    os.replace(temporary, path)


def dispatch(request_path, result_path, authority):
    raw = Path(request_path).read_bytes()
    r, captured = authority.validate(raw)
    output = external(result_path)
    if (output.parent != authority.result_directory or output in authority.paths or output == Path(request_path).resolve()
            or output == Path(r["actorRepository"]) or Path(r["actorRepository"]) in output.parents
            or output == Path(r["evidenceDirectory"]) or Path(r["evidenceDirectory"]) in output.parents
            or any(output == Path(item["path"]).resolve() for item in r["releasedInputs"])
            or (r["mode"] == "mechanical" and output == Path(r["mechanicalFixture"]).resolve())):
        raise ValueError("Result must be a separate external output, outside Actor/evidence/authority/input paths")
    result = base_result(r, raw)
    if r["arm"] == "government":
        return dispatch_government(request_path, result_path, authority, r, raw, captured, result)
    if r["arm"] == "classic":
        result.update(status="readiness_gap", inferencePerformed=False)
        result["gaps"] = ["native product dispatch/resume not integrated; no reservation or simulation"]
        write_result(result_path, result)
        return result
    argv = command(r, authority)
    if r["mode"] == "live" and r["operation"] == "run_task":
        runner.validate_start(r)
    ledger = authority.ledger(r["operation"], dispatch_id=r["dispatchId"])
    record = ledger.dispatch_record(r["dispatchId"])
    if record:
        ledger.verify_dispatch_binding(record, authority.profile_id)
    if record and record["execution_sha"] != execution_sha(r):
        raise ValueError("booked execution mismatch")
    if r["operation"] == "stop":
        ledger.stop()
        result.update(status="stopped", inferencePerformed=False)
        result["gaps"] = ["shared trial stop persisted; active owned processes observe it on poll; completion requires resume"]
    elif r["operation"] == "resume":
        result = recover(r, raw, argv, ledger, record, result, authority)
    else:
        try:
            attempt = ledger.reserve_dispatch(r["dispatchId"], execution_sha(r), raw, argv, r["task"]["id"],
                                              r["purpose"], authority.grant["maxActorSessions"])
        except LimitReached as exc:
            result["gaps"] = [str(exc)]
        else:
            if not ledger.claim_dispatch(r["dispatchId"], "launching"):
                result["gaps"] = ["reservation claimed by recovery; launch forbidden"]
            else:
                evidence = Path(r["evidenceDirectory"])
                try:
                    evidence.mkdir(parents=True, exist_ok=False)
                    (evidence / "request.json").write_bytes(raw)
                    (evidence / "released-inputs").mkdir()
                    for index, entry in enumerate(r["releasedInputs"]):
                        (evidence / "released-inputs" / str(index)).write_bytes(captured[entry["path"]])
                    def monitor(directory):
                        observed = usage(directory, r)
                        ledger.observe(attempt, observed["providerRequests"], observed["reportedInputPlusOutputTokens"])
                        _, reason = ledger.running_bound(r["task"]["id"])
                        if time.time() >= authority.grant["expiresAt"]:
                            reason = reason or "grant_expired"
                        if (observed["reportedInputPlusOutputTokens"] or 0) >= authority.grant["retrospectiveTokenThreshold"]:
                            reason = reason or "retrospective_session_token_threshold"
                        if r["mode"] == "live" and observed["agentTurnsStarted"] > 1:
                            reason = reason or "unexpected_second_agent_turn"
                        if observed.get("unexpectedToolActivity"):
                            reason = reason or "unexpected_tool_activity"
                        return reason
                    remaining, reason = ledger.running_bound(r["task"]["id"])
                    if reason:
                        result.update(status="stopped", inferencePerformed=False)
                        result["gaps"] = [reason]
                        result = ledger.complete_dispatch(r["dispatchId"], result, None, None)
                    else:
                        env = {key: val for key, val in os.environ.items() if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
                        bounded(argv, r["actorRepository"], evidence / "process", min(r["wallSeconds"], remaining, authority.grant["expiresAt"] - time.time()),
                                stdin=captured[r["prompt"]["path"]], env=env,
                                stop_path=evidence / "STOP", poll_stop=monitor)
                        result = finalize(r, raw, argv, ledger, attempt, authority)
                except (OSError, ValueError, KeyError) as exc:
                    # A launch may have occurred. Keep the reservation active without a terminal receipt.
                    result["inferencePerformed"] = None if r["mode"] == "live" else False
                    result["gaps"] = ["dispatch interrupted/ambiguous; resume without relaunch", str(exc)]
    write_result(result_path, result)
    return result


def dispatch_government(request_path, result_path, authority, request, raw, captured, result):
    """Run one accepted native queue under an acyclic, digest-bound bootstrap."""
    import government
    import native_controller
    from native_fixture_budget import DEADLINE as FIXTURE_PROCESS_DEADLINE
    product = request.get("product", {}).get("government") if isinstance(request.get("product"), dict) else None
    if not isinstance(product, dict) or not isinstance(product.get("roleAuthorization"), dict):
        result.update(status="readiness_gap", inferencePerformed=False,
                      gaps=government.readiness_gaps())
        write_result(result_path, result)
        return result
    if request["operation"] != "run_task":
        result.update(status="readiness_gap", inferencePerformed=False,
                      gaps=["native Government resume/recovery is not yet integrated; no relaunch"])
        write_result(result_path, result)
        return result
    bound = government.bind_request(request)
    import native_controller
    native_controller.validate_native_fixture_grant(request, captured, bound)
    binding = request.get("product", {}).get("government", {}).get("roleAuthorization")
    if not isinstance(binding, dict):
        raise ValueError("Request.product.government.roleAuthorization binding required")
    auth_path = Path(binding.get("path", ""))
    if not auth_path.is_absolute():
        raise ValueError("absolute role authorization path required")
    auth_path = auth_path.resolve(strict=True)
    auth_raw = auth_path.read_bytes()
    captured_auth = next((content for source, content in captured.items()
                          if Path(source).resolve() == auth_path), None)
    if digest(auth_raw) != binding.get("sha256") or captured_auth != auth_raw:
        raise ValueError("role authorization must be an exact released Request input")
    import government_roles
    government_roles.preflight_authorization(request, raw, authority, captured,
                                            str(auth_path), auth_raw, binding["sha256"])
    queue_plan = government.plan_request(request)
    argv = queue_plan["argv"]
    ledger = authority.ledger("run_task", dispatch_id=request["dispatchId"])
    record = ledger.dispatch_record(request["dispatchId"])
    if record:
        ledger.verify_dispatch_binding(record, authority.profile_id)
        if record.get("execution_sha") != execution_sha(request):
            raise ValueError("booked Government execution mismatch")
        result.update(status="incomplete", inferencePerformed=None,
                      gaps=["controller launch may have occurred; native resume is disabled and no relaunch is allowed"])
        write_result(result_path, result)
        return result

    evidence = Path(request["evidenceDirectory"])
    bootstrap_path = None
    bootstrap_sha = None
    snapshot_root = None
    process = None
    process_raw = None
    process_receipts = []
    process_verified = False
    controller_finished = False
    def native_fixture_start_gate():
        if request.get("nativeFixtureR4Grant") is not None:
            fresh_request, fresh_captured = authority.validate(raw)
            native_controller.validate_r4_entry_for_launch(fresh_request, fresh_captured, argv)
        if request.get("nativeFixtureR5Grant") is not None:
            fresh_request, fresh_captured = authority.validate(raw)
            native_controller.validate_r5_entry_for_launch(fresh_request, fresh_captured, argv)
    try:
        native_fixture_start_gate()
        ledger.reserve_controller_dispatch(request["dispatchId"], execution_sha(request), raw, argv,
                                           request["task"]["id"], request["purpose"],
                                           authority.grant["maxActorSessions"])
    except LimitReached as exc:
        result.update(status="blocked", gaps=[str(exc)])
        write_result(result_path, result)
        return result
    if not ledger.claim_dispatch(request["dispatchId"], "launching"):
        result.update(status="blocked", gaps=["controller booking claimed by recovery; launch forbidden"])
        write_result(result_path, result)
        return result
    try:
        evidence.mkdir(parents=True, exist_ok=False)
        (evidence / "request.json").write_bytes(raw)
        snapshot_root = evidence / "released-inputs"
        snapshot_root.mkdir()
        for index, item in enumerate(request["releasedInputs"]):
            (snapshot_root / str(index)).write_bytes(captured[item["path"]])
        bootstrap_path, bootstrap_sha = native_controller.write_bundle(
            evidence / "controller-bootstrap.json", request_path=request_path, request_raw=raw,
            authority=authority, authorization_path=auth_path,
            authorization_sha256=binding["sha256"], allow_live=(request["mode"] == "live"))
        controller_env = {key: value for key, value in os.environ.items()
                          if key not in ("OPENAI_API_KEY", "CODEX_API_KEY")}
        controller_env[native_controller.BOOTSTRAP_PATH_ENV] = str(bootstrap_path)
        controller_env[native_controller.BOOTSTRAP_SHA_ENV] = bootstrap_sha

        def monitor(directory):
            remaining, reason = ledger.running_bound(request["task"]["id"])
            if time.time() >= authority.grant["expiresAt"]:
                reason = reason or "grant_expired"
            elapsed = ledger.controller_elapsed(request["dispatchId"])
            if request.get("nativeFixtureR5Grant") is not None and elapsed >= (
                    R5_CONTROLLER_DEADLINE_SECONDS - R5_CLEANUP_MARGIN_SECONDS):
                reason = reason or "R5_controller_wall_deadline"
            if elapsed >= min(request["wallSeconds"], authority.grant["maxSessionWallSeconds"]):
                reason = reason or "controller_wall_deadline"
            if directory.joinpath("STOP").exists():
                reason = reason or "operator_stop_sentinel"
            return reason

        remaining, reason = ledger.running_bound(request["task"]["id"])
        if reason:
            result.update(status="stopped", inferencePerformed=False, gaps=[reason])
            result = ledger.finish_controller_dispatch(request["dispatchId"], result,
                                                      {"status": "not-launched", "reason": reason})
            write_result(result_path, result)
            return result
        # Divide remaining wall time conservatively across the controller plus
        # the maximum two concurrent native role processes.
        wall = min(float(request["wallSeconds"]), float(FIXTURE_PROCESS_DEADLINE),
                   float(authority.grant["maxSessionWallSeconds"]),
                   remaining / 3.0, max(0.0, float(authority.grant["expiresAt"]) - time.time()))
        if wall <= 0:
            raise LimitReached("no shared wall-time remains for native controller launch")
        native_fixture_start_gate()
        if request.get("nativeFixtureR5Grant") is not None:
            wall = r5_process_timeout(wall, ledger.controller_elapsed(request["dispatchId"]))
            if wall <= 0:
                raise LimitReached("R5 controller deadline leaves no process time after cleanup reserve")
        receipt = bounded(argv, request["actorRepository"], evidence / "process", wall,
                          env=controller_env, stop_path=evidence / "STOP", poll_stop=monitor)
        process_path = evidence / "process" / "process.json"
        process_raw = process_path.read_bytes()
        process = json.loads(process_raw)
        if process.get("argv") != argv or process.get("cwd") != request["actorRepository"]:
            raise ValueError("native controller process receipt identity mismatch")
        process_receipts = process.get("receipts")
        if not isinstance(process_receipts, list) or len(process_receipts) != 2:
            raise ValueError("native controller stdout/stderr receipts are required")
        for item, name in zip(process_receipts, ("stdout.log", "stderr.log")):
            log_path = (evidence / "process" / name).resolve(strict=True)
            if (Path(item.get("path", "")).resolve() != log_path or
                    item.get("sha256") != digest(log_path.read_bytes())):
                raise ValueError("native controller raw process receipt mismatch")
        process_verified = True
        result = government.translate_queue_result(request, raw, evidence / "process" / "stdout.log",
                                                  receipt["returnCode"])
        result.update(controllerProcess=process, inferencePerformed=None,
                      grantSha256=authority.grant_sha, protocolSha256=authority.protocol_sha,
                      runtimeSourceSha256=runtime_pins())
        result["receipts"].extend(process_receipts)
        result["receipts"].append({"path": str(process_path.resolve()), "sha256": digest(process_raw),
                                   "kind": "controller-process"})
        result["receipts"].append({"path": str(bootstrap_path), "sha256": bootstrap_sha,
                                   "kind": "controller-bootstrap"})
        for index, item in enumerate(request["releasedInputs"]):
            snapshot = snapshot_root / str(index)
            if digest(snapshot.read_bytes()) != item["sha256"]:
                raise ValueError("controller released-input snapshot mismatch")
            result["receipts"].append({"path": str(snapshot.resolve()), "sha256": item["sha256"],
                                       "kind": "released-input-snapshot"})
        elapsed_seconds = ledger.controller_elapsed(request["dispatchId"])
        validate_r5_terminal_deadline(request, elapsed_seconds)
        result = ledger.finish_controller_dispatch(request["dispatchId"], result,
                  {"argv": argv, "cwd": request["actorRepository"], "process": process,
                   "processSha256": digest(process_raw), "bootstrapSha256": bootstrap_sha,
                   "elapsedSeconds": elapsed_seconds})
        controller_finished = True
    except Exception as exc:
        result.update(status="incomplete", inferencePerformed=None,
                      gaps=["native controller outcome is ambiguous; reservation consumed and relaunch disabled", str(exc)])
        preserved = list(result.get("receipts", [])) if isinstance(result.get("receipts"), list) else []
        if process_verified and isinstance(process, dict) and isinstance(process_raw, bytes):
            result["controllerProcess"] = process
            for item in process_receipts:
                if isinstance(item, dict):
                    preserved.append(item)
            process_path = evidence / "process" / "process.json"
            preserved.append({"path": str(process_path.resolve()), "sha256": digest(process_raw),
                              "kind": "controller-process"})
        if bootstrap_path is not None and bootstrap_sha is not None and Path(bootstrap_path).is_file():
            preserved.append({"path": str(Path(bootstrap_path).resolve()), "sha256": bootstrap_sha,
                              "kind": "controller-bootstrap"})
        if snapshot_root is not None:
            for index, item in enumerate(request["releasedInputs"]):
                snapshot = snapshot_root / str(index)
                try:
                    snapshot_raw = snapshot.read_bytes()
                except OSError:
                    continue
                if digest(snapshot_raw) == item["sha256"]:
                    preserved.append({"path": str(snapshot.resolve()), "sha256": item["sha256"],
                                      "kind": "released-input-snapshot"})
        # Do not discard queue/run receipts from a translator result that was
        # constructed before a later controller bookkeeping failure.
        unique = {}
        for item in preserved:
            if isinstance(item, dict) and isinstance(item.get("path"), str) and isinstance(item.get("kind"), str):
                unique[(item["path"], item["kind"])] = item
        result["receipts"] = list(unique.values())
        if not controller_finished:
            process_record = {"argv": argv, "cwd": request["actorRepository"],
                              "process": process, "processSha256": digest(process_raw)
                              if isinstance(process_raw, bytes) else None,
                              "bootstrapSha256": bootstrap_sha,
                              "elapsedSeconds": ledger.controller_elapsed(request["dispatchId"]),
                              "status": "incomplete"}
            try:
                result = ledger.finish_controller_dispatch(request["dispatchId"], result, process_record)
                controller_finished = True
            except (ValueError, OSError, KeyError):
                result.setdefault("gaps", []).append("controller incomplete receipt could not be finalized")
    write_result(result_path, result)
    return result


def finalize(request, raw, argv, ledger, attempt, authority):
    directory = Path(request["evidenceDirectory"]) / "process"
    process = json.loads((directory / "process.json").read_bytes())
    if process["argv"] != argv or process["cwd"] != request["actorRepository"]:
        raise ValueError("process receipt execution mismatch")
    if (Path(request["evidenceDirectory"]) / "request.json").read_bytes() != raw:
        raise ValueError("initial evidence Request mismatch")
    expected = [(directory / name).resolve() for name in ("stdout.log", "stderr.log")]
    if len(process["receipts"]) != 2:
        raise ValueError("missing raw process receipts")
    for receipt, path in zip(process["receipts"], expected):
        if Path(receipt["path"]).resolve() != path or digest(path.read_bytes()) != receipt["sha256"]:
            raise ValueError("raw process receipt mismatch")
    observed = usage(directory, request)
    ledger.observe(attempt, observed["providerRequests"], observed["reportedInputPlusOutputTokens"])
    _, stop = ledger.running_bound(request["task"]["id"])
    if (observed["reportedInputPlusOutputTokens"] or 0) >= authority.grant["retrospectiveTokenThreshold"]:
        stop = stop or "retrospective_session_token_threshold"
    if request["mode"] == "live" and observed["agentTurnsStarted"] > 1:
        stop = stop or "unexpected_second_agent_turn"
    if observed.get("unexpectedToolActivity"):
        stop = stop or "unexpected_tool_activity"
    result = base_result(request, raw)
    result.update(attemptId=attempt, initialRequestSha256=digest(raw), usage=observed, process=process,
                  inferencePerformed=None if request["mode"] == "live" else False,
                  accountingStopReason=stop, grantSha256=authority.grant_sha, protocolSha256=authority.protocol_sha,
                  requestedModel=runner.MODEL if request["mode"] == "live" else None,
                  requestedReasoning=runner.REASONING if request["mode"] == "live" else None,
                  resolvedServingModel=None, runnerPin=authority.pin)
    measurement = ledger.measurement()
    result["measurementProfileId"] = measurement["profileId"]
    result["measurementProfileSha256"] = profile_sha(measurement["profileId"])
    result["contaminated"] = bool(observed.get("unexpectedToolActivity"))
    result["status"] = "stopped" if process["stopReason"] or stop else ("completed" if process["returnCode"] == 0 else "failed")
    if observed["providerRequests"] is None:
        result["gaps"].append("provider requests unobservable/null; agent turns are not provider requests")
    if (observed["reportedInputPlusOutputTokens"] is None or
            (observed["providerRequests"] is None and measurement["unknownProviderRequestsAdmission"] == "block")):
        result["gaps"].append("unknown required usage blocks later admission")
        if result["status"] == "completed":
            result["status"] = "incomplete"
    result["receipts"] = process["receipts"] + [{"path": str(directory / "process.json"),
                           "sha256": digest((directory / "process.json").read_bytes()), "kind": "process"}]
    for index, entry in enumerate(request["releasedInputs"]):
        snapshot = directory.parent / "released-inputs" / str(index)
        if digest(snapshot.read_bytes()) != entry["sha256"]:
            raise ValueError("released input snapshot mismatch")
        result["receipts"].append({"path": str(snapshot), "sha256": entry["sha256"], "kind": "released-input-snapshot"})
    if request["mode"] == "live":
        result["gaps"] += LIVE_GAPS
    return ledger.complete_dispatch(request["dispatchId"], result, observed["providerRequests"], observed["reportedInputPlusOutputTokens"])


def recover(r, raw, argv, ledger, record, result, authority):
    if not record:
        result["gaps"] = ["no original reservation; resume never reserves or launches"]
        return result
    if record["result"] is not None:
        recovered = json.loads(record["result"])
    elif record["phase"] in {"reserved", "recovering"}:
        initial = json.loads(record["request"])
        recovered = base_result(initial, record["request"])
        recovered.update(status="incomplete", inferencePerformed=False, attemptId=record["attempt"])
        recovered["gaps"] = ["interrupted booking consumed; no relaunch; usage unknown"]
        recovered = ledger.abandon_reserved(r["dispatchId"], recovered)
        if recovered is None:
            result["gaps"] = ["launcher won reservation CAS; resume later without relaunch"]
            return result
    elif (Path(r["evidenceDirectory"]) / "process/process.json").is_file():
        if json.loads(record["command"]) != argv:
            raise ValueError("booked command mismatch")
        recovered = finalize(json.loads(record["request"]), record["request"], argv, ledger, record["attempt"], authority)
    else:
        result["gaps"] = ["ambiguous launch or recovery without terminal receipt; reservation stays active; no relaunch/refill"]
        return result
    # Resume has its own exact bytes; the initial request binding stays visible.
    recovered = dict(recovered)
    recovered.update(operation=r["operation"], requestSha256=digest(raw), initialRequestSha256=digest(record["request"]))
    return recovered
