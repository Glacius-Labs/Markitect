"""Conventional variant adapter for the shared work-item Playground contract.

This adapter owns only Conventional setup and native runtime lifecycle. The
Playground still owns cases, budgets, snapshots, and independent assessment.
"""
from __future__ import annotations

from datetime import datetime, timezone
import base64
import hashlib
import json
import os
from pathlib import Path
import queue
import subprocess
import threading
import time
import uuid
from typing import Any, Callable

from .service import Service


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _envelope(state: str, **fields: Any) -> dict[str, Any]:
    return {"schema": 1, "state": state, **fields}


def _durable_json(path: Path, value: dict[str, Any]) -> None:
    temp = path.with_name(path.name + "." + uuid.uuid4().hex + ".tmp")
    data = (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
    with temp.open("xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temp, path)


def _bounded_stop_and_capture(proc: subprocess.Popen[bytes], timeout_error: subprocess.TimeoutExpired) -> tuple[bytes, bytes]:
    """Stop only the direct child and keep timeout cleanup bounded."""
    try:
        proc.kill()
    except OSError:
        pass
    try:
        proc.wait(timeout=2)
    except subprocess.TimeoutExpired:
        pass
    out = timeout_error.output or b""
    err = timeout_error.stderr or b""
    for stream in (proc.stdout, proc.stderr):
        if stream is not None:
            try:
                stream.close()
            except OSError:
                pass
    if isinstance(out, str):
        out = out.encode("utf-8", "backslashreplace")
    if isinstance(err, str):
        err = err.encode("utf-8", "backslashreplace")
    return out, err


class ConventionalAdapter:
    """One config-bound Conventional controller implementing the generic API."""

    def __init__(self, config_path: str | Path, *, service: Service | None = None,
                 runtime_probe: Callable[[dict[str, Any]], dict[str, Any]] | None = None):
        self.config_path = Path(config_path).resolve()
        self._service = service if service is not None else Service(self.config_path)
        self._runtime_probe = runtime_probe
        self._config = self._service.config
        self._source_root = Path(__file__).resolve().parents[1]
        self._setup_context: dict[str, Any] | None = None
        self._setup_receipt: dict[str, Any] | None = None
        self._closed = False
        self._manifest_source_pins: dict[str, str] | None = None
        pins = self._config.get("filePins", {})
        expected = pins.get(str(Path(__file__).resolve())) if isinstance(pins, dict) else None
        if expected != _sha(Path(__file__).resolve()):
            raise ValueError("filePins must bind the exact Conventional adapter source")

    def describe(self) -> dict[str, Any]:
        backend = self._config.get("backend")
        if backend == "codex-app-server":
            runtime = {"kind": backend, "owner": "adapter", "scope": "owned stdio direct child"}
        elif backend == "codex-cli":
            runtime = {"kind": backend, "owner": "adapter", "scope": "owned native exec direct child"}
        else:
            runtime = {"kind": str(backend), "owner": "none", "scope": "unsupported"}
        source = Path(__file__).resolve()
        pins = self._manifest_source_pins or {str(source): _sha(source)}
        return {"schema": 1, "id": "conventional", "method": "Conventional", "version": "1",
                "sourcePins": dict(pins),
                "capabilities": {"ensure_runtime": True, "start": True, "resume": True,
                                 "status": True, "cancel": True, "close": True},
                "runtime": runtime}

    def bind_manifest_source_pins(self, source_pins: dict[str, str]) -> None:
        """Bind the generic loader's already-verified manifest source set."""
        if not isinstance(source_pins, dict) or not source_pins:
            raise ValueError("manifest sourcePins must be a nonempty object")
        normalized: dict[str, str] = {}
        for name, expected in source_pins.items():
            path = Path(name).resolve()
            if not path.is_file() or not isinstance(expected, str) or _sha(path) != expected:
                raise ValueError("manifest source pin does not match current bytes: " + name)
            normalized[str(path)] = expected
        if str(Path(__file__).resolve()) not in normalized:
            raise ValueError("manifest sourcePins must include conventional adapter.py")
        self._manifest_source_pins = normalized

    def setup(self, context: dict[str, Any]) -> dict[str, Any]:
        if self._closed:
            return _envelope("blocked", reason="adapter is closed")
        if not isinstance(context, dict) or context.get("schema") != 1:
            return _envelope("blocked", reason="setup context must use schema 1")
        if context.get("method") != "Conventional":
            return _envelope("blocked", reason="Conventional adapter requires method Conventional")
        setup_started = context.get("setupStartedAt") or _now()
        try:
            repo = self._absolute_dir(context.get("repoPath"), "repoPath")
            audit = self._absolute_dir(context.get("auditPath"), "auditPath")
            if repo == audit or repo in audit.parents or audit in repo.parents:
                raise ValueError("repoPath and auditPath must be disjoint")
            if self._absolute_dir(self._config.get("cwd"), "configured cwd") != repo:
                raise ValueError("setup repoPath does not match frozen configured cwd")
            run = json.loads((audit / "run.json").read_text(encoding="utf-8"))
            run_id = (repo / ".study/run-id").read_text(encoding="ascii").strip()
            if run.get("method") != "Conventional" or run.get("runId") != run_id:
                raise ValueError("repo is not a matching lifecycle.prepare Conventional checkout")
            source_root = Path(context.get("setupSourceRoot", self._source_root / "public" / "conventional")).resolve()
            fragment = source_root / "AGENTS.fragment.md"
            if not fragment.is_file():
                raise ValueError("ordinary Conventional AGENTS fragment is missing")
            profile = context.get("profile")
            if not isinstance(profile, dict):
                raise ValueError("setup context requires a profile object")
            model = profile.get("model", self._config.get("model"))
            effort = profile.get("effort", self._config.get("effort"))
            backend = profile.get("nativeBackend", self._config.get("backend"))
            if (model != self._config.get("model") or effort != self._config.get("effort") or
                    backend != self._config.get("backend")):
                raise ValueError("setup profile differs from the frozen runtime configuration")
            profile_path = repo / ".study/runtime-profile.json"
            agents_path = repo / "AGENTS.md"
            marker = "<!-- playground-adapter:conventional:v1 -->"
            fragment_text = fragment.read_text(encoding="utf-8").rstrip()
            current_agents = agents_path.read_text(encoding="utf-8") if agents_path.exists() else ""
            if marker in current_agents:
                receipt_path = audit / "adapter-setup.json"
                receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
                self._verify_setup_receipt(receipt, repo, agents_path, profile_path, fragment)
                self._setup_context = dict(context)
                self._setup_receipt = receipt
                return _envelope("ready", receipt=receipt, observations=[self._observation("adapter.warning", {"message": "setup already installed; verified existing receipt"})])
            if (repo / ".git").exists() is False:
                raise ValueError("setup requires the Git repository created by lifecycle.prepare")
            if self._git(repo, "branch", "--show-current") != "main":
                raise ValueError("Conventional setup must be committed on main")
            if self._git(repo, "status", "--porcelain"):
                raise ValueError("refusing to install setup over a dirty prepared checkout")
            runtime_options = self._config.get("runtimeOptions") or {}
            profile_doc = {"schema": 1, "method": "Conventional", "model": model, "effort": effort,
                           "nativeBackend": backend, "nativeHelperModel": runtime_options.get("nativeHelperModel", model),
                           "nativeHelperEffort": runtime_options.get("nativeHelperEffort", effort),
                           "sandbox": runtime_options.get("sandbox"),
                           "approvalPolicy": runtime_options.get("approvalPolicy"),
                           "memoryEnabled": runtime_options.get("memoryEnabled"),
                           "windowsSandbox": runtime_options.get("windowsSandbox"),
                           "scopedGitApproval":runtime_options.get("scopedGitApproval",False),
                           "gitApprovalShell":runtime_options.get("gitApprovalShell"),
                           "scopedGitRequiredArgvPrefix":["-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"] if runtime_options.get("scopedGitApproval") else None,
                           "scopedGitConfigurationBoundary":"child-local system/global config excluded; inherited Git environment removed; only inert owned local config accepted" if runtime_options.get("scopedGitApproval") else None,
                           "allowLoginShell":runtime_options.get("allowLoginShell"),
                           "nativeMaxConcurrentAgents": runtime_options.get("nativeMaxConcurrentAgents"),
                           "nativeMaxDepth": "unsupported by documented Codex config; unknown",
                           "studyLimits": dict(profile),
                           "setupContext": {key: context[key] for key in ("case", "station") if key in context}}
            profile_bytes = (json.dumps(profile_doc, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
            agents_bytes = (current_agents.rstrip() + "\n\n" + marker + "\n" + fragment_text + "\nThe installed finite runtime profile is .study/runtime-profile.json. Keep its model, effort and shared limits for every native role.\n").encode("utf-8")
            if runtime_options.get("scopedGitApproval"):
                agents_bytes += ("For Git metadata changes, use the exact absolute pinned Git executable from gitApprovalShell and scopedGitRequiredArgvPrefix before the single Git action. The approval broker recognizes only one owned-repository action per request; a PowerShell wrapper must use the pinned executable with -NoProfile and exactly one leading call operator (&) for the quoted Git executable. System/global Git configuration is excluded in this runtime; any local config outside the inert allowlist blocks approval. This configuration is an explicit study approximation.\n").encode("utf-8")
            agents_path.write_bytes(agents_bytes)
            profile_path.write_bytes(profile_bytes)
            self._git(repo, "add", "AGENTS.md", ".study/runtime-profile.json")
            checked = subprocess.run(["git", "-C", str(repo), "diff", "--cached", "--check"],
                                     capture_output=True, timeout=60)
            if checked.returncode:
                raise ValueError("cached setup diff failed whitespace check: " + checked.stderr.decode("utf-8", "replace"))
            self._git(repo, "commit", "-m", "Install ordinary Conventional rules and runtime profile")
            receipt = {"schema": 1, "method": "Conventional", "runId": run_id,
                       "repoPath": str(repo), "auditPath": str(audit), "setupCommit": self._git(repo, "rev-parse", "HEAD"),
                       "setupStartedAt": setup_started, "setupEndedAt": _now(),
                       "sourcePins": self.describe()["sourcePins"],
                       "setupAssets": {str(fragment): _sha(fragment), str(agents_path): _sha(agents_path),
                                       str(profile_path): _sha(profile_path)},
                       "runtimeProfile": profile_doc, "runtimeReadiness": "not established"}
            receipt_path = audit / "adapter-setup.json"
            receipt_path.parent.mkdir(parents=True, exist_ok=True)
            receipt_path.write_bytes((json.dumps(receipt, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
            self._setup_context = dict(context)
            self._setup_receipt = receipt
            return _envelope("ready", receipt=receipt,
                             observations=[self._observation("adapter.warning", {"message": "setup committed; runtime readiness remains untested"})])
        except (OSError, ValueError, subprocess.SubprocessError, KeyError, TypeError) as exc:
            return _envelope("blocked", reason=f"{type(exc).__name__}: {exc}")

    def ensure_runtime(self) -> dict[str, Any]:
        if self._closed:
            return _envelope("blocked", reason="adapter is closed")
        try:
            # Admission validates the exact immutable configuration, external order,
            # prepared run and source pins before any native process is created.
            _, spec, order, binding = self._service._admit()
            probe_allowed = order.get("runtimeChecksAuthorized") is True
            if not probe_allowed:
                return _envelope("blocked", reason="external order does not authorize native runtime startup checks")
            if self._runtime_probe is not None:
                observed = self._runtime_probe(spec)
                if not isinstance(observed, dict):
                    raise ValueError("runtime probe returned no evidence object")
                state = "ready" if observed.get("protocolReady") is True and observed.get("toolReady") is True else "blocked"
                return _envelope(state, runtime=observed,
                                 observations=[self._observation("runtime.ready" if state == "ready" else "runtime.blocked", observed)])
            backend = spec["backend"]
            version = None
            if backend == "codex-app-server":
                protocol = self._probe_app_server(spec, order, binding)
                if protocol.get("processTerminalConfirmed") is False:
                    return _envelope("uncertain", runtime=protocol, reason="owned readiness process terminal is unresolved; no further checks")
            elif backend == "codex-cli":
                version = self._probe_cli_version(spec, order, binding)
                if not version.get("versionReady"):
                    return _envelope("blocked", runtime={"version": version, "sandboxReady": "NOT RUN",
                                                           "toolReady": False},
                                     reason="native Codex CLI version probe failed")
                protocol = {"protocolReady": "not exercised", "probe": "native exec JSONL protocol not started"}
            else:
                return _envelope("blocked", reason="unsupported native backend")
            sandbox = self._probe_native_sandbox(spec, order, binding)
            if sandbox.get("processTerminalConfirmed") is False:
                return _envelope("uncertain", runtime=sandbox, reason="owned sandbox probe terminal is unresolved")
            ready = sandbox.get("sandboxReady") is True and (
                protocol.get("protocolReady") is True if backend == "codex-app-server"
                else bool(version and version.get("versionReady")))
            git_verified=sandbox.get("gitMetadataReady") is True
            broker_configured=(backend=="codex-app-server" and (spec.get("runtimeOptions") or {}).get("scopedGitApproval") is True and (spec.get("runtimeOptions") or {}).get("approvalPolicy")=="on-request")
            observed = {"backend": backend, "version": version, **protocol, **sandbox,
                        "gitMergeReadiness":"verified metadata write" if git_verified else "conditional; real native approval not observed" if broker_configured else "blocked protected metadata",
                        "actualGitApprovalObserved":False,"scopedGitApprovalConfigured":broker_configured,
                        "toolReady": ready, "nativeTurnStarted": False,
                        "ownedProcessScope": "direct-child-only"}
            ready=ready and (git_verified or broker_configured)
            return _envelope("ready" if ready else "blocked", runtime=observed,
                             observations=[self._observation("runtime.ready" if ready else "runtime.blocked", observed)])
        except Exception as exc:
            state = "uncertain" if (Path(self._config["audit"])/"adapter-runtime-checks/active.json").exists() else "blocked"
            return _envelope(state, reason=f"{type(exc).__name__}: {exc}")

    def start(self, prompt: str) -> dict[str, Any]:
        return self._submit("start", prompt)

    def resume(self, run_id: str, prompt: str) -> dict[str, Any]:
        return self._submit("resume", prompt, run_id)

    def _submit(self, action: str, prompt: str, run_id: str | None = None) -> dict[str, Any]:
        if self._closed:
            return _envelope("blocked", reason="adapter is closed")
        try:
            result = self._service.start(prompt) if action == "start" else self._service.resume(run_id, prompt)
            state = result.get("state")
            if state in {"starting", "running"}:
                state = "accepted"
            return _envelope(state or "uncertain", runId=result.get("runId"), nativeSessionId=result.get("nativeSessionId"),
                             nativeTurnId=result.get("nativeTurnId"), taskAssessment="NOT RUN",
                             humanAcceptance="not established", observations=[self._observation("attempt.accepted", {
                                 "runId": result.get("runId"), "backend": result.get("backend")})])
        except Exception as exc:
            return _envelope("blocked", reason=f"{type(exc).__name__}: {exc}")

    def status(self, run_id: str) -> dict[str, Any]:
        try:
            result = self._service.status(run_id)
            runtime_failure = self._owned_runtime_failure(run_id)
            state = result.get("state")
            if state == "starting":
                state = "accepted"
            elif state == "running":
                state = "running"
            return _envelope(state or "uncertain", runId=run_id,
                             nativeSessionId=result.get("nativeSessionId"), nativeTurnId=result.get("nativeTurnId"),
                             usage=result.get("usage"), usageScope=result.get("usageScope"),
                             taskAssessment=result.get("taskAssessment", "NOT RUN"),
                             humanAcceptance=result.get("humanAcceptance", "not established"),
                             controllerOwnership=result.get("controllerOwnership"),
                             runtimeFailure=runtime_failure,
                             observations=[])
        except Exception as exc:
            return _envelope("failed", runId=run_id, reason=f"{type(exc).__name__}: {exc}")

    def _owned_runtime_failure(self, run_id: str) -> dict[str, str] | None:
        """Recognize one exact native tool failure in this Service-owned run only."""
        folder, _ = self._service._record(run_id)
        events = folder / "events.jsonl"
        if not events.is_file() or events.is_symlink():
            return None
        marker = b"helper_unknown_error: setup refresh had errors"
        if marker not in events.read_bytes():
            return None
        return {"kind": "native_tool_setup", "reason": marker.decode("ascii")}

    def cancel(self, run_id: str) -> dict[str, Any]:
        try:
            result = self._service.cancel(run_id)
            return _envelope(result.get("state", "uncertain"), runId=run_id,
                             cancelRequested=result.get("cancelRequested", False),
                             nativeSessionId=result.get("nativeSessionId"), nativeTurnId=result.get("nativeTurnId"),
                             observations=[])
        except Exception as exc:
            return _envelope("failed", runId=run_id, reason=f"{type(exc).__name__}: {exc}")

    def close(self) -> dict[str, Any]:
        if self._closed:
            return _envelope("closed", alreadyClosed=True)
        try:
            result = self._service.close()
            self._closed = True
            return _envelope("closed", service=result, observations=[self._observation("adapter.warning", {"message": "adapter closed; unresolved native ownership remains represented by service receipts"})])
        except Exception as exc:
            return _envelope("uncertain", reason=f"{type(exc).__name__}: {exc}")

    def _probe_cli_version(self, spec: dict[str, Any], order: dict[str, Any], binding: dict[str, Any]) -> dict[str, Any]:
        executable = spec["command"][0]
        receipt = self._reserve_runtime_check(order, binding, "codex-cli-version")
        observed: dict[str, Any] = {"checkId": receipt["checkId"], "kind": "codex-cli-version",
                                   "executable": executable, "versionReady": False, "nativeTurnStarted": False}
        argv = [*spec["command"], "--version"]
        observed.update({"argv": argv, "cwd": spec["cwd"], "startedAt": _now()})
        try:
            proc = subprocess.Popen(argv, cwd=spec["cwd"], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            observed.update({"pid": proc.pid, "dispatchOccurred": True})
            try:
                stdout_bytes, stderr_bytes = proc.communicate(timeout=30)
            except subprocess.TimeoutExpired as timeout_error:
                stdout_bytes, stderr_bytes = _bounded_stop_and_capture(proc, timeout_error)
                observed["timedOut"] = True
            stdout = stdout_bytes.decode("utf-8", "replace").strip()
            stderr = stderr_bytes.decode("utf-8", "replace").strip()
            observed.update({"endedAt": _now(), "exitCode": proc.returncode, "stdout": stdout, "stderr": stderr,
                             "stdoutBase64": base64.b64encode(stdout_bytes).decode("ascii"),
                             "stdoutByteLength": len(stdout_bytes),
                             "stderrBase64": base64.b64encode(stderr_bytes).decode("ascii"),
                             "stderrByteLength": len(stderr_bytes),
                             "versionReady": proc.returncode == 0 and bool(stdout)})
        except (OSError, subprocess.SubprocessError) as exc:
            observed.update({"dispatchOccurred": "pid" in observed,
                             "disposition": "unknown-after-launch" if "pid" in observed else "known-not-dispatched",
                             "endedAt": _now(),
                             "error": f"{type(exc).__name__}: {exc}"})
        observed["processTerminalConfirmed"] = not observed.get("dispatchOccurred") or (
            "pid" in observed and proc.poll() is not None)
        self._finish_runtime_check(receipt, observed, release=observed["processTerminalConfirmed"])
        return observed

    def _probe_native_sandbox(self, spec: dict[str, Any], order: dict[str, Any], binding: dict[str, Any],
                              *, windows_host: bool | None = None) -> dict[str, Any]:
        options = spec.get("runtimeOptions") or {}
        sandbox = options.get("windowsSandbox")
        if not (os.name == "nt" if windows_host is None else windows_host):
            return {"sandboxReady": False, "probe": "Windows Codex sandbox probe requires Windows"}
        if sandbox not in {"mxc", "elevated"}:
            return {"sandboxReady": False, "probe": "runtimeOptions.windowsSandbox must explicitly select mxc or elevated"}
        repo = Path(spec["cwd"]).resolve()
        token = uuid.uuid4().hex
        probe_name = ".playground-runtime-probe-" + token + ".txt"
        probe_path = repo / probe_name
        metadata_probe = repo / ".git" / ("playground-runtime-probe-" + token + ".tmp")
        payload = "playground-runtime-probe:" + token
        # PowerShell is passed as an argv item to Codex's native sandbox command;
        # user input and shell text are not interpolated into this script.
        escaped_repo = str(repo).replace("'", "''")
        escaped_probe = str(probe_path).replace("'", "''")
        escaped_metadata = str(metadata_probe).replace("'", "''")
        script = (
            "$ErrorActionPreference='Stop';"
            f"$p='{escaped_probe}';"
            f"[IO.File]::WriteAllText($p,'{payload}');"
            f"if([IO.File]::ReadAllText($p) -ne '{payload}'){{exit 41}};"
            "Write-Output 'PLAYGROUND_WRITE_READ_OK';"
            f"git -C '{escaped_repo}' status --porcelain *> $null;"
            "if($LASTEXITCODE -ne 0){exit 42};"
            f"$m='{escaped_metadata}';"
            f"$expectedGit=[IO.Path]::GetFullPath((Join-Path '{escaped_repo}' '.git'));"
            f"$actualGit=(git -C '{escaped_repo}' rev-parse --absolute-git-dir);"
            "if($LASTEXITCODE -ne 0 -or [IO.Path]::GetFullPath($actualGit) -ne $expectedGit){exit 50};"
            "if(([IO.Path]::GetDirectoryName($m)) -ne $expectedGit -or "
            f"[IO.Path]::GetFileName($m) -ne 'playground-runtime-probe-{token}.tmp'){{exit 51}};"
            "try{[IO.File]::WriteAllText($m,'owned metadata readiness');"
            "if([IO.File]::ReadAllText($m) -ne 'owned metadata readiness'){exit 52};"
            "Remove-Item -LiteralPath $m -Force;Write-Output 'PLAYGROUND_REPO_GIT_WRITE_OK'}"
            "catch{Write-Output 'PLAYGROUND_REPO_GIT_WRITE_BLOCKED'};"
            "Remove-Item -LiteralPath $p -Force;"
            "Write-Output 'PLAYGROUND_SANDBOX_PROBE_OK';"
        )
        receipt = self._reserve_runtime_check(order, binding, "codex-native-sandbox")
        command = [*spec["command"], "-c", f'windows.sandbox="{sandbox}"', "sandbox",
                   "--include-managed-config", "--permission-profile", ":workspace", "--",
                   "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script]
        observed: dict[str, Any] = {"checkId": receipt["checkId"], "kind": "codex-native-sandbox",
                                   "sandbox": sandbox, "probePath": str(probe_path),
                                   "sandboxReady": False, "nativeTurnStarted": False}
        started_at = _now()
        try:
            proc = subprocess.Popen(command, cwd=str(repo), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            observed.update({"argv": command, "cwd": str(repo), "pid": proc.pid, "startedAt": started_at,
                             "dispatchOccurred": True})
            try:
                stdout_bytes, stderr_bytes = proc.communicate(timeout=60)
                return_code = proc.returncode
            except subprocess.TimeoutExpired as timeout_error:
                stdout_bytes, stderr_bytes = _bounded_stop_and_capture(proc, timeout_error)
                return_code = proc.returncode
                observed["timedOut"] = True
            stdout = stdout_bytes.decode("utf-8", "replace").strip()
            stderr = stderr_bytes.decode("utf-8", "replace").strip()
            observed.update({"endedAt": _now(), "exitCode": return_code, "stdout": stdout, "stderr": stderr,
                             "stdoutBase64": base64.b64encode(stdout_bytes).decode("ascii"),
                             "stdoutByteLength": len(stdout_bytes),
                             "stderrBase64": base64.b64encode(stderr_bytes).decode("ascii"),
                             "stderrByteLength": len(stderr_bytes),
                             "processTerminalConfirmed": proc.poll() is not None,
                             "sandboxReady": return_code == 0 and "PLAYGROUND_SANDBOX_PROBE_OK" in stdout,
                             "writeReadReady": return_code == 0 and "PLAYGROUND_WRITE_READ_OK" in stdout,
                             "gitCommandReady": return_code == 0 and "PLAYGROUND_REPO_GIT_WRITE_OK" in stdout,
                             "gitMetadataReady": return_code == 0 and "PLAYGROUND_REPO_GIT_WRITE_OK" in stdout,
                             "gitMetadataProbePath":str(metadata_probe),
                             "gitReadinessScope":"exact prepared repository metadata; no branch/index/main mutation"})
        except (OSError, subprocess.SubprocessError) as exc:
            observed.update({"dispatchOccurred": "pid" in observed,
                             "disposition": "unknown-after-launch" if "pid" in observed else "known-not-dispatched",
                             "endedAt": _now(),
                             "error": f"{type(exc).__name__}: {exc}"})
        finally:
            # Cleanup is restricted to this uniquely named probe file under cwd.
            try:
                resolved_probe = probe_path.resolve()
                if resolved_probe.parent == repo and probe_path.name == probe_name and not probe_path.is_symlink():
                    probe_path.unlink(missing_ok=True)
            except OSError as exc:
                observed["cleanupError"] = type(exc).__name__
            try:
                if metadata_probe.parent.resolve()==(repo/".git").resolve() and (repo/".git").is_dir() and not (repo/".git").is_symlink():
                    metadata_probe.unlink(missing_ok=True)
            except OSError as exc:
                observed["metadataCleanupError"]=type(exc).__name__
            self._finish_runtime_check(receipt, observed,
                                       release=observed["processTerminalConfirmed"])
        return observed

    def _reserve_runtime_check(self, order: dict[str, Any], binding: dict[str, Any], kind: str) -> dict[str, Any]:
        limit = order.get("maxRuntimeStartupChecks")
        if type(limit) is not int or limit <= 0:
            raise ValueError("external order must set positive maxRuntimeStartupChecks")
        audit = Path(self._config["audit"]).resolve()
        store = audit / "adapter-runtime-checks"
        store.mkdir(exist_ok=True)
        lock = store / "active.json"
        try:
            with lock.open("xb") as stream:
                stream.write(json.dumps({"pid": os.getpid(), "kind": kind, "at": _now()}).encode("utf-8"))
                stream.flush()
                os.fsync(stream.fileno())
        except FileExistsError as exc:
            raise ValueError("another adapter runtime check is unresolved; do not retry") from exc
        try:
            checks = sorted(path for path in store.glob("check-*") if path.is_dir())
            if len(checks) >= min(limit, 6):
                raise ValueError("finite runtime startup check allowance exhausted")
            check_id = f"{len(checks) + 1:02d}-{uuid.uuid4().hex}"
            folder = store / ("check-" + check_id)
            folder.mkdir()
            receipt = {"schema": 1, "checkId": check_id, "kind": kind, "state": "reserved",
                       "reservedAt": _now(), "configSha256": binding["configSha256"],
                       "orderSha256": binding["orderSha256"], "runId": binding["runId"]}
            _durable_json(folder / "receipt.json", receipt)
            return {**receipt, "folder": str(folder), "activeLock": str(lock)}
        except Exception:
            lock.unlink(missing_ok=True)
            raise

    @staticmethod
    def _finish_runtime_check(reservation: dict[str, Any], observed: dict[str, Any], *, release: bool = True) -> None:
        receipt = {key: value for key, value in reservation.items() if key not in {"folder", "activeLock"}}
        receipt.update({"state": "finished" if release else "unresolved", "finishedAt": _now(), "observed": observed})
        folder = Path(reservation["folder"])
        _durable_json(folder / "receipt.json", receipt)
        if release:
            Path(reservation["activeLock"]).unlink(missing_ok=True)

    def _probe_app_server(self, spec: dict[str, Any], order: dict[str, Any], binding: dict[str, Any]) -> dict[str, Any]:
        from .backends import _runtime_config_args
        reservation = self._reserve_runtime_check(order, binding, "codex-app-server-initialize")
        args = [*spec["command"], *_runtime_config_args(spec.get("runtimeOptions"), spec["model"], spec["effort"]), "app-server"]
        timeout = max(60.0, float(spec.get("requestTimeoutSeconds", 60)))
        client_wire = (json.dumps({"id": 1, "method": "initialize", "params": {
            "clientInfo": {"name": "playground-conventional-adapter", "version": "1"}}},
            separators=(",", ":")) + "\n").encode("utf-8")
        observed: dict[str, Any] = {"checkId": reservation["checkId"], "backend": "codex-app-server",
                                   "argv": args, "cwd": spec["cwd"], "startedAt": _now(),
                                   "owner": "adapter", "ownedProcessScope": "direct-child-only",
                                   "protocolReady": False, "toolReady": "unknown", "nativeTurnStarted": False,
                                   "clientInitializeBase64": base64.b64encode(client_wire).decode("ascii"),
                                   "clientInitializeByteLength": len(client_wire)}
        try:
            proc = subprocess.Popen(args, cwd=spec["cwd"], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, bufsize=0)
        except OSError as exc:
            observed.update({"dispatchOccurred": False, "endedAt": _now(),
                             "processTerminalConfirmed": True,
                             "error": f"{type(exc).__name__}: {exc}"})
            self._finish_runtime_check(reservation, observed)
            return observed
        observed.update({"pid": proc.pid, "dispatchOccurred": True})
        messages: queue.Queue[tuple[str, bytes | None]] = queue.Queue()
        stdout_parts: list[bytes] = []
        stderr_parts: list[bytes] = []

        def read_stdout() -> None:
            assert proc.stdout is not None
            try:
                for line in iter(proc.stdout.readline, b""):
                    stdout_parts.append(line)
                    messages.put(("stdout", line))
            finally:
                messages.put(("stdout-eof", None))

        def read_stderr() -> None:
            assert proc.stderr is not None
            try:
                while True:
                    chunk = proc.stderr.read(4096)
                    if not chunk:
                        break
                    stderr_parts.append(chunk)
                    messages.put(("stderr", chunk))
            finally:
                messages.put(("stderr-eof", None))

        stdout_reader = threading.Thread(target=read_stdout, daemon=True)
        stderr_reader = threading.Thread(target=read_stderr, daemon=True)
        stdout_reader.start()
        stderr_reader.start()
        try:
            assert proc.stdin is not None
            proc.stdin.write(client_wire)
            proc.stdin.flush()
            deadline = time.monotonic() + timeout
            while time.monotonic() < deadline:
                try:
                    kind, line = messages.get(timeout=min(0.1, max(0.001, deadline - time.monotonic())))
                except queue.Empty:
                    if proc.poll() is not None:
                        break
                    continue
                if kind == "stdout-eof":
                    break
                if kind != "stdout" or line is None:
                    continue
                try:
                    message = json.loads(line)
                except (json.JSONDecodeError, UnicodeDecodeError):
                    continue
                if message.get("id") == 1:
                    if message.get("error") is None and isinstance(message.get("result"), dict):
                        observed["protocolReady"] = True
                        observed["initializeResultFields"] = sorted(message["result"].keys())
                        observed["initializeResponseBase64"] = base64.b64encode(line).decode("ascii")
                        observed["initializeResponseByteLength"] = len(line)
                    else:
                        observed["error"] = message.get("error")
                        observed["initializeResponseBase64"] = base64.b64encode(line).decode("ascii")
                        observed["initializeResponseByteLength"] = len(line)
                    break
            if observed["protocolReady"]:
                notification_wire = (json.dumps({"method": "initialized", "params": {}},
                                                 separators=(",", ":")) + "\n").encode("utf-8")
                observed["clientInitializedBase64"] = base64.b64encode(notification_wire).decode("ascii")
                observed["clientInitializedByteLength"] = len(notification_wire)
                proc.stdin.write(notification_wire)
                proc.stdin.flush()
        except (OSError, subprocess.SubprocessError) as exc:
            observed["error"] = f"{type(exc).__name__}: {exc}"
        finally:
            # This process is owned by the adapter; terminate only this exact child.
            if proc.poll() is None:
                try:
                    proc.terminate()
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    try:
                        proc.wait(timeout=2)
                    except subprocess.TimeoutExpired:
                        pass
            observed.update({"endedAt": _now(), "exitCode": proc.poll(),
                             "processTerminalConfirmed": proc.poll() is not None})
            stdout_reader.join(timeout=0.25)
            stderr_reader.join(timeout=0.25)
            stdout_bytes = b"".join(stdout_parts)
            stderr_bytes = b"".join(stderr_parts)
            observed.update({"stdoutBase64": base64.b64encode(stdout_bytes).decode("ascii"),
                             "stdoutByteLength": len(stdout_bytes),
                             "stderrBase64": base64.b64encode(stderr_bytes).decode("ascii"),
                             "stderrByteLength": len(stderr_bytes),
                             "stdout": stdout_bytes.decode("utf-8", "backslashreplace"),
                             "stderr": stderr_bytes.decode("utf-8", "backslashreplace")})
            for stream in (proc.stdin, proc.stdout, proc.stderr):
                if stream is not None:
                    try:
                        stream.close()
                    except OSError:
                        pass
            self._finish_runtime_check(reservation, observed,
                                       release=observed["processTerminalConfirmed"])
        return observed

    @staticmethod
    def _absolute_dir(value: Any, label: str) -> Path:
        path = Path(value)
        if not path.is_absolute() or not path.is_dir() or path.is_symlink():
            raise ValueError(f"{label} must be an existing non-symlink absolute directory")
        return path.resolve()

    @staticmethod
    def _git(repo: Path, *args: str) -> str:
        result = subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, timeout=60)
        return result.stdout.decode("utf-8", "replace").strip()

    @staticmethod
    def _observation(kind: str, data: dict[str, Any]) -> dict[str, Any]:
        return {"schema": 1, "type": kind, "observedAt": _now(), "data": data}

    @staticmethod
    def _verify_setup_receipt(receipt: dict[str, Any], repo: Path, agents: Path,
                              profile: Path, fragment: Path) -> None:
        assets = receipt.get("setupAssets", {})
        for path in (agents, profile, fragment):
            expected = assets.get(str(path))
            if expected is None or expected != _sha(path):
                raise ValueError("existing setup receipt no longer matches setup files: " + str(path))
        if receipt.get("repoPath") != str(repo):
            raise ValueError("existing setup receipt belongs to another repository")


def create_adapter(config_path: str | Path) -> ConventionalAdapter:
    """Manifest factory entry point."""
    return ConventionalAdapter(config_path)
