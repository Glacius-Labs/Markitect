"""Bounded native Codex execution transports for the Conventional arm.

This module records transport evidence only. A terminal runtime event does not
evaluate the work product or establish study success.
"""

from __future__ import annotations

import json
import base64
import math
import os
import queue
import subprocess
import threading
import time
from collections import deque
from typing import Any, Callable


Emit = Callable[[dict[str, Any]], None]
RUNTIME_OPTION_KEYS = {"sandbox", "approvalPolicy", "memoryEnabled",
                       "nativeHelperModel", "nativeHelperEffort"}


class BackendSpecError(ValueError):
    """The frozen backend specification is incomplete or unsafe to use."""


def validate_runtime_options(options: Any, model: str, effort: str) -> dict[str, Any] | None:
    """Validate the closed, optional ordinary-runtime policy binding."""
    if options is None:
        return None
    if not isinstance(options, dict) or set(options) - RUNTIME_OPTION_KEYS:
        raise BackendSpecError("runtimeOptions contains unknown fields")
    required = {"sandbox", "approvalPolicy", "memoryEnabled"}
    if not required <= set(options):
        raise BackendSpecError("runtimeOptions requires sandbox, approvalPolicy, and memoryEnabled")
    if not isinstance(options["sandbox"], str) or options["sandbox"] not in {"read-only", "workspace-write"}:
        raise BackendSpecError("runtimeOptions.sandbox must be read-only or workspace-write")
    if not isinstance(options["approvalPolicy"], str) or options["approvalPolicy"] not in {"never", "on-request"}:
        raise BackendSpecError("runtimeOptions.approvalPolicy must be never or on-request")
    if type(options["memoryEnabled"]) is not bool:
        raise BackendSpecError("runtimeOptions.memoryEnabled must be a boolean")
    if "nativeHelperModel" in options:
        if not isinstance(options["nativeHelperModel"], str) or options["nativeHelperModel"] != model:
            raise BackendSpecError("runtimeOptions.nativeHelperModel must equal the run model")
    if "nativeHelperEffort" in options:
        if not isinstance(options["nativeHelperEffort"], str) or options["nativeHelperEffort"] != effort:
            raise BackendSpecError("runtimeOptions.nativeHelperEffort must equal the run effort")
    return dict(options)


def _runtime_config_args(options: dict[str, Any] | None, model: str, effort: str) -> list[str]:
    if options is None:
        return []
    settings = [
        ("sandbox_mode", json.dumps(options["sandbox"])),
        ("approval_policy", json.dumps(options["approvalPolicy"])),
        ("features.memories", "true" if options["memoryEnabled"] else "false"),
    ]
    if "nativeHelperModel" in options:
        settings.append(("agents.default_subagent_model", json.dumps(model)))
    if "nativeHelperEffort" in options:
        settings.append(("agents.default_subagent_reasoning_effort", json.dumps(effort)))
    result: list[str] = []
    for key, value in settings:
        result.extend(["-c", f"{key}={value}"])
    return result


def _launch_event(emit: Emit, backend: str, proc: subprocess.Popen[bytes], cwd: str, stdio: str) -> None:
    metadata = {"kind": "native-launch", "backend": backend, "pid": proc.pid,
                "stdio": stdio, "cwd": cwd, "ownedProcessScope": "direct-child-only"}
    _emit(emit, backend, "launch", json.dumps(metadata, ensure_ascii=False, sort_keys=True), metadata)


def _runtime_binding_event(emit: Emit, session: str, thread_result: dict[str, Any]) -> None:
    thread = thread_result.get("thread") if isinstance(thread_result.get("thread"), dict) else thread_result
    fields = ("model", "reasoningEffort", "modelProvider", "cwd", "sandbox",
              "approvalPolicy", "instructionSources")
    reported = {field: thread[field] if field in thread else thread_result[field]
                for field in fields if field in thread or field in thread_result}
    metadata = {"kind": "native-runtime-binding", "nativeSessionId": session,
                "reported": reported, "claimScope": "native-reported fields only"}
    _emit(emit, "codex-app-server", "runtime-binding",
          json.dumps(metadata, ensure_ascii=False, sort_keys=True), metadata)


