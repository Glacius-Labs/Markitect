"""Run finite public work-item trajectories through explicitly pinned adapters."""
from __future__ import annotations

import argparse
import base64
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import time

from adapters.contract import validate_descriptor, validate_result
from adapters.loader import load_adapter
import lifecycle
import workspace_policy


ROOT = Path(__file__).resolve().parent
CHECK_SPEC = importlib.util.spec_from_file_location(
    "variant_public_checker", ROOT / "public/common/checks/acceptance.py")
CHECKER = importlib.util.module_from_spec(CHECK_SPEC)
CHECK_SPEC.loader.exec_module(CHECKER)
ACTIVE_STATES = {"accepted", "starting", "running"}
UNRESOLVED_STATES = {"uncertain", "needs_input"}


def now():
    return datetime.now(timezone.utc)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def _workspace_manifest(root):
    """Hash the captured candidate bytes, excluding any Git metadata."""
    root = Path(root)
    return {path.relative_to(root).as_posix(): digest(path)
            for path in sorted(root.rglob("*"))
            if path.is_file() and ".git" not in path.relative_to(root).parts}


def _capture_initial_public_workspace(repo, audit):
    """Bind a stable copy of the prepared workspace before native readiness/actor work."""
    repo, audit = Path(repo), Path(audit)
    target = audit / "initial-public"
    binding_path = audit / "initial-public-binding.json"
    if target.exists() or binding_path.exists():
        raise ValueError("initial public workspace capture already exists")

    def files(root):
        result = {}
        for path in sorted(root.rglob("*")):
            relative = path.relative_to(root)
            if relative.parts and relative.parts[0] == ".git":
                continue
            if path.is_symlink():
                raise ValueError("prepared workspace contains a symlink")
            if path.is_file():
                result[relative.as_posix()] = digest(path)
            elif not path.is_dir():
                raise ValueError("prepared workspace contains a non-regular entry")
        return result

    before = files(repo)
    if not before:
        raise ValueError("prepared public workspace is empty")
    target.mkdir()
    for relative in before:
        source = repo / Path(relative)
        destination = target / Path(relative)
        destination.parent.mkdir(parents=True, exist_ok=True)
        source_bytes = source.read_bytes()
        if hashlib.sha256(source_bytes).hexdigest() != before[relative]:
            raise ValueError("prepared workspace changed while starting the initial public capture")
        destination.write_bytes(source_bytes)
    captured = files(target)
    after = files(repo)
    if captured != before or after != before:
        raise ValueError("prepared workspace changed during initial public capture")
    binding = {"schema": 1, "kind": "prepared_public_workspace", "sourceRepo": str(repo.resolve()),
               "capturedAt": now().isoformat(), "fileManifest": before}
    binding["fileManifestSha256"] = hashlib.sha256(
        (json.dumps(before, sort_keys=True, separators=(",", ":")) + "\n").encode("utf-8")).hexdigest()
    save(binding_path, binding)
    return binding


def _all_public_checks_pass(assessment):
    findings = assessment.get("findings") if isinstance(assessment, dict) else None
    return (isinstance(findings, list) and bool(findings) and
            all(isinstance(item, dict) and item.get("status") == "PASS" for item in findings))


