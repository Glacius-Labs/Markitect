"""Pure Request/Result translation for the documented Government G5 queue API.

These helpers plan and decode. They do not launch a product process or claim the
accepted G5 candidate is study-ready.
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import re

QUEUE_API = "markitect.government-queue/v1alpha1"
RUN_API = "markitect.government-execution/v1alpha1"
PIN = json.loads(Path(__file__).with_name("government-pin.json").read_text(encoding="utf-8"))
G5_SOURCE = PIN["accepted"]["sourceCommit"]
G5_HANDOFF_SHA256 = PIN["accepted"]["handoff"]["sha256"]
G5_BINARY_SHA256 = PIN["accepted"]["binary"]["sha256"]
_HEX256 = re.compile(r"^[0-9a-f]{64}$")


def sha256(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def _file_binding(value, label, *, absolute):
    if not isinstance(value, dict) or not isinstance(value.get("path"), str):
        raise ValueError(f"{label} path binding required")
    path = Path(value["path"])
    if path.is_absolute() != absolute:
        raise ValueError(f"{label} path must be {'absolute' if absolute else 'repository-relative'}")
    if not _HEX256.fullmatch(str(value.get("sha256", ""))):
        raise ValueError(f"{label} SHA-256 binding required")
    return path


def _resolved_file(path: Path, label: str) -> Path:
    try:
        result = path.resolve(strict=True)
    except OSError as exc:
        raise ValueError(f"{label} file unavailable") from exc
    if not result.is_file():
        raise ValueError(f"{label} must be a file")
    return result


def _verify(path: Path, expected: str, label: str) -> Path:
    target = _resolved_file(path, label)
    if sha256(target.read_bytes()) != expected:
        raise ValueError(f"{label} digest mismatch")
    return target


def _inside(path: Path, root: Path) -> bool:
    return path == root or root in path.parents


def _validate_accepted_pin(gov: dict, handoff_path: Path, executable: Path,
                           source_sha: str, handoff: dict, pin: dict) -> None:
    accepted = pin.get("accepted", {})
    pinned_handoff = accepted.get("handoff", {})
    pinned_binary = accepted.get("binary", {})
    if accepted.get("acceptance", {}).get("status") != "accepted":
        raise ValueError("accepted Government source pin unavailable")
    if (source_sha != accepted.get("sourceCommit") or
            str(handoff_path) != str(Path(pinned_handoff.get("path", "")).resolve()) or
            gov["handoff"]["sha256"] != pinned_handoff.get("sha256") or
            executable != Path(pinned_binary.get("path", "")).resolve() or
            gov["executable"]["sha256"] != pinned_binary.get("sha256") or
            handoff.get("sourceSHA") != accepted.get("sourceCommit") or
            handoff.get("binary", {}).get("sha256") != pinned_binary.get("sha256") or
            Path(handoff.get("binary", {}).get("path", "")).resolve() != executable):
        raise ValueError("Request does not match the exact accepted Government G5 pin")


def bind_request(request: dict) -> dict:
    """Validate exact v1.2 Request.product bindings against a one-job G5 backlog.

    `product.government` uses file bindings for executable, projectConfig, order,
    runtime, backlog and queueStateDirectory. Runtime/backlog files must also be
    listed in Request.releasedInputs. The backlog may select only this Request's
    single public task. No files are rewritten.
    """
    if not isinstance(request, dict) or request.get("arm") != "government":
        raise ValueError("Government Request required")
    operation = request.get("operation")
    if operation not in {"run_task", "resume"}:
        raise ValueError("only documented Government queue/resume operations can be planned")
    root = Path(request.get("actorRepository", ""))
    if not root.is_absolute():
        raise ValueError("absolute actorRepository required")
    root = root.resolve(strict=True)
    evidence = Path(request.get("evidenceDirectory", ""))
    if not evidence.is_absolute():
        raise ValueError("absolute evidenceDirectory required")
    evidence = evidence.resolve()
    if _inside(evidence, root) or _inside(root, evidence):
        raise ValueError("Actor and evidence directories must be separate")
    product = request.get("product")
    gov = product.get("government") if isinstance(product, dict) else None
    if not isinstance(gov, dict):
        raise ValueError("Request.product.government bindings required")

    executable = _file_binding(gov.get("executable"), "Government executable", absolute=True)
    executable = _verify(executable, gov["executable"]["sha256"], "Government executable")
    source_sha = gov["executable"].get("sourceCommit")
    if not isinstance(source_sha, str) or not re.fullmatch(r"[0-9a-f]{40}", source_sha):
        raise ValueError("Government source SHA required")
    handoff_raw = _file_binding(gov.get("handoff"), "Government handoff", absolute=True)
    handoff_path = _verify(handoff_raw, gov["handoff"]["sha256"], "Government handoff")

    config_rel = _file_binding(gov.get("projectConfig"), "Government project config", absolute=False)
    order_rel = _file_binding(gov.get("order"), "Government Order", absolute=False)
    config_target = (root / config_rel).resolve()
    order_target = (root / order_rel).resolve()
    if not _inside(config_target, root) or not _inside(order_target, root):
        raise ValueError("Government project config and Order must stay inside actorRepository")
    config = _verify(config_target, gov["projectConfig"]["sha256"], "Government project config")
    order = _verify(order_target, gov["order"]["sha256"], "Government Order")

    runtime_raw = _file_binding(gov.get("runtime"), "Government runtime", absolute=True)
    backlog_raw = _file_binding(gov.get("backlog"), "Government backlog", absolute=True)
    released = {str(Path(item["path"]).resolve()): item.get("sha256")
                for item in request.get("releasedInputs", []) if isinstance(item, dict) and item.get("path")}
    if (_inside(handoff_path, root) or _inside(handoff_path, evidence) or
            released.get(str(handoff_path)) != gov["handoff"]["sha256"]):
        raise ValueError("Government handoff must be a separately released, digest-bound input")
    handoff = json.loads(handoff_path.read_bytes())
    _validate_accepted_pin(gov, handoff_path, executable, source_sha, handoff, PIN)
    binary_record = handoff.get("binary", {})
    if (handoff.get("phase") != "G5" or handoff.get("sourceSHA") != source_sha or
            binary_record.get("sha256") != gov["executable"]["sha256"] or
            Path(binary_record.get("path", "")).resolve() != executable):
        raise ValueError("Government handoff/source/binary pins do not match Request.product")
    runtime = _verify(runtime_raw, gov["runtime"]["sha256"], "Government runtime")
    backlog_path = _verify(backlog_raw, gov["backlog"]["sha256"], "Government backlog")
    for label, path, expected in (("runtime", runtime, gov["runtime"]["sha256"]),
                                  ("backlog", backlog_path, gov["backlog"]["sha256"])):
        if _inside(path, root) or _inside(path, evidence) or released.get(str(path)) != expected:
            raise ValueError(f"Government {label} must be a separately released, digest-bound input")

    state_dir = Path(gov.get("queueStateDirectory", ""))
    if not state_dir.is_absolute():
        raise ValueError("absolute queueStateDirectory required")
    state_dir = state_dir.resolve()
    if _inside(state_dir, root) or _inside(state_dir, evidence) or _inside(evidence, state_dir):
        raise ValueError("Government queue state must be separate from Actor and evidence directories")

    backlog = json.loads(backlog_path.read_bytes())
    if backlog.get("apiVersion") != QUEUE_API or Path(backlog.get("stateDirectory", "")).resolve() != state_dir:
        raise ValueError("Government backlog API/state binding mismatch")
    jobs = backlog.get("jobs")
    if not isinstance(jobs, list) or len(jobs) != 1 or not isinstance(jobs[0], dict):
        raise ValueError("one Request-bound Government job is required")
    job = jobs[0]
    if job.get("id") != request.get("task", {}).get("id") or job.get("dependsOn", []) != []:
        raise ValueError("Government backlog must select exactly the current task without future/dependent jobs")
    if job.get("configPath") != config_rel.as_posix() or job.get("orderPath") != order_rel.as_posix():
        raise ValueError("Government backlog config/Order mapping mismatch")
    if Path(job.get("runtimePath", "")).resolve() != runtime:
        raise ValueError("Government backlog runtime mapping mismatch")

    runtime_config = json.loads(runtime.read_bytes())
    if runtime_config.get("apiVersion") != RUN_API:
        raise ValueError("Government runtime API mismatch")
    run_state_dir = Path(runtime_config.get("stateDirectory", ""))
    if not run_state_dir.is_absolute():
        raise ValueError("Government runtime stateDirectory must be absolute")
    run_state_dir = run_state_dir.resolve()
    if _inside(run_state_dir, root) or _inside(run_state_dir, evidence) or _inside(evidence, run_state_dir):
        raise ValueError("Government runtime stateDirectory must be separate from Actor and evidence")
    limits = request.get("limits")
    native_limits = backlog.get("limits")
    if not isinstance(limits, dict) or not isinstance(native_limits, dict):
        raise ValueError("common Request limits and native backlog limits required")
    bounds = (("actorStarts", "taskActorCalls"), ("maxRepairs", "maxSemanticRepairRoundsPerTask"),
              ("maxWallTimeSeconds", "taskWallSeconds"), ("maxParallelism", "maxParallelActors"))
    for native_key, common_key in bounds:
        native_value, common_value = native_limits.get(native_key), limits.get(common_key)
        minimum = 0 if native_key == "maxRepairs" else 1
        if (type(native_value) is not int or type(common_value) is not int or
                not minimum <= native_value <= common_value):
            raise ValueError(f"Government backlog {native_key} exceeds or lacks its common Request bound")
        if (native_key == "maxParallelism" and request.get("nativeFixtureGrant") is not None and
                native_value > 2):
            raise ValueError("Government native fixture backlog exceeds its source-grant parallel limit")
    timeout = runtime_config.get("timeoutSeconds")
    if type(timeout) is not int or not 0 < timeout <= limits["taskWallSeconds"]:
        raise ValueError("Government runtime timeout exceeds the common task wall bound")
    roles = configured_roles(runtime_config)
    return {"executable": executable, "sourceCommit": source_sha, "handoff": handoff_path,
            "projectConfig": config, "order": order, "runtime": runtime,
            "backlog": backlog_path, "backlogValue": backlog, "job": job,
            "queueStateDirectory": state_dir, "runStateDirectory": run_state_dir, "roles": roles}


def configured_roles(runtime_config: dict) -> list[dict]:
    """Describe configured native slots; this is not a reservation schedule."""
    roles = []
    for key, phase in (("executor", "execute"), ("verifier", "review")):
        runner = runtime_config.get(key)
        if isinstance(runner, dict):
            roles.append(_role(phase, runner, response_role="executor" if phase == "execute" else "verifier"))
    recursion = runtime_config.get("recursion")
    if recursion is not None:
        if not isinstance(recursion, dict) or not isinstance(recursion.get("areas"), list):
            raise ValueError("malformed Government recursive runtime")
        for area in recursion["areas"]:
            if not isinstance(area, dict) or not isinstance(area.get("area"), dict):
                raise ValueError("malformed Government recursive Area runner binding")
            identity = area["area"]
            area_name = identity.get("name")
            if not isinstance(area_name, str) or not area_name:
                raise ValueError("recursive Area runner binding requires an identity")
            for key, phase in (("executor", "execute"), ("verifier", "review")):
                runner = area.get(key)
                if not isinstance(runner, dict):
                    raise ValueError("malformed Government recursive runner binding")
                roles.append(_role(phase, runner, area=area_name,
                                   response_role="executor" if phase == "execute" else "verifier"))
    for item in runtime_config.get("ressorts", []):
        if not isinstance(item, dict) or not isinstance(item.get("runner"), dict):
            raise ValueError("malformed Government Ressort runner binding")
        identity = item.get("ressort", {})
        name = identity.get("name") if isinstance(identity, dict) else None
        roles.append(_role("vote", item["runner"], ressort=name, response_role="verifier"))
    slot_ids = [role["slotId"] for role in roles]
    if (not roles or any(not role["slotId"] or not role["command"] for role in roles) or
            len(slot_ids) != len(set(slot_ids))):
        raise ValueError("Government runtime has no fully bound executable role slots")
    return roles


def _role(phase, runner, *, response_role, ressort=None, area=None):
    return {"phase": phase, "responseRole": response_role, "slotId": runner.get("slotId"),
            "command": runner.get("command"), "args": runner.get("args", []),
            "model": runner.get("model"), "modelOptions": runner.get("modelOptions"),
            "providerVersion": runner.get("providerVersion"), "runtimeFiles": runner.get("runtimeFiles", []),
            "ressort": ressort, "area": area}


def queue_argv(executable, repository, backlog) -> list[str]:
    return [str(Path(executable).resolve()), "government", "--repo", str(Path(repository).resolve()),
            "--action", "queue", "--backlog", str(Path(backlog).resolve()), "--write"]


def resume_argv(executable, repository, backlog, queue_directory) -> list[str]:
    return [str(Path(executable).resolve()), "government", "--repo", str(Path(repository).resolve()),
            "--action", "resume", "--backlog", str(Path(backlog).resolve()),
            "--queue", str(Path(queue_directory).resolve()), "--write"]


def plan_request(request: dict) -> dict:
    """Return a review-only native argv plan; never authorizes or starts it."""
    bound = bind_request(request)
    if request["operation"] == "run_task":
        argv = queue_argv(bound["executable"], request["actorRepository"], bound["backlog"])
    else:
        queue = Path(request["product"]["government"].get("queueDirectory", ""))
        if not queue.is_absolute() or not _inside(queue.resolve(), bound["queueStateDirectory"]):
            raise ValueError("resume requires the exact existing queue inside the bound queue state directory")
        argv = resume_argv(bound["executable"], request["actorRepository"], bound["backlog"], queue)
    gaps = readiness_gaps()
    gaps.append("argv is a review plan only; this helper cannot execute or dispatch the native queue")
    gaps.append("Government role middleware is not yet wired through the native queue/top-level dispatcher")
    return {"dispatchable": False, "operation": request["operation"], "argv": argv,
            "cwd": str(Path(request["actorRepository"]).resolve()), "sourceCommit": bound["sourceCommit"],
            "executableSha256": sha256(bound["executable"].read_bytes()),
            "inputDigests": {name: sha256(path.read_bytes()) for name, path in
                             (("projectConfig", bound["projectConfig"]), ("order", bound["order"]),
                              ("runtime", bound["runtime"]), ("backlog", bound["backlog"]),
                              ("handoff", bound["handoff"]))},
            "configuredRoles": bound["roles"], "gaps": gaps}


def _counter(summary, key):
    value = summary.get(key)
    if not isinstance(value, dict) or value.get("known") is not True or value.get("unknown") is not False:
        return None
    total = value.get("total")
    return total if type(total) is int and total >= 0 else None


def _safe_receipt(path: Path, root: Path, label: str) -> dict:
    resolved = _resolved_file(path, label)
    if not _inside(resolved, root):
        raise ValueError(f"{label} escaped Government queue directory")
    return {"path": str(resolved), "sha256": sha256(resolved.read_bytes()), "kind": label}


def _run_receipt(path: Path, run_directory: Path, label: str, allowed_names: set[str]) -> dict:
    resolved = _resolved_file(path, label)
    run_directory = run_directory.resolve(strict=True)
    if resolved.parent != run_directory or resolved.name not in allowed_names:
        raise ValueError(f"{label} is outside the exact native run directory")
    return {"path": str(resolved), "sha256": sha256(resolved.read_bytes()), "kind": label}


def _promotion_filename(run_id: str, completion: bool) -> str:
    prefix = "promotion-completion-" if completion else "promotion-"
    return prefix + sha256(run_id.encode("utf-8")) + ".json"


def _has_evidence_identity(report: dict) -> bool:
    evidence = report.get("evidence") if isinstance(report, dict) else None
    return (isinstance(evidence, dict) and
            isinstance(evidence.get("id"), str) and bool(evidence["id"]) and
            isinstance(evidence.get("materialCandidateId"), str) and bool(evidence["materialCandidateId"]) and
            type(evidence.get("round")) is int and evidence["round"] > 0)


def _validate_run_report(report: dict, run_id: str, configured_slots: dict[str, tuple[str, str]],
                         *, require_acceptance: bool = False,
                         root_review_slots: set[str] | None = None) -> list[str]:
    if not isinstance(report, dict):
        raise ValueError("Government run report must be an object")
    if report.get("apiVersion") != RUN_API or report.get("runId") != run_id:
        raise ValueError("Government run report identity mismatch")
    evidence = report.get("evidence")
    if not isinstance(evidence, dict):
        raise ValueError("Government evidence identity missing")
    material = evidence.get("materialCandidateId")
    evidence_id = evidence.get("id")
    decision = report.get("decision")
    if require_acceptance and decision is None:
        raise ValueError("accepted Government job requires a complete native AcceptanceDecision")
    if decision is not None and not isinstance(decision, dict):
        raise ValueError("Government decision record malformed")
    if decision is not None and (decision.get("materialCandidateId") != material or
                                 decision.get("evidenceId") != evidence_id or
                                 decision.get("round") != evidence.get("round")):
        raise ValueError("Government decision is not bound to report evidence/candidate")
    if decision is not None:
        digest_id = re.compile(r"^sha256:[0-9a-f]{64}$")
        if (not digest_id.fullmatch(str(decision.get("id", ""))) or
                not digest_id.fullmatch(str(decision.get("priorAuthorityDigest", ""))) or
                decision.get("priorAuthorityDigest") != report.get("priorConstitution")):
            raise ValueError("Government decision identity/prior authority binding is malformed")
    round_number = evidence.get("round")
    cabinet = report.get("cabinet") or []
    votes = report.get("votes") or []
    actors = report.get("actors") or []
    if not all(isinstance(items, list) for items in (cabinet, votes, actors)):
        raise ValueError("Government cabinet/vote/actor records must be arrays")
    if require_acceptance and not cabinet:
        raise ValueError("accepted Government job requires a nonempty frozen cabinet")
    if decision is not None and len(votes) != len(cabinet):
        raise ValueError("Government decision lacks one explicit vote per selected Ressort")
    expected_voters = set()
    for seat in cabinet:
        if not isinstance(seat, dict) or not isinstance(seat.get("ressort", {}), dict):
            raise ValueError("Government cabinet record malformed")
        identity = seat.get("ressort", {})
        key = (identity.get("namespace"), identity.get("name"))
        if (not all(isinstance(part, str) and part for part in key) or
                not isinstance(seat.get("slotId"), str) or not seat["slotId"] or
                key in expected_voters):
            raise ValueError("Government cabinet identity is missing or duplicated")
        expected_voters.add(key)
    observed_voters = set()
    actor_run_slots = {}
    actor_responses = {}
    for actor in actors:
        if not isinstance(actor, dict):
            raise ValueError("Government actor record malformed")
        result = actor.get("result", {})
        if not isinstance(result, dict) or not isinstance(result.get("Response", {}), dict):
            raise ValueError("Government actor Response record malformed")
        response = result.get("Response", {})
        if response.get("runId"):
            actor_run_slots[response["runId"]] = actor.get("slotId")
            actor_responses[response["runId"]] = response
    observed_vote_ids = []
    for vote in votes:
        if not isinstance(vote, dict):
            raise ValueError("Government vote record malformed")
        if (vote.get("materialCandidateId") != material or vote.get("evidenceId") != evidence_id or
                vote.get("round") != round_number):
            raise ValueError("Government vote candidate/evidence/round binding mismatch")
        if not isinstance(vote.get("ressort", {}), dict):
            raise ValueError("Government vote Ressort record malformed")
        identity = vote.get("ressort", {})
        key = (identity.get("namespace"), identity.get("name"))
        if not all(isinstance(part, str) and part for part in key):
            raise ValueError("Government vote Ressort identity malformed")
        if require_acceptance and vote.get("outcome") not in {"assent", "assent-unaffected"}:
            raise ValueError("accepted Government job requires positive final votes from every Ressort")
        if key not in expected_voters or key in observed_voters:
            raise ValueError("Government vote is outside or duplicated in the selected cabinet")
        seat = next(item for item in cabinet if (item.get("ressort", {}).get("namespace"),
                                                  item.get("ressort", {}).get("name")) == key)
        if (vote.get("priorMandate") != seat.get("priorMandate") or
                vote.get("mandateDigest") != seat.get("mandateDigest")):
            raise ValueError("Government vote authority differs from selected cabinet")
        provenance = vote.get("provenance", {})
        if (not isinstance(vote.get("id"), str) or not vote.get("id") or not isinstance(provenance, dict) or
                actor_run_slots.get(provenance.get("runId")) != provenance.get("slotId") or
                provenance.get("slotId") != seat.get("slotId")):
            raise ValueError("Government vote is not bound to the selected native verifier role")
        if require_acceptance:
            response = actor_responses.get(provenance.get("runId"), {})
            observations = response.get("verifierObservations")
            if (response.get("role") != "verifier" or response.get("outcome") != "passed" or
                    response.get("uncertainty") != [] or not isinstance(observations, list) or
                    len(observations) != 1 or not isinstance(observations[0], dict) or
                    observations[0].get("subject") != "government-vote" or
                    observations[0].get("outcome") != "passed"):
                raise ValueError("accepted Government vote requires its one explicit passing native verifier observation")
            try:
                vote_body = json.loads(observations[0].get("detail", ""))
            except (TypeError, json.JSONDecodeError) as exc:
                raise ValueError("accepted Government vote observation detail is malformed") from exc
            if (not isinstance(vote_body, dict) or vote_body.get("outcome") != vote.get("outcome") or
                    vote_body.get("materialCandidateId") != material or vote_body.get("evidenceId") != evidence_id or
                    vote_body.get("round") != round_number or not isinstance(vote_body.get("reason"), str) or
                    not vote_body["reason"].strip()):
                raise ValueError("accepted Government vote record differs from its native passing observation")
        observed_vote_ids.append(vote["id"])
        observed_voters.add(key)
    if decision is not None and observed_voters != expected_voters:
        raise ValueError("Government decision does not cover its complete selected cabinet")
    decision_vote_ids = decision.get("voteIds") if isinstance(decision, dict) else None
    if decision is not None and (not isinstance(decision_vote_ids, list) or
                                 any(not isinstance(item, str) for item in decision_vote_ids) or
                                 len(observed_vote_ids) != len(set(observed_vote_ids)) or
                                 set(decision_vote_ids) != set(observed_vote_ids)):
        raise ValueError("Government decision vote set is incomplete or duplicated")
    promotion = report.get("promotion")
    if promotion is not None and not isinstance(promotion, dict):
        raise ValueError("Government promotion record malformed")
    if (promotion or {}).get("status") == "promoted" and decision is None:
        raise ValueError("Government promotion has no bound decision")
    kinds = []
    for actor in actors:
        result = actor.get("result", {})
        response, receipt = result.get("Response", {}), result.get("Receipt", {})
        if not isinstance(receipt, dict):
            raise ValueError("Government actor Receipt record malformed")
        role, slot, phase = response.get("role"), actor.get("slotId"), actor.get("phase")
        if (not role or not slot or slot not in configured_slots or not phase or
                configured_slots[slot] != (phase, role) or not response.get("runId") or
                response.get("runId") != receipt.get("runId")):
            raise ValueError("Government actor role/receipt binding is incomplete")
        if response.get("inputDigest") != receipt.get("inputDigest"):
            raise ValueError("Government actor response/receipt input binding mismatch")
        if require_acceptance and phase == "review":
            observations = response.get("verifierObservations")
            uncertainty = response.get("uncertainty")
            if (response.get("outcome") != "passed" or uncertainty != [] or
                    not isinstance(observations, list) or not observations or
                    any(not isinstance(item, dict) or item.get("outcome") != "passed" or
                        not isinstance(item.get("subject"), str) or not item["subject"] or
                        not isinstance(item.get("detail"), str) or not item["detail"].strip()
                        for item in observations)):
                raise ValueError("accepted Government job requires explicit passing review observations without uncertainty")
        kinds.append(f"government-role:{phase}:{slot}:{role}")
    if require_acceptance:
        expected_root_review_slots = root_review_slots or set()
        observed_root_reviews = {actor.get("slotId") for actor in actors if isinstance(actor, dict)
                                 and actor.get("phase") == "review"}
        if not expected_root_review_slots.intersection(observed_root_reviews):
            raise ValueError("accepted Government job lacks a configured passing root-review actor receipt")
        plan = report.get("plan")
        integration_reviews = plan.get("integrationReviews") if isinstance(plan, dict) else None
        if not isinstance(integration_reviews, list) or not integration_reviews:
            raise ValueError("accepted Government job lacks its frozen integration-review plan")
        covered = set()
        for actor in actors:
            if isinstance(actor, dict) and actor.get("phase") == "review":
                scopes = actor.get("scopes")
                if not isinstance(scopes, list) or not all(isinstance(scope, str) for scope in scopes):
                    raise ValueError("accepted Government review actor must bind its reviewed scopes")
                covered.update(scopes)
        for identity in integration_reviews:
            if (not isinstance(identity, dict) or
                    not all(isinstance(identity.get(key), str) and identity[key]
                            for key in ("namespace", "name")) or
                    f"{identity['namespace']}/{identity['name']}" not in covered):
                raise ValueError("accepted Government job lacks a passing review receipt for a planned integration scope")
    for vote in votes:
        ressort = vote.get("ressort", {})
        identity = f"{ressort.get('namespace', '')}/{ressort.get('name', '')}".strip("/")
        kinds.append(f"government-vote:{identity}:{vote.get('outcome', 'unknown')}")
    if decision:
        kinds.append("government-decision")
    promotion = report.get("promotion")
    if promotion:
        kinds.append(f"government-promotion:{promotion.get('status', 'unknown')}")
    return kinds


def translate_queue_result(request: dict, request_raw: bytes, stdout_path, exit_code: int) -> dict:
    """Validate G5 queue/run reports and map them to a non-accepting v1.2 Result."""
    if not isinstance(request_raw, bytes):
        raise ValueError("exact Request file bytes required")
    try:
        recorded_request = json.loads(request_raw)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ValueError("Request file bytes must contain valid JSON") from exc
    if recorded_request != request:
        raise ValueError("Request object differs from exact Request file bytes")
    request_sha256 = sha256(request_raw)
    bound = bind_request(request)
    if request["operation"] == "resume":
        queue_dir = Path(request["product"]["government"]["queueDirectory"]).resolve(strict=True)
    else:
        # The queue output must name a newly created child of this pre-bound parent.
        queue_dir = None
    stdout = _resolved_file(Path(stdout_path), "Government queue stdout")
    evidence = Path(request.get("evidenceDirectory", "")).resolve(strict=True)
    if not _inside(stdout, evidence):
        raise ValueError("Government queue stdout must stay inside the bound evidence directory")
    value = json.loads(stdout.read_bytes())
    if not isinstance(value, dict):
        raise ValueError("Government queue result must be an object")
    if value.get("apiVersion") != QUEUE_API:
        raise ValueError("Government queue API mismatch")
    reported_queue = Path(value.get("queueDirectory", ""))
    if not reported_queue.is_absolute():
        raise ValueError("Government queueDirectory must be absolute")
    reported_queue = reported_queue.resolve(strict=True)
    if queue_dir is None:
        queue_dir = reported_queue
        if queue_dir.parent != bound["queueStateDirectory"]:
            raise ValueError("new Government queue is outside the bound state directory")
    elif reported_queue != queue_dir:
        raise ValueError("Government resume returned a different queue")
    if value.get("queueId") != queue_dir.name or value.get("backlogDigest") != sha256(bound["backlog"].read_bytes()):
        raise ValueError("Government queue/backlog binding mismatch")
    if type(exit_code) is not int:
        raise ValueError("native process exit code required")
    actor_starts, in_flight = value.get("actorStarts"), value.get("inFlightActors")
    if any(type(count) is not int or count < 0 for count in (actor_starts, in_flight)):
        raise ValueError("Government native actor counters malformed")

    sequence = value.get("journalSequence")
    if type(sequence) is not int or sequence < 1:
        raise ValueError("Government journal sequence missing")
    queue_report_path = queue_dir / f"queue-report-{sequence:08d}.json"
    queue_report_raw = _resolved_file(queue_report_path, "Government immutable queue report").read_bytes()
    if json.loads(queue_report_raw) != value:
        raise ValueError("Government stdout and immutable queue report differ")
    receipts = [{"path": str(stdout), "sha256": sha256(stdout.read_bytes()), "kind": "government-queue-stdout"},
                _safe_receipt(queue_report_path, queue_dir, "government-queue-report")]
    events_path = queue_dir / "events.jsonl"
    if not events_path.is_file():
        raise ValueError("Government queue event log is required")
    receipts.append(_safe_receipt(events_path, queue_dir, "government-queue-events"))

    jobs = value.get("jobs")
    if (not isinstance(jobs, list) or len(jobs) != 1 or not isinstance(jobs[0], dict) or
            jobs[0].get("id") != request.get("task", {}).get("id")):
        raise ValueError("Government result job does not match the single Request task")
    accepted_job = jobs[0].get("state") in {"accepted-scoped", "accepted-complete"}
    has_report = all(isinstance(jobs[0].get(key), str) and jobs[0].get(key)
                     for key in ("reportPath", "reportDigest", "runId"))
    if accepted_job and not has_report:
        raise ValueError("accepted Government job must bind its run report and digest")
    if exit_code != 0:
        status = "failed"
    elif value.get("status") == "blocked":
        status = "blocked"
    elif (value.get("status") == "complete" and in_flight == 0 and
          jobs[0].get("state") in {"accepted-scoped", "accepted-complete"}):
        status = "completed"
    else:
        status = "incomplete"
    native_roles = []
    incomplete_evidence_identity = False
    for job in jobs:
        path, digest_value, run_id = job.get("reportPath"), job.get("reportDigest"), job.get("runId")
        if any(x is not None for x in (path, digest_value, run_id)):
            if not all(isinstance(x, str) and x for x in (path, digest_value, run_id)):
                raise ValueError("Government job report binding is partial")
            if not _HEX256.fullmatch(digest_value):
                raise ValueError("Government job report digest malformed")
            if Path(run_id).name != run_id or not run_id.startswith("government-run-"):
                raise ValueError("Government runId is not a native operational directory name")
            run_directory = bound["runStateDirectory"] / run_id
            report_path = _resolved_file(Path(path), "Government run report")
            if (report_path.parent != run_directory or
                    report_path.name not in {"report.json", "recovered-report.json"}):
                raise ValueError("Government run report escaped its exact native run directory")
            if sha256(report_path.read_bytes()) != digest_value:
                raise ValueError("Government run report path/digest mismatch")
            report = json.loads(report_path.read_bytes())
            if not isinstance(report, dict) or report.get("apiVersion") != RUN_API or report.get("runId") != run_id:
                raise ValueError("Government run report identity mismatch")
            if not _has_evidence_identity(report):
                # A missing candidate/evidence identity makes a job non-accepting,
                # even if the queue labels it accepted. Preserve its bound file
                # receipt; only complete positive reports receive role/vote validation.
                incomplete_evidence_identity = True
                role_kinds = []
            else:
                role_kinds = _validate_run_report(report, run_id, {
                    role["slotId"]: (role["phase"], role["responseRole"]) for role in bound["roles"]},
                    require_acceptance=accepted_job,
                    root_review_slots={role["slotId"] for role in bound["roles"]
                                       if role["phase"] == "review" and not role.get("area")})
            receipts.append({"path": str(report_path), "sha256": digest_value, "kind": "government-run-report"})
            for kind in role_kinds:
                receipts.append({"path": str(report_path), "sha256": digest_value, "kind": kind})
                native_roles.append(kind)
            promotion = report.get("promotion")
            if isinstance(promotion, dict):
                for key, kind in (("intentPath", "government-promotion-intent"),
                                  ("completionPath", "government-promotion-completion")):
                    path_value = promotion.get(key)
                    if path_value:
                        completion = key == "completionPath"
                        receipts.append(_run_receipt(Path(path_value), run_directory, kind,
                                                     {_promotion_filename(run_id, completion)}))

    usage = value.get("usage") if isinstance(value.get("usage"), dict) else {}
    input_tokens, output_tokens = _counter(usage, "inputTokens"), _counter(usage, "outputTokens")
    if incomplete_evidence_identity and status == "completed":
        status = "incomplete"
    gaps = ["Native queue/run status and role receipts are product evidence, not independent task acceptance.",
            "Provider request/turn counts remain unknown; per-role middleware still needs native queue integration."]
    capabilities = ["validated Government queue, job, role, vote and promotion receipts"]
    if incomplete_evidence_identity:
        gaps.append("Native run report lacks evidence identity; preserved bound queue/run receipts as incomplete.")
        capabilities = ["validated Government queue, job and run-report file receipts"]
    return {"schemaVersion": 1, "trialId": request["trialId"], "operation": request["operation"],
            "mode": request["mode"], "requestSha256": request_sha256,
            "dispatchId": request.get("dispatchId"), "status": status, "candidateCommit": None,
            "capabilities": capabilities,
            "gaps": gaps,
            "receipts": receipts,
            "usage": {"providerRequests": None, "providerTurns": None,
                      "reportedInputPlusOutputTokens": (input_tokens + output_tokens
                                                           if input_tokens is not None and output_tokens is not None else None),
                      "rawUsage": usage, "counterScope": "government-native-queue-summary",
                      "resolvedProviderModel": None, "nativeActorStarts": value.get("actorStarts"),
                      "nativeInFlightActors": value.get("inFlightActors"),
                      "nativeRolesObserved": native_roles},
            "inferencePerformed": None,
            "government": {"queueId": value["queueId"], "queueStatus": value["status"],
                           "backlogDigest": value["backlogDigest"], "nativeJournalDigest": value.get("journalDigest"),
                           "nativeJobStates": [{"id": job["id"], "state": job.get("state"),
                                                "runId": job.get("runId")} for job in jobs]}}


def readiness_gaps():
    """Return concrete S1 blockers while retaining the accepted G5 pin."""
    return [
        (f"Government G5 source {G5_SOURCE} / binary SHA-256 {G5_BINARY_SHA256} is accepted for preparation; "
         "no native product process has been validated by this Scientist harness"),
        "product pin, project config, Order, runtime, backlog, queue state and all exact digests must be frozen in Request.product",
        "the operator-bound Government role authorization and wrapper runtime must be frozen and released for every root, recursive Area, and Ressort slot",
        "the common trial ledger controller/role bridge is implemented but has not been exercised through native queue dispatch",
        "provider request and turn counts remain unknown in agentexec Usage; native queue evidence is not independent acceptance",
    ]