def _validate(spec: dict[str, Any], prompt: str, resume_id: str | None) -> tuple[str, list[str], str, str, str, float, float, dict[str, Any] | None]:
    if not isinstance(spec, dict):
        raise BackendSpecError("spec must be an object")
    backend = spec.get("backend")
    if backend not in {"codex-cli", "codex-app-server"}:
        raise BackendSpecError("backend must be codex-cli or codex-app-server")
    command = spec.get("command")
    if (not isinstance(command, list) or not command or
            any(not isinstance(part, str) or not part for part in command)):
        raise BackendSpecError("command must be a non-empty list of non-empty strings")
    if not os.path.isabs(command[0]):
        raise BackendSpecError("command executable must be an absolute path")
    cwd = spec.get("cwd")
    if not isinstance(cwd, str) or not os.path.isabs(cwd) or not os.path.isdir(cwd):
        raise BackendSpecError("cwd must be an existing absolute directory")
    model = spec.get("model")
    effort = spec.get("effort")
    if not isinstance(model, str) or not model.strip() or not isinstance(effort, str) or not effort.strip():
        raise BackendSpecError("model and effort must be non-empty strings")
    if effort not in {"none", "minimal", "low", "medium", "high", "xhigh"}:
        raise BackendSpecError("effort must be one of none, minimal, low, medium, high, xhigh")
    runtime_options = validate_runtime_options(spec.get("runtimeOptions"), model, effort)
    if not isinstance(prompt, str) or not prompt:
        raise BackendSpecError("prompt must be a non-empty string")
    if resume_id is not None and (not isinstance(resume_id, str) or not resume_id.strip()):
        raise BackendSpecError("resume_id must be a non-empty string when supplied")
    timeout = _positive_finite(spec.get("timeoutSeconds"), "timeoutSeconds")
    request_timeout = _positive_finite(spec.get("requestTimeoutSeconds", 300), "requestTimeoutSeconds")
    return backend, command, cwd, model, effort, timeout, request_timeout, runtime_options


def _positive_finite(value: Any, name: str) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise BackendSpecError(f"{name} must be a finite positive number")
    value = float(value)
    if not math.isfinite(value) or value <= 0:
        raise BackendSpecError(f"{name} must be a finite positive number")
    return value


def _emit(emit: Emit, source: str, stream: str, raw: str, parsed: Any = None,
          raw_bytes: bytes | None = None) -> None:
    event: dict[str, Any] = {"source": source, "stream": stream, "raw": raw}
    if raw_bytes is not None:
        event["rawBase64"] = base64.b64encode(raw_bytes).decode("ascii")
        event["rawByteLength"] = len(raw_bytes)
    if parsed is not None:
        event["parsed"] = parsed
    emit(event)


def _read_lines(stream: Any, output: queue.Queue[tuple[str, bytes | None]], kind: str = "line") -> None:
    try:
        for line in iter(stream.readline, b""):
            output.put((kind, line))
    finally:
        try:
            stream.close()
        finally:
            output.put((f"{kind}-eof", None))


def _get_nested(obj: Any, *keys: str) -> Any:
    value = obj
    for key in keys:
        if not isinstance(value, dict):
            return None
        value = value.get(key)
    return value


def _session_id(message: Any) -> str | None:
    if not isinstance(message, dict):
        return None
    for path in (("thread_id",), ("threadId",), ("session_id",), ("sessionId",),
                 ("thread", "id"), ("thread", "threadId"), ("session", "id")):
        value = _get_nested(message, *path)
        if isinstance(value, str) and value:
            return value
    return None


def _turn_id(message: Any) -> str | None:
    if not isinstance(message, dict):
        return None
    for path in (("turn_id",), ("turnId",), ("turn", "id"), ("turn", "turnId"),
                 ("result", "turn", "id"), ("params", "turn", "id"),
                 ("params", "turn", "turnId")):
        value = _get_nested(message, *path)
        if isinstance(value, str) and value:
            return value
    return None


def _usage(message: Any) -> Any:
    if not isinstance(message, dict):
        return None
    for path in (("usage",), ("tokenUsage",), ("turn", "usage"), ("result", "usage"),
                 ("params", "usage"), ("params", "tokenUsage"), ("params", "turn", "usage")):
        value = _get_nested(message, *path)
        if value is not None:
            return value
    return None