def save(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    data = (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode("utf-8")
    temp = path.with_name(path.name + ".tmp")
    with temp.open("wb") as stream:
        stream.write(data)
        stream.flush()
    deadline = time.monotonic() + 2
    while True:
        try:
            temp.replace(path)
            break
        except PermissionError:
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.01)


def git(repo, *args):
    return subprocess.check_output(["git", "-C", str(repo), *args], timeout=600).decode("utf-8").strip()


def checked_call(adapter, action, *args):
    method = "ensure_runtime" if action == "ensure-runtime" else action
    return validate_result(method, getattr(adapter, method)(*args))


def _runtime_error(status):
    """Read the adapter's structured runtime-failure receipt, never native log text."""
    failure = status.get("runtimeFailure")
    return failure if failure not in (None, False, "") else None


def _integration_check(repo, captured, main_before, main_after_capture):
    """Require new main content and a clean actor index/worktree before advancing."""
    branch = captured.get("branch", {})
    status_raw = base64.b64decode(branch.get("statusPorcelainV2Base64", ""))
    controller_delta = b".study/station.json"
    dirty_rows = []
    for row in status_raw.split(b"\0"):
        if not row:
            continue
        if row.startswith(b"1 "):
            fields = row.split(b" ", 8)
            xy, path = fields[1], fields[8]
            if xy == b".M" and path == controller_delta:
                continue
        dirty_rows.append(row)
    staged_is_clean = branch.get("stagedDiffSha256") == hashlib.sha256(b"").hexdigest()
    main_advanced = captured.get("immutableMain", {}).get("commit") != main_before
    ref_stable = captured.get("immutableMain", {}).get("commit") == main_after_capture
    selected_head = branch.get("head")
    main_commit = captured.get("immutableMain", {}).get("commit")
    selected_head_integrated = selected_head == main_commit
    if not selected_head_integrated and selected_head:
        selected_head_integrated = subprocess.run(
            ["git", "-C", str(repo), "merge-base", "--is-ancestor", selected_head, main_commit],
            capture_output=True, timeout=60).returncode == 0
    worktree_clean = not dirty_rows and staged_is_clean
    return {"mainAdvanced": main_advanced, "mainRefStableAfterCapture": ref_stable,
            "selectedHeadIntegratedInMain": selected_head_integrated,
            "worktreeAndIndexCleanExceptControllerStationDelta": worktree_clean,
            "unexpectedGitStatusRowsBase64": [base64.b64encode(row).decode("ascii") for row in dirty_rows],
            "stagedDiffSha256": branch.get("stagedDiffSha256"),
            "controllerDeltaAllowed": controller_delta.decode("ascii")}


def run_station(adapter, prompt, parent, audit, number, expires, ownership):
    """Run one existing station and retain an owned unresolved handle on ambiguity."""
    stage = f"S{number}"
    action = "resume" if parent else "start"
    # If the adapter call itself fails after native dispatch, there may be no handle.
    # Stay conservative and never close or replay that possible execution.
    ownership.update(active=True, runId=None, disposition="dispatch outcome unknown")
    try:
        accepted = checked_call(adapter, action, *((parent, prompt) if parent else (prompt,)))
    except Exception as exc:
        result = {"schema": 1, "state": "uncertain", "runtimeFailure": {
            "source": "adapter dispatch call", "errorType": type(exc).__name__, "detail": str(exc)}}
        save(audit / f"adapter-start-{stage}.json", result)
        return result
    save(audit / f"adapter-start-{stage}.json", accepted)
    if accepted["state"] == "blocked":
        ownership.update(active=False, runId=None, disposition="known pre-dispatch block")
        return accepted
    handle = accepted.get("runId")
    ownership.update(runId=handle, disposition="adapter accepted or terminal result")
    print(json.dumps({"event": "adapter-attempt-accepted", "station": stage,
                      "runId": handle, "state": accepted["state"]}), flush=True)
    if not handle:
        result = {"schema": 1, "state": "uncertain", "runtimeFailure": {
            "source": "adapter dispatch result", "detail": "accepted result has no runId"}}
        save(audit / f"adapter-status-{stage}.json", result)
        return result
    if accepted["state"] in UNRESOLVED_STATES:
        return accepted
    if accepted["state"] not in ACTIVE_STATES:
        ownership.update(active=False, disposition="adapter reported terminal dispatch result")
        return accepted

    while True:
        try:
            status = checked_call(adapter, "status", handle)
            if status.get("runId") != handle:
                raise RuntimeError("status returned a different runId")
        except Exception as exc:
            status = {"schema": 1, "state": "uncertain", "runId": handle,
                      "runtimeFailure": {"source": "adapter status call",
                                         "errorType": type(exc).__name__, "detail": str(exc)}}
            save(audit / f"adapter-status-{stage}.json", status)
            return status
        save(audit / f"adapter-status-{stage}.json", status)
        if status["state"] not in ACTIVE_STATES:
            if status["state"] not in UNRESOLVED_STATES:
                ownership.update(active=False, disposition="adapter reported terminal status")
            return status
        if now() >= expires:
            try:
                cancel = checked_call(adapter, "cancel", handle)
                if cancel.get("runId") != handle:
                    raise RuntimeError("cancel returned a different runId")
                save(audit / f"adapter-cancel-{stage}.json", cancel)
            except Exception as exc:
                result = {"schema": 1, "state": "uncertain", "runId": handle,
                          "runtimeFailure": {"source": "adapter cancel call",
                                             "errorType": type(exc).__name__, "detail": str(exc)}}
                save(audit / f"adapter-status-{stage}.json", result)
                return result
            cleanup_deadline = time.monotonic() + 60
            while time.monotonic() < cleanup_deadline:
                try:
                    status = checked_call(adapter, "status", handle)
                    if status.get("runId") != handle:
                        raise RuntimeError("status returned a different runId")
                except Exception as exc:
                    status = {"schema": 1, "state": "uncertain", "runId": handle,
                              "runtimeFailure": {"source": "post-cancel status call",
                                                 "errorType": type(exc).__name__, "detail": str(exc)}}
                    save(audit / f"adapter-status-{stage}.json", status)
                    return status
                save(audit / f"adapter-status-{stage}.json", status)
                if status["state"] not in ACTIVE_STATES:
                    if status["state"] not in UNRESOLVED_STATES:
                        ownership.update(active=False, disposition="terminal after owned cancellation")
                    return status
                time.sleep(0.25)
            return {"schema": 1, "state": "uncertain", "runId": handle,
                    "runtimeFailure": {"source": "controller deadline",
                                       "detail": "owned cancellation has unresolved terminal state"}}
        time.sleep(0.25)


def _source_files(source_roots):
    files = set()
    for source_root in source_roots:
        rel = Path(source_root)
        if rel.is_absolute() or ".." in rel.parts:
            raise ValueError("sourceRoots must be relative paths within the Playground")
        root = (ROOT / rel).resolve()
        if ROOT.resolve() not in root.parents and root != ROOT.resolve():
            raise ValueError("source root escapes the Playground")
        if not root.is_dir():
            raise ValueError(f"source root does not exist: {source_root}")
        files.update(path.resolve() for path in root.rglob("*.py") if path.is_file())
    return sorted(files)


def _close(adapter, audit, result, *, ownership_unresolved):
    if adapter is None:
        result["closeDisposition"] = "adapter was not constructed"
        return
    if ownership_unresolved:
        result["closeDisposition"] = "deferred because native ownership or disposition is unresolved"
        result["stopStudy"] = True
        save(audit / "adapter-close-disposition.json", {"state": "deferred", "reason": result["closeDisposition"]})
        return
    try:
        receipt = checked_call(adapter, "close")
    except Exception as exc:
        receipt = {"schema": 1, "state": "uncertain", "runtimeFailure": {
            "source": "adapter close call", "errorType": type(exc).__name__, "detail": str(exc)}}
    save(audit / "adapter-close-result.json", receipt)
    result["closeDisposition"] = receipt
    if receipt.get("state") != "closed":
        result["stopStudy"] = True


def _capture_known_block(repo, audit, stage, reason):
    frozen = lifecycle.freeze(repo, audit, reason=reason)
    return {"station": stage, "state": "blocked_before_agent", "capture": str(audit / "final-freeze"),
            "captureBindingSha256": digest(audit / "final-freeze" / "binding.json"),
            "finalMainCommit": frozen["immutableMain"]["commit"],
            "assessment": "NOT RUN"}


def execute(plan, choice, destination, executable, source, overall_expiry):
    case = choice["case"]
    method = choice["method"]
    completion_target = plan.get("completionTarget", "main_merge")
    if completion_target not in {"main_merge", "workspace_snapshot"}:
        raise ValueError("completionTarget must be main_merge or workspace_snapshot")
    begun = now()
    expires = min(begun + timedelta(seconds=plan["jobWallSeconds"]), overall_expiry)
    if completion_target == "workspace_snapshot":
        final_reserve = plan.get("freshFinalSeconds", 2700) + plan.get("captureReserveSeconds", 900)
        actor_limit = begun + timedelta(seconds=plan.get("implementationSeconds", 10800))
        actor_expires = min(actor_limit, expires - timedelta(seconds=final_reserve))
    else:
        actor_expires = expires - timedelta(seconds=plan.get("captureAssessmentReserveSeconds", 900))
    case_root = destination / case
    case_root.mkdir()
    repo, audit = case_root / "repo", case_root / "audit"
    variant = dict(choice["config"])
    backend = variant.get("backend", "adapter-declared")
    profile = {"id": plan.get("profileId", plan["id"]), "method": method,
               "modelRequested": variant.get("model"), "effortRequested": variant.get("effort"),
               "runtimeRequested": variant.get("backend"), "entrance": "versioned adapter",
               "environment": "host-native", "referenceEquivalence": "unverified"}
    result = {"case": case, "method": method, "backend": backend, "sourceCommit": source,
              "completionTarget": completion_target,
              "startedAt": begun.isoformat(), "jobExpiresAt": expires.isoformat(), "stations": [],
              "status": "preparing", "humanAcceptance": "not established", "transportParity": "unverified"}
    save(case_root / "trajectory-result.json", result)
    adapter = None
    prepared = False
    ownership = {"active": False, "runId": None, "disposition": "no actor dispatched"}
    try:
        prepared_run = lifecycle.prepare(
            ROOT / "public", repo, audit, case=case, method=method, profile=profile,
            pins={"playgroundSourceCommit": source},
            authorization={"execution_authorized": True, "actualOrderRef": plan["id"],
                           "scope": "direct user adapter delivery; no prior grant reuse"},
            completion_target=completion_target)
        prepared = True

        sources = set(_source_files(plan["sourceRoots"]))
        for source_file in plan.get("sourceFiles", []):
            relative = Path(source_file)
            if relative.is_absolute() or ".." in relative.parts:
                raise ValueError("sourceFiles must be relative paths within the Playground")
            pinned_file = (ROOT / relative).resolve()
            if ROOT.resolve() not in pinned_file.parents or not pinned_file.is_file():
                raise ValueError(f"source file is missing or outside the Playground: {source_file}")
            sources.add(pinned_file)
        sources.update({Path(__file__).resolve(), (ROOT / "lifecycle.py").resolve(),
                        (ROOT / "public/common/checks/acceptance.py").resolve()})
        file_pins = {str(path): digest(path) for path in sorted(sources)}
        config = {**variant, "schema": 1, "execution_authorized": True,
                  "actualOrderPath": str(audit / "actual-order.json"),
                  "command": [str(executable), *choice.get("commandArgs", [])],
                  "filePins": {**file_pins, str(executable): digest(executable)},
                  "cwd": str(repo), "audit": str(audit),
                  "timeoutSeconds": plan["turnWallSeconds"],
                  "requestTimeoutSeconds": plan.get("requestTimeoutSeconds", 600),
                  "threadOptions": {}, "turnOptions": {},
                  "runtimeBinding": {"sourceCommit": source, "executableVersion": plan["executableVersion"],
                      "environment": "host-native", "referenceEquivalence": "unverified",
                      "globalConfigChanges": False,
                      **choice.get("runtimeBinding", {})}}
        config_path = audit / "adapter-config.json"
        save(config_path, config)
        order_expiry = min(actor_expires, expires)
        save(audit / "actual-order.json", {
            "execution_authorized": True, "actualOrderRef": plan["id"],
            "configSha256": digest(config_path), "runId": prepared_run["runId"],
            "expiresAt": order_expiry.isoformat(), "maxTurns": plan["outerTurnsPerTrajectory"],
            "runtimeChecksAuthorized": True, "maxRuntimeStartupChecks": plan.get("maxRuntimeStartupChecks", 2)})

        manifest_roots = choice["manifestSourceRoots"]
        manifest_sources = _source_files(manifest_roots)
        manifest_pins = {str(path): digest(path) for path in manifest_sources}
        manifest_path = audit / "adapter-manifest.json"
        save(manifest_path, {"schema": 1, "factory": choice["factory"],
                             "configPath": str(config_path), "sourcePins": manifest_pins})
        adapter = load_adapter(manifest_path)
        descriptor = validate_descriptor(adapter.describe())
        if descriptor["method"] != method:
            raise ValueError("declared adapter method differs from prepared run")
        save(audit / "adapter-descriptor.json", descriptor)

        git(repo, "checkout", "main")
        setup_source_root = choice.get("setupSourceRoot")
        if setup_source_root is not None:
            setup_source_root = str((ROOT / setup_source_root).resolve())
        limits = plan.get("limits", {})
        context = {"schema": 1, "method": method, "case": case, "station": "S1",
                   "completionTarget": completion_target,
                   "repoPath": str(repo), "auditPath": str(audit),
                   "setupStartedAt": begun.isoformat(), "profile": {
                       "model": variant.get("model"), "effort": variant.get("effort"),
                       "nativeBackend": variant.get("backend"),
                       "jobWallSeconds": plan["jobWallSeconds"],
                       "roleTurnSeconds": plan["turnWallSeconds"],
                       "sharedStartAllowance": plan["startAllowance"],
                       "sharedPilotStartAllowance": plan["startAllowance"],
                       "nativeHelperModel": (variant.get("runtimeOptions") or {}).get("nativeHelperModel"),
                       "nativeHelperEffort": (variant.get("runtimeOptions") or {}).get("nativeHelperEffort"),
                       "maxConcurrentAgents": limits.get("maxConcurrentAgents", plan.get("maxConcurrentAgents", 4)),
                       "maxDepth": limits.get("maxDepth", plan.get("maxDepth", 2)),
                       "nativeStartScope": plan.get("nativeStartScope", "parents/helpers/final/failed start requests"),
                       "nativeStartEnforcement": plan.get("nativeStartEnforcement",
                           "cooperative and receipt-based; hard global native enforcement unestablished"),
                       "startCounting": plan.get("startCounting", "observed receipts; hard native enforcement unknown")}}
        if setup_source_root:
            context["setupSourceRoot"] = setup_source_root

        try:
            setup = checked_call(adapter, "setup", context)
        except Exception as exc:
            setup = {"schema": 1, "state": "uncertain", "runtimeFailure": {
                "source": "adapter setup call", "errorType": type(exc).__name__, "detail": str(exc)}}
            ownership.update(active=True, disposition="adapter setup call failed; disposition unknown")
        save(audit / "adapter-setup-result.json", setup)
        setup_status = {"ready": "complete", "blocked": "blocked", "failed": "blocked",
                        "uncertain": "partial"}[setup["state"]]
        lifecycle.record_setup(audit, owner=descriptor["id"], status=setup_status, details=setup)
        if setup["state"] != "ready":
            if setup["state"] in UNRESOLVED_STATES:
                ownership.update(active=True, disposition="adapter setup is unresolved")
                result.update(status="setup_unresolved", setup=setup, stopStudy=True)
                _close(adapter, audit, result, ownership_unresolved=True)
            else:
                result.update(status="setup_blocked", setup=setup)
                result["stations"].append(_capture_known_block(repo, audit, "S1", "adapter setup did not become ready"))
                _close(adapter, audit, result, ownership_unresolved=False)
            result["endedAt"] = now().isoformat()
            save(case_root / "trajectory-result.json", result)
            save(audit / "trajectory-result.json", result)
            return result

        initial_public_binding = None
        if completion_target == "workspace_snapshot":
            initial_public_binding = _capture_initial_public_workspace(repo, audit)
            result["initialPublicWorkspaceBindingSha256"] = digest(audit / "initial-public-binding.json")

        if completion_target == "main_merge":
            git(repo, "checkout", "codex/backlog")
            git(repo, "merge", "--ff-only", "main")
        try:
            readiness = checked_call(adapter, "ensure-runtime")
        except Exception as exc:
            readiness = {"schema": 1, "state": "uncertain", "runtimeFailure": {
                "source": "adapter ensure_runtime call", "errorType": type(exc).__name__, "detail": str(exc)}}
            ownership.update(active=True, disposition="runtime readiness call failed; disposition unknown")
        save(audit / "adapter-runtime-readiness.json", readiness)
        save(audit / "study-setup-binding.json", {
            "sourceCommit": source, "setupCommit": git(repo, "rev-parse", "main"),
            "completionTarget": completion_target,
            "initialPublicWorkspaceBindingSha256": (digest(audit / "initial-public-binding.json")
                                                     if initial_public_binding is not None else None),
            "setupStartedAt": begun.isoformat(), "setupEndedAt": now().isoformat(),
            "jobExpiresAt": expires.isoformat(), "actorExpiresAt": actor_expires.isoformat(),
            "configSha256": digest(config_path), "manifestSha256": digest(manifest_path),
            "inputs": lifecycle._manifest(repo, exclude_git=True), "runtimeReadiness": readiness})
        if readiness["state"] != "ready":
            lifecycle.record_execution(audit, metadata={"readiness": readiness,
                "agentStarted": False if readiness["state"] != "uncertain" else "unknown"})
            result.update(runtimeReadiness=readiness)
            if readiness["state"] in UNRESOLVED_STATES:
                ownership.update(active=True, disposition="runtime readiness is unresolved")
                result.update(status="runtime_readiness_unresolved", stopStudy=True)
                _close(adapter, audit, result, ownership_unresolved=True)
            else:
                result.update(status="runtime_blocked_before_agent")
                result["stations"].append(_capture_known_block(repo, audit, "S1", "runtime was not ready before agent start"))
                _close(adapter, audit, result, ownership_unresolved=False)
            result["endedAt"] = now().isoformat()
            save(case_root / "trajectory-result.json", result)
            save(audit / "trajectory-result.json", result)
            return result

        result.update(status="running", runtimeReadiness=readiness)
        prompt = (ROOT / "public/task-prompt.txt").read_text(encoding="utf-8")
        if completion_target == "workspace_snapshot":
            prompt = workspace_policy.render("task-prompt.txt", prompt)
        parent = None
        main_before_station = git(repo, "rev-parse", "main")
        for number in range(1, 5):
            stage = f"S{number}"
            status = run_station(adapter, prompt, parent, audit, number, actor_expires, ownership)
            lifecycle.record_execution(audit, metadata={"station": stage, "adapterStatus": status})
            if status["state"] in UNRESOLVED_STATES:
                result.update(status="unresolved_runtime", reason="native disposition unresolved; no snapshot, close, or replay",
                              unresolvedStatus=status, stopStudy=True,
                              closeDisposition="deferred because native ownership or disposition is unresolved")
                save(audit / "unresolved-runtime.json", status)
                break
            if status["state"] == "blocked":
                result.update(status="runtime_blocked_before_agent", blockedStatus=status)
                result["stations"].append(_capture_known_block(repo, audit, stage, "adapter blocked before dispatch"))
                break

            captured = lifecycle.snapshot(repo, audit)
            snapshot_root = audit / f"snapshot-{stage}"
            if completion_target == "workspace_snapshot":
                candidate_info = captured.get("assessmentCandidate")
                if (captured.get("completionTarget") != completion_target or
                        not isinstance(candidate_info, dict) or
                        candidate_info.get("kind") != "workspace_snapshot" or
                        not isinstance(candidate_info.get("manifest"), dict) or
                        not candidate_info.get("manifest")):
                    raise ValueError("workspace snapshot is missing its bound nonempty assessment candidate")
                candidate_rel = Path(candidate_info.get("path", ""))
                if candidate_rel.is_absolute() or ".." in candidate_rel.parts:
                    raise ValueError("workspace snapshot candidate path escapes its capture")
                candidate = (snapshot_root / candidate_rel).resolve()
                if snapshot_root.resolve() not in candidate.parents:
                    raise ValueError("workspace snapshot candidate path escapes its capture")
                candidate_manifest = _workspace_manifest(candidate)
                if candidate_manifest != candidate_info["manifest"]:
                    raise ValueError("workspace snapshot candidate bytes differ from its bound manifest")
                assessment_path = candidate
            else:
                assessment_path = snapshot_root / "immutable-main"
                candidate_info = {"kind": "main_merge", "path": "immutable-main",
                                  "manifest": captured.get("immutableMain", {}).get("rawGitManifest", {})}
                if captured.get("completionTarget", "main_merge") != "main_merge":
                    raise ValueError("snapshot completion target differs from the frozen plan")
            assessment = CHECKER.assess(assessment_path, case, number)
            save(audit / f"assessment-{stage}.json", assessment)
            candidate_stable = True
            if completion_target == "workspace_snapshot":
                candidate_stable = _workspace_manifest(assessment_path) == candidate_info["manifest"]
            row = {"station": stage, "nativeState": status["state"],
                   "completionTarget": completion_target,
                   "assessmentCandidate": candidate_info,
                   "assessmentCandidateUnchangedAfterPublicChecks": candidate_stable,
                   "mainCommit": captured["immutableMain"]["commit"], "publicChecks": assessment,
                   "snapshotBindingSha256": digest(audit / f"snapshot-{stage}" / "binding.json"),
                   "runtimeFailure": _runtime_error(status)}
            main_after_capture = git(repo, "rev-parse", "main")
            integration = _integration_check(repo, captured, main_before_station, main_after_capture)
            row["integration"] = integration
            result["stations"].append(row)
            print(json.dumps({"event": "station-captured", "case": case, "station": stage,
                              "nativeState": status["state"], "runtimeFailure": bool(row["runtimeFailure"]),
                              "checks": {key: sum(item["status"] == key for item in assessment["findings"])
                                         for key in ("PASS", "FAIL", "NOT RUN", "EVALUATION_ERROR")}}), flush=True)
            if status["state"] != "completed" or row["runtimeFailure"]:
                lifecycle.freeze(repo, audit, reason="terminal adapter/runtime failure; no automatic replay")
                result["status"] = "runtime_failed"
                break
            integration_complete = (integration["mainAdvanced"] and integration["mainRefStableAfterCapture"] and
                                    integration["selectedHeadIntegratedInMain"] and
                                    integration["worktreeAndIndexCleanExceptControllerStationDelta"])
            snapshot_complete = (candidate_info.get("kind") == "workspace_snapshot" and
                                 bool(candidate_info.get("manifest")) and
                                 candidate_stable and
                                 _all_public_checks_pass(assessment))
            if not (snapshot_complete if completion_target == "workspace_snapshot" else integration_complete):
                lifecycle.freeze(repo, audit, reason=("workspace snapshot lacked a stable candidate or complete public checks"
                                                       if completion_target == "workspace_snapshot" else
                                                       "station did not produce a clean integrated main advancement"))
                result.update(status="station_incomplete", stopStudy=True,
                              reason=("workspace snapshot is missing stable candidate bytes or public checks did not all pass"
                                      if completion_target == "workspace_snapshot" else
                                      "native completion is not proof of integrated clean work"))
                break
            ownership.update(active=False, disposition="terminal completed station")
            parent = status["runId"]
            main_before_station = captured["immutableMain"]["commit"]
            if number < 4:
                lifecycle.advance(repo, audit)
            else:
                frozen = lifecycle.freeze(repo, audit, reason="four declared stations completed; independent final assessment follows")
                if completion_target == "workspace_snapshot":
                    frozen_candidate = frozen.get("assessmentCandidate")
                    if (frozen.get("completionTarget") != completion_target or
                            not isinstance(frozen_candidate, dict) or not frozen_candidate.get("manifest")):
                        raise ValueError("final workspace freeze lacks a bound assessment candidate")
                    final_candidate = (audit / "final-freeze" / frozen_candidate["path"]).resolve()
                    if _workspace_manifest(final_candidate) != frozen_candidate["manifest"]:
                        raise ValueError("final workspace candidate differs from its bound manifest")
                    result.update(status="trajectory_completed", completionTarget=completion_target,
                                  assessmentCandidate=frozen_candidate,
                                  finalCandidatePath=str(final_candidate))
                else:
                    result.update(status="trajectory_completed", finalMainCommit=frozen["immutableMain"]["commit"],
                                  finalMainPath=str(audit / "final-freeze" / "immutable-main"))

        _close(adapter, audit, result, ownership_unresolved=ownership["active"])
        if result.get("status") == "trajectory_completed" and result.get("closeDisposition", {}).get("state") == "closed":
            try:
                import final_assessor
                assessor_expires = min(expires, now() + timedelta(
                    seconds=plan.get("freshFinalSeconds", 5400) if completion_target == "workspace_snapshot" else 5400))
                final_result = final_assessor.assess(
                    plan, choice, executable,
                    Path(result.get("finalCandidatePath", result.get("finalMainPath"))), audit, assessor_expires,
                    completion_target=completion_target)
            except Exception as exc:
                final_result = {"state": "uncertain", "reason": f"{type(exc).__name__}: {exc}",
                                "replay": "forbidden"}
            result["independentFinalAssessment"] = final_result
            if final_result.get("state") != "completed":
                result.update(status="final_assessment_unresolved", stopStudy=True,
                              reason="independent final assessment did not complete with a report")
            elif (completion_target == "workspace_snapshot" and
                  (final_result.get("candidateFilesUnchanged") is not True or
                   final_result.get("frozenInitialPublicRequirementsUnchanged") is not True or
                   final_result.get("initialPublicWorkspaceBindingUnchanged") is not True)):
                result.update(status="final_assessment_unresolved", stopStudy=True,
                              reason="candidate or frozen initial requirements changed during final assessment")
    except Exception as exc:
        result.update(status="controller_error", reason=f"{type(exc).__name__}: {exc}",
                      outcome="preserved without replay")
        if prepared:
            try:
                lifecycle.record_execution(audit, metadata={"controllerError": result["reason"],
                                                            "ownership": ownership})
            except Exception:
                pass
        _close(adapter, audit, result, ownership_unresolved=ownership["active"])
    result["endedAt"] = now().isoformat()
    save(case_root / "trajectory-result.json", result)
    if prepared:
        save(audit / "trajectory-result.json", result)
    return result


def _validate_plan(plan):
    if not isinstance(plan, dict) or plan.get("schema") != 1 or plan.get("execution_authorized") is not True:
        raise ValueError("study plan must be an explicitly authorized schema 1 object")
    trajectories = plan.get("trajectories")
    if not isinstance(trajectories, list) or not 1 <= len(trajectories) <= 2:
        raise ValueError("study plan must contain one or two finite trajectories")
    completion_target = plan.get("completionTarget", "main_merge")
    if completion_target not in {"main_merge", "workspace_snapshot"}:
        raise ValueError("completionTarget must be main_merge or workspace_snapshot")
    expected_overall = 14400 if completion_target == "workspace_snapshot" else 28800
    if (plan.get("jobWallSeconds") != 14400 or plan.get("turnWallSeconds") != 5400 or
            plan.get("overallWallSeconds") != expected_overall or plan.get("startAllowance") != 256 or
            plan.get("outerTurnsPerTrajectory") != 4):
        limits = "four-hour/256-start" if completion_target == "workspace_snapshot" else "four-hour/eight-hour/256-start"
        raise ValueError(f"study plan must bind the authorized {limits} limits")
    if completion_target == "workspace_snapshot":
        if (plan.get("implementationSeconds") != 10800 or plan.get("freshFinalSeconds") != 2700 or
                plan.get("captureAssessmentReserveSeconds") != 900):
            raise ValueError("workspace snapshot plan must bind the 3h/45m/15m implementation and assessment reserves")
        for key in ("absoluteEndUtc", "lastStartUtc"):
            value = plan.get(key)
            if not isinstance(value, str):
                raise ValueError(f"workspace snapshot plan must bind {key}")
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
            if parsed.tzinfo is None:
                raise ValueError(f"workspace snapshot {key} must include a UTC offset")
    if not isinstance(plan.get("sourceRoots"), list) or not plan["sourceRoots"]:
        raise ValueError("study plan must declare common sourceRoots for execution pins")
    if not isinstance(plan.get("sourceFiles", []), list):
        raise ValueError("study plan sourceFiles must be a list of relative file paths")
    if plan.get("completionTarget", "main_merge") not in {"main_merge", "workspace_snapshot"}:
        raise ValueError("completionTarget must be main_merge or workspace_snapshot")
    if not isinstance(plan.get("executableVersion"), str) or not plan["executableVersion"].strip():
        raise ValueError("study plan must bind the native executable version")
    for choice in trajectories:
        if (not isinstance(choice, dict) or not isinstance(choice.get("case"), str) or
                not isinstance(choice.get("method"), str) or not isinstance(choice.get("factory"), str) or
                not isinstance(choice.get("config"), dict) or
                not isinstance(choice.get("manifestSourceRoots"), list) or not choice["manifestSourceRoots"]):
            raise ValueError("each trajectory requires case, method, factory, config, and manifestSourceRoots")
        if choice["case"] not in lifecycle.CASES:
            raise ValueError("trajectory uses an unsupported public case")
        if choice["method"] != choice["method"].strip() or not choice["method"]:
            raise ValueError("trajectory method must be a nonempty stable identifier")
    return plan


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", type=Path, required=True)
    parser.add_argument("--destination", type=Path, required=True)
    parser.add_argument("--executable", type=Path, required=True)
    args = parser.parse_args(argv)
    plan = _validate_plan(json.loads(args.plan.read_text(encoding="utf-8")))
    if git(ROOT, "status", "--porcelain"):
        raise ValueError("source checkout must be clean before the study starts")
    checkout_root = Path(git(ROOT, "rev-parse", "--show-toplevel")).resolve()
    if not args.destination.is_absolute():
        raise ValueError("destination must be an absolute external path")
    destination = args.destination.resolve()
    if (destination.exists() or destination == checkout_root or checkout_root in destination.parents or
            destination in checkout_root.parents):
        raise ValueError("destination must be fresh and external to the entire source checkout")
    executable = args.executable.resolve()
    if not executable.is_file():
        raise ValueError("native executable path must exist before starting the study")
    from delivery_binding import verify
    authority_bytes = verify(plan, executable)
    from delivery_binding import reserve
    reserve(plan, destination)
    destination.mkdir(parents=True)
    (destination / "delivery-order.json").write_bytes(authority_bytes)
    source = git(ROOT, "rev-parse", "HEAD")
    started = now()
    overall = started + timedelta(seconds=plan["overallWallSeconds"])
    if plan.get("completionTarget", "main_merge") == "workspace_snapshot":
        absolute_end = datetime.fromisoformat(plan["absoluteEndUtc"].replace("Z", "+00:00"))
        if absolute_end.tzinfo is None:
            raise ValueError("workspace absoluteEndUtc must include a UTC offset")
        overall = min(overall, absolute_end.astimezone(timezone.utc))
    save(destination / "study-binding.json", {
        "plan": plan, "planSha256": digest(args.plan), "sourceCommit": source,
        "controllerSha256": digest(__file__), "executable": str(executable),
        "executableSha256": digest(executable), "startedAt": started.isoformat(),
        "overallExpiresAt": overall.isoformat(), "sourceCheckout": str(checkout_root)})
    results = []
    save(destination / "study-results.json", {"trajectories": results, "status": "running"})
    for choice in plan["trajectories"]:
        if now() >= overall:
            break
        result = execute(plan, choice, destination, executable, source, overall)
        results.append(result)
        save(destination / "study-results.json", {
            "trajectories": results, "status": "complete" if len(results) == len(plan["trajectories"]) and all(
                row.get("status") == "trajectory_completed" and not row.get("stopStudy") for row in results) else "stopped",
            "interpretation": "adapter-specific functional evidence; no automatic method or transport winner"})
        if result.get("status") != "trajectory_completed" or result.get("stopStudy"):
            break
    final = {"event": "study-ended", "results": results,
             "endedAt": now().isoformat(), "overallExpiresAt": overall.isoformat()}
    save(destination / "study-ended.json", final)
    print(json.dumps(final, ensure_ascii=False), flush=True)
    return 0 if len(results) == len(plan["trajectories"]) and all(
        row["status"] == "trajectory_completed" for row in results) else 1


if __name__ == "__main__":
    for stream in (sys.stdin, sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(encoding="utf-8")
    raise SystemExit(main())