def _state_from_terminal(message: Any) -> tuple[str, Any]:
    status = None
    if isinstance(message, dict):
        status = (message.get("status") or _get_nested(message, "turn", "status") or
                  _get_nested(message, "params", "status") or
                  _get_nested(message, "params", "turn", "status"))
        event_type = message.get("type")
        method = message.get("method")
        if event_type in {"turn.completed", "turn.complete"}:
            if status in {"interrupted", "cancelled", "canceled"}:
                return "cancelled", status
            if status in {"failed", "error"}:
                return "failed", status
            if status in {None, "completed", "complete", "success"}:
                return "completed", status or "completed"
            return "uncertain", status
        if method == "turn/completed":
            if status in {"completed", "complete", "success"}:
                return "completed", status
            if status in {"failed", "error"}:
                return "failed", status
            if status in {"interrupted", "cancelled", "canceled"}:
                return "cancelled", status
            return "uncertain", status
        if event_type == "turn.failed" or method == "turn/failed":
            return "failed", status or "failed"
        if event_type in {"turn.interrupted", "turn.cancelled"} or method in {"turn/interrupted", "turn/cancelled"}:
            return "cancelled", status or "interrupted"
    if status in {"completed", "complete", "success"}:
        return "completed", status
    if status in {"interrupted", "cancelled", "canceled"}:
        return "cancelled", status
    return "uncertain", status


def _result(state: str, session: str | None, turn: str | None, usage: Any, detail: str) -> dict[str, Any]:
    return {"state": state, "nativeSessionId": session, "nativeTurnId": turn,
            "usage": usage, "usageScope": "native-reported; aggregate scope unknown",
            "detail": detail, "ownedProcessScope": "direct-child-only"}


def _deliver(proc: subprocess.Popen[bytes], wire: bytes, deadline: float,
             cancel: threading.Event | None = None) -> None:
    """A child that never reads stdin cannot bypass the finite turn window."""
    finished = threading.Event()
    errors: list[Exception] = []

    def write() -> None:
        try:
            assert proc.stdin is not None
            # Raw pipe writes may be partial; never resend bytes already written.
            remaining = memoryview(wire)
            while remaining:
                count = proc.stdin.write(remaining)
                if not count:
                    raise BrokenPipeError("native stdin accepted no bytes")
                remaining = remaining[count:]
            proc.stdin.flush()
        except Exception as exc:
            errors.append(exc)
        finally:
            finished.set()

    threading.Thread(target=write, daemon=True).start()
    while not finished.wait(0.05):
        if (cancel is not None and cancel.is_set()) or time.monotonic() >= deadline:
            raise OSError("native stdin delivery cancelled or timed out; partial dispatch is uncertain")
    if errors:
        raise OSError(str(errors[0])) from errors[0]


def _deliver_rpc(proc: subprocess.Popen[bytes], wire: bytes) -> None:
    _deliver(proc, wire, min(proc._wrapper_deadline, time.monotonic() + proc._wrapper_request_timeout),
             proc._wrapper_cancel)


def run(spec: dict[str, Any], prompt: str, resume_id: str | None,
        emit: Emit, cancel: threading.Event) -> dict[str, Any]:
    """Run or resume one ordinary Codex turn, preserving raw transport events.

    No retries are performed. The returned state describes the transport/run
    outcome only; callers must assess captured workspace state independently.
    """
    backend, command, cwd, model, effort, timeout, request_timeout, runtime_options = _validate(spec, prompt, resume_id)
    if not callable(emit) or not callable(getattr(cancel, "is_set", None)):
        raise BackendSpecError("emit must be callable and cancel must be a threading.Event")
    if backend == "codex-cli":
        return _run_cli(command, cwd, model, effort, timeout, prompt, resume_id, runtime_options, emit, cancel)
    return _run_app_server(spec, command, cwd, model, effort, timeout, request_timeout, runtime_options,
                           prompt, resume_id, emit, cancel)


def _run_cli(command: list[str], cwd: str, model: str, effort: str, timeout: float,
             prompt: str, resume_id: str | None, runtime_options: dict[str, Any] | None, emit: Emit,
             cancel: threading.Event) -> dict[str, Any]:
    args = list(command)
    args.extend(["exec"])
    if resume_id is not None:
        args.extend(["resume", resume_id])
    args.extend(["--json", "--model", model, "-c", f'model_reasoning_effort="{effort}"'])
    args.extend(_runtime_config_args(runtime_options, model, effort))
    args.append("-")
    try:
        proc = subprocess.Popen(args, cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, bufsize=0)
    except OSError as exc:
        return _result("failed", resume_id, None, None, f"could not start Codex CLI: {exc}")
    _launch_event(emit, "codex-cli", proc, cwd, "stdin/stdout/stderr JSONL")
    deadline = time.monotonic() + timeout
    try:
        assert proc.stdin is not None
        _deliver(proc, prompt.encode("utf-8"), deadline, cancel)
        proc.stdin.close()
    except OSError as exc:
        _stop_process(proc)
        _close_pipes(proc)
        return _result("uncertain", resume_id, None, None, f"prompt delivery failed: {exc}")

    lines: queue.Queue[tuple[str, bytes | None]] = queue.Queue()
    assert proc.stdout is not None and proc.stderr is not None
    threading.Thread(target=_read_lines, args=(proc.stdout, lines, "stdout"), daemon=True).start()
    threading.Thread(target=_read_lines, args=(proc.stderr, lines, "stderr"), daemon=True).start()
    cancel_grace = 5.0
    session = resume_id
    turn: str | None = None
    usage = None
    terminal: tuple[str, Any] | None = None
    stdout_eof = False
    stderr_eof = False
    cancelled = False
    identity_mismatch = False
    cancel_at: float | None = None
    while True:
        now = time.monotonic()
        if cancel.is_set() and not cancelled:
            cancelled = True
            cancel_at = now
            _emit(emit, "codex-cli", "control", "cancel requested")
            _terminate(proc)
        if cancelled and cancel_at is not None and now - cancel_at >= min(cancel_grace, timeout):
            _stop_process(proc)
            return _result("uncertain", session, turn, usage,
                           "CLI cancellation stopped only the direct child; native turn interruption was not confirmed")
        if now >= deadline:
            _stop_process(proc)
            return _result("uncertain", session, turn, usage,
                           "CLI deadline expired; output pipes or descendant work may remain open")
        try:
            kind, raw = lines.get(timeout=0.05)
        except queue.Empty:
            if stdout_eof and stderr_eof and proc.poll() is not None:
                break
            continue
        if kind == "stdout-eof":
            stdout_eof = True
        elif kind == "stderr-eof":
            stderr_eof = True
        elif kind in {"stderr", "stdout"} and raw is not None:
            text = raw.decode("utf-8", errors="backslashreplace")
            if kind == "stderr":
                _emit(emit, "codex-cli", "stderr", text, raw_bytes=raw)
            else:
                try:
                    parsed = json.loads(raw)
                except (json.JSONDecodeError, UnicodeDecodeError):
                    parsed = None
                _emit(emit, "codex-cli", "stdout", text, parsed, raw)
                if isinstance(parsed, dict):
                    observed_session = _session_id(parsed)
                    observed_turn = _turn_id(parsed)
                    event_type = parsed.get("type")
                    session_matches = (not identity_mismatch and
                                       (not session or not observed_session or observed_session == session))
                    if event_type == "thread.started" and observed_session:
                        if session is None:
                            session = observed_session
                        else:
                            session_matches = observed_session == session
                            if not session_matches:
                                identity_mismatch = True
                    if identity_mismatch:
                        session_matches = False
                    if event_type == "turn.started" and observed_turn and session_matches:
                        if turn is None:
                            turn = observed_turn
                        else:
                            session_matches = session_matches and observed_turn == turn
                    if (event_type in {"turn.completed", "turn.failed", "turn.cancelled", "turn.interrupted"}
                            and observed_turn and session_matches):
                        if turn is None:
                            turn = observed_turn
                        else:
                            session_matches = session_matches and observed_turn == turn
                    usage = _usage(parsed) if _usage(parsed) is not None else usage
                    if (event_type in {"turn.completed", "turn.failed", "turn.cancelled", "turn.interrupted"}
                            and session_matches):
                        state, status = _state_from_terminal(parsed)
                        terminal = (state, status)
        if stdout_eof and stderr_eof and proc.poll() is not None:
            break
    try:
        code = proc.wait(timeout=1)
    except subprocess.TimeoutExpired:
        _stop_process(proc)
        return _result("uncertain", session, turn, usage,
                       "CLI emitted EOF but its direct child did not exit")
    _close_pipes(proc)
    if identity_mismatch:
        return _result("uncertain", session, turn, usage,
                       "CLI reported a different native thread id; the requested session was not confirmed")
    if cancelled:
        if terminal and terminal[0] == "cancelled" and session and turn:
            return _result("cancelled", session, turn, usage,
                           f"CLI confirmed interruption for its native turn: {terminal[1]}")
        return _result("uncertain", session, turn, usage,
                       "CLI process stopped after cancellation request; exact native turn interruption was not confirmed")
    if terminal is None:
        return _result("uncertain", session, turn, usage,
                       f"CLI exited with code {code} without a terminal turn event")
    state, status = terminal
    if code != 0 and state == "completed":
        return _result("failed", session, turn, usage,
                       f"CLI reported completed but exited with code {code}")
    return _result(state, session, turn, usage, f"CLI terminal status: {status}")


def _terminate(proc: subprocess.Popen[bytes]) -> None:
    if proc.poll() is None:
        try:
            proc.terminate()
        except OSError:
            pass


def _stop_process(proc: subprocess.Popen[bytes]) -> None:
    _terminate(proc)
    try:
        proc.wait(timeout=2)
    except subprocess.TimeoutExpired:
        try:
            proc.kill()
        except OSError:
            pass
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            pass
    # Reader threads may still be blocked because a descendant inherited a
    # pipe. Closing their buffered stream from this thread can itself block.
    # Close stdin after stopping the direct child; daemon readers release their
    # output handles once EOF arrives. Process ownership is explicitly limited
    # to the direct child.
    stream = proc.stdin
    if stream is not None and not stream.closed:
        try:
            stream.close()
        except OSError:
            pass


def _close_pipes(proc: subprocess.Popen[bytes]) -> None:
    for name in ("stdin", "stdout", "stderr"):
        stream = getattr(proc, name, None)
        if stream is not None and not stream.closed:
            try:
                stream.close()
            except OSError:
                pass


def _rpc(proc: subprocess.Popen[bytes], request_id: int, method: str, params: dict[str, Any] | None,
         emit: Emit) -> None:
    message: dict[str, Any] = {"id": request_id, "method": method}
    if params is not None:
        message["params"] = params
    raw = json.dumps(message, ensure_ascii=False, separators=(",", ":"))
    assert proc.stdin is not None
    wire = (raw + "\n").encode("utf-8")
    _deliver_rpc(proc, wire)
    _emit(emit, "codex-app-server", "client", raw, message, wire)


def _run_app_server(spec: dict[str, Any], command: list[str], cwd: str, model: str, effort: str,
                    timeout: float, request_timeout: float, runtime_options: dict[str, Any] | None, prompt: str,
                    resume_id: str | None, emit: Emit,
                    cancel: threading.Event) -> dict[str, Any]:
    # Keep ordinary runtime defaults until individual permission options have
    # been reviewed and bound in the frozen run specification.
    thread_options = spec.get("threadOptions", {})
    turn_options = spec.get("turnOptions", {})
    if not isinstance(thread_options, dict) or thread_options:
        raise BackendSpecError("threadOptions must be empty until permission choices are frozen and reviewed")
    if not isinstance(turn_options, dict) or turn_options:
        raise BackendSpecError("turnOptions must be empty; turn identity and instructions are runtime-owned")
    args = [*command, *_runtime_config_args(runtime_options, model, effort), "app-server"]
    try:
        proc = subprocess.Popen(args, cwd=cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, bufsize=0)
    except OSError as exc:
        return _result("failed", resume_id, None, None, f"could not start App Server: {exc}")
    _launch_event(emit, "codex-app-server", proc, cwd, "stdin/stdout/stderr JSONL")

    messages: queue.Queue[tuple[str, Any]] = queue.Queue()

    def read_stdout() -> None:
        assert proc.stdout is not None
        try:
            for line in iter(proc.stdout.readline, b""):
                raw = line.decode("utf-8", errors="backslashreplace")
                try:
                    parsed = json.loads(line)
                except (json.JSONDecodeError, UnicodeDecodeError):
                    parsed = None
                _emit(emit, "codex-app-server", "stdout", raw, parsed, line)
                messages.put(("message", parsed if isinstance(parsed, dict) else {"_invalid": raw}))
        finally:
            proc.stdout.close()
            messages.put(("eof", None))

    def read_stderr() -> None:
        assert proc.stderr is not None
        try:
            for line in iter(proc.stderr.readline, b""):
                _emit(emit, "codex-app-server", "stderr",
                      line.decode("utf-8", errors="backslashreplace"), raw_bytes=line)
        finally:
            proc.stderr.close()

    stdout_thread = threading.Thread(target=read_stdout, daemon=True)
    stderr_thread = threading.Thread(target=read_stderr, daemon=True)
    stdout_thread.start()
    stderr_thread.start()
    deadline = time.monotonic() + timeout
    proc._wrapper_deadline = deadline
    proc._wrapper_request_timeout = request_timeout
    proc._wrapper_cancel = cancel
    next_id = 1
    pending: dict[int, tuple[str, float]] = {}
    early: deque[dict[str, Any]] = deque()
    session = resume_id
    turn: str | None = None
    usage = None
    phase = "initialize"
    phase_id: int | None = None
    terminal: dict[str, Any] | None = None
    incoming_attention = False
    try:
        _rpc(proc, next_id, "initialize", {"clientInfo": {"name": "markitect-conventional-playground", "version": "1"}}, emit)
        phase_id = next_id
        pending[next_id] = (phase, time.monotonic() + request_timeout)
        next_id += 1
        while True:
            # If cancellation arrives while turn/start is in flight, first
            # collect its reply so any interrupt names the exact owned turn.
            if cancel.is_set() and phase != "done" and not (phase == "turn-start" and turn is None):
                if turn:
                    try:
                        _rpc(proc, next_id, "turn/interrupt", {"threadId": session, "turnId": turn}, emit)
                    except OSError:
                        pass
                    interrupt_deadline = time.monotonic() + min(15.0, request_timeout)
                    while time.monotonic() < interrupt_deadline:
                        item = _next_message(messages, proc, 0.05)
                        if item is None:
                            continue
                        kind, message = item
                        if kind == "eof":
                            break
                        if _matches_terminal(message, session, turn):
                            _terminate(proc)
                            state, status = _state_from_terminal(message)
                            return _result("cancelled" if state == "cancelled" else "uncertain", session, turn,
                                           _usage(message) or usage,
                                           f"cancel requested; matching terminal status: {status}")
                    _terminate(proc)
                    return _result("uncertain", session, turn, usage,
                                   "cancel requested; matching interruption terminal was not observed")
                _terminate(proc)
                return _result("cancelled", session, turn, usage, "cancelled before a native turn was started")
            if time.monotonic() >= deadline:
                _terminate(proc)
                return _result("uncertain", session, turn, usage, "App Server deadline expired; outcome is uncertain")
            if phase_id is not None and phase_id in pending and time.monotonic() >= pending[phase_id][1]:
                _terminate(proc)
                return _result("uncertain", session, turn, usage, f"App Server request timed out: {phase}")
            item = _next_message(messages, proc, 0.05)
            if item is None:
                continue
            kind, message = item
            if kind == "eof":
                _terminate(proc)
                return _result("uncertain", session, turn, usage, "App Server closed its output before a terminal event")
            if not isinstance(message, dict):
                continue
            method = message.get("method")
            if method and message.get("id") is not None:
                if _requires_user_input(method, message):
                    incoming_attention = True
                    _terminate(proc)
                    return _result("needs_input", session, turn, usage,
                                   f"App Server requested user approval/input via {method}; no approval was granted")
                # Unknown server-originated calls are not approved. A protocol
                # error response makes the unsupported request explicit.
                response = {"id": message["id"], "error": {"code": -32601, "message": "Unsupported client request; no action was authorized"}}
                raw = json.dumps(response, separators=(",", ":"))
                assert proc.stdin is not None
                wire = (raw + "\n").encode("utf-8")
                _deliver_rpc(proc, wire)
                _emit(emit, "codex-app-server", "client", raw, response, wire)
                continue
            if "id" in message and message.get("id") in pending:
                reply_id = message["id"]
                completed_phase, _ = pending.pop(reply_id)
                phase_id = None
                if message.get("error") is not None:
                    _terminate(proc)
                    return _result("failed", session, turn, usage,
                                   f"App Server rejected {completed_phase}: {message['error']}")
                result = message.get("result", {})
                if completed_phase == "initialize":
                    _send_notification(proc, "initialized", {}, emit)
                    method_name = "thread/resume" if resume_id else "thread/start"
                    params = dict(thread_options)
                    params.update({"cwd": cwd, "model": model})
                    if runtime_options is not None:
                        params["sandbox"] = runtime_options["sandbox"]
                        params["approvalPolicy"] = runtime_options["approvalPolicy"]
                    if resume_id:
                        params["threadId"] = resume_id
                    request_id = next_id
                    next_id += 1
                    _rpc(proc, request_id, method_name, params, emit)
                    pending[request_id] = ("thread", time.monotonic() + request_timeout)
                    phase, phase_id = "thread", request_id
                elif completed_phase == "thread":
                    returned_session = _session_id(result)
                    if resume_id and returned_session and returned_session != resume_id:
                        _terminate(proc)
                        return _result("uncertain", resume_id, None, usage,
                                       "App Server resume returned a different native thread id; no turn was started")
                    session = returned_session or session
                    if not session:
                        _terminate(proc)
                        return _result("failed", None, None, usage, "App Server thread response omitted its native thread id")
                    _runtime_binding_event(emit, session, result)
                    params = {"cwd": cwd, "model": model}
                    request_id = next_id
                    next_id += 1
                    params.update({"threadId": session, "input": [{"type": "text", "text": prompt}],
                                   "model": model, "effort": effort})
                    _rpc(proc, request_id, "turn/start", params, emit)
                    pending[request_id] = ("turn-start", time.monotonic() + request_timeout)
                    phase, phase_id = "turn-start", request_id
                elif completed_phase == "turn-start":
                    turn = _turn_id(result)
                    if not turn:
                        _terminate(proc)
                        return _result("uncertain", session, None, usage, "turn/start reply omitted native turn id")
                    phase, phase_id = "terminal", None
                    # Notifications can race ahead of the start reply. Retain
                    # them and only accept an exact matching thread and turn.
                    for queued in early:
                        if _matches_terminal(queued, session, turn):
                            terminal = queued
                            break
                    if terminal is not None:
                        usage = _usage(terminal) or usage
                        state, status = _state_from_terminal(terminal)
                        _terminate(proc)
                        return _result(state, session, turn, usage,
                                       f"App Server terminal status: {status}")
                continue
            if method:
                params = message.get("params", {})
                if session and _session_id(params) == session:
                    reported_usage = _usage(message)
                    if reported_usage is not None:
                        usage = reported_usage
                if phase == "terminal" and _matches_terminal(message, session, turn):
                    terminal = message
                    usage = _usage(message) or usage
                    state, status = _state_from_terminal(message)
                    _terminate(proc)
                    return _result(state, session, turn, usage, f"App Server terminal status: {status}")
                if method in {"turn/completed", "turn/failed", "turn/interrupted", "turn/cancelled"}:
                    early.append(message)
                continue
        # unreachable
    except (BrokenPipeError, OSError) as exc:
        _terminate(proc)
        return _result("uncertain", session, turn, usage, f"App Server transport failed: {exc}")
    finally:
        # Do not leave a child process owned by this invocation running.
        if proc.poll() is None:
            _stop_process(proc)
        stdout_thread.join(timeout=0.25)
        stderr_thread.join(timeout=0.25)
        if not stdout_thread.is_alive() and not stderr_thread.is_alive():
            _close_pipes(proc)


def _next_message(messages: queue.Queue[tuple[str, Any]], proc: subprocess.Popen[bytes], wait: float) -> tuple[str, Any] | None:
    try:
        return messages.get(timeout=wait)
    except queue.Empty:
        return None


def _send_notification(proc: subprocess.Popen[bytes], method: str, params: dict[str, Any], emit: Emit) -> None:
    message = {"method": method, "params": params}
    raw = json.dumps(message, separators=(",", ":"))
    assert proc.stdin is not None
    wire = (raw + "\n").encode("utf-8")
    _deliver_rpc(proc, wire)
    _emit(emit, "codex-app-server", "client", raw, message, wire)


def _requires_user_input(method: str, message: dict[str, Any]) -> bool:
    name = method.lower()
    return any(word in name for word in ("approval", "permission", "input", "elicitation"))


def _matches_terminal(message: dict[str, Any], session: str | None, turn: str | None) -> bool:
    method = message.get("method")
    if method not in {"turn/completed", "turn/failed", "turn/interrupted", "turn/cancelled"}:
        return False
    params = message.get("params", {})
    return bool(session and turn and _session_id(params) == session and _turn_id(params) == turn)
