"""One durable control service shared by the CLI and MCP facades.

The checked-in configuration is disabled. Enabling execution requires an external,
finite order bound to the exact configuration and prepared Conventional checkout.
This is an admission guard, not proof of human authority or native helper limits.
"""
from __future__ import annotations

from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import threading
import time
import uuid
from .backends import validate_runtime_options


ACTIVE = {"starting", "running"}
RESUMABLE = {"completed", "failed", "cancelled"}
RESULT_STATES = RESUMABLE | {"uncertain", "needs_input"}
CONFIG_KEYS = {"schema", "execution_authorized", "actualOrderPath", "backend",
               "command", "filePins", "cwd", "audit", "model", "effort",
               "timeoutSeconds", "requestTimeoutSeconds", "threadOptions",
               "turnOptions", "runtimeBinding", "runtimeOptions"}


def _now():
    return datetime.now(timezone.utc)


def _time():
    return _now().isoformat()


def _digest(data):
    return hashlib.sha256(data).hexdigest()


def _bytes(value):
    return (json.dumps(value, ensure_ascii=False, sort_keys=True, indent=2) + "\n").encode("utf-8")


def _read(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def _new(path, data):
    with path.open("xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def _save(path, value):
    temp = path.with_name(path.name + "." + uuid.uuid4().hex + ".tmp")
    _new(temp, _bytes(value))
    # Windows readers can briefly prevent replacing an open status file. Retry
    # only this local write; a native request is never retried here.
    deadline = time.monotonic() + 2
    while True:
        try:
            os.replace(temp, path)
            return
        except PermissionError:
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.01)


def _absolute(value, name):
    if not isinstance(value, str) or not Path(value).is_absolute():
        raise ValueError(name + " must be an absolute path")
    path = Path(value)
    if path.is_symlink():
        raise ValueError(name + " cannot be a symlink")
    return path.resolve()


def _positive(value, name):
    if isinstance(value, bool) or not isinstance(value, (float, int)) or not math.isfinite(value) or value <= 0:
        raise ValueError(name + " must be finite and positive")
    return value


class Service:
    def __init__(self, config_path, *, backend_runner=None):
        self.config_path = Path(config_path).resolve()
        self.config_bytes = self.config_path.read_bytes()
        self.config = json.loads(self.config_bytes)
        if not isinstance(self.config, dict) or set(self.config) - CONFIG_KEYS or self.config.get("schema") != 1:
            raise ValueError("invalid or unknown configuration fields")
        if type(self.config.get("execution_authorized")) is not bool:
            raise ValueError("execution_authorized must be explicit true or false")
        self.config_sha = _digest(self.config_bytes)
        self._runner = backend_runner
        self._mutex = threading.RLock()
        self._workers = {}
        self._closed = False

    def _paths(self):
        repo = _absolute(self.config.get("cwd"), "cwd")
        audit = _absolute(self.config.get("audit"), "audit")
        if repo == audit or repo in audit.parents or audit in repo.parents:
            raise ValueError("repository and audit must be disjoint")
        store = audit / "conventional-execution"
        if store.is_symlink():
            raise ValueError("execution store cannot be a symlink")
        return repo, audit, store

    def _admit(self):
        if self._closed:
            raise ValueError("controller is closed")
        if self.config["execution_authorized"] is not True:
            raise ValueError("execution disabled; preparation grants do not authorize native starts")
        if self.config_path.read_bytes() != self.config_bytes:
            raise ValueError("configuration changed; instantiate a new controller")
        repo, audit, store = self._paths()
        run = _read(audit / "run.json")
        if (run.get("method") != "Conventional" or
                _absolute(run.get("repoPath"), "run repoPath") != repo or
                _absolute(run.get("auditPath"), "run auditPath") != audit or
                (repo / ".study/run-id").read_text(encoding="ascii").strip() != run.get("runId")):
            raise ValueError("prepared Conventional run binding does not match")
        if (audit / "final-freeze").exists():
            raise ValueError("prepared run is frozen")
        command = self.config.get("command")
        if (not isinstance(command, list) or not command or
                not all(isinstance(arg, str) and arg and "\x00" not in arg for arg in command)):
            raise ValueError("command must be a nonempty argument array")
        executable = _absolute(command[0], "command executable")
        pins = self.config.get("filePins")
        if not isinstance(pins, dict):
            raise ValueError("filePins must include exact executable bytes")
        normalized_pins = {}
        for name, digest in pins.items():
            path = _absolute(name, "file pin")
            if not isinstance(digest, str) or _digest(path.read_bytes()) != digest:
                raise ValueError("file pin changed: " + name)
            normalized_pins[str(path)] = digest
        if str(executable) not in normalized_pins:
            raise ValueError("filePins must include exact executable bytes")
        source_root = Path(__file__).resolve().parent
        required_sources = [source_root / name for name in ("service.py", "backends.py", "mcp.py")]
        required_sources.append(source_root.parent / "conventional_wrapper.py")
        if any(str(path) not in normalized_pins for path in required_sources):
            raise ValueError("filePins must include wrapper service, backend, MCP and CLI source")
        for key in ("threadOptions", "turnOptions"):
            if self.config.get(key, {}) != {}:
                raise ValueError(key + " must be empty; this wrapper inherits native defaults")
        if self.config.get("backend") not in {"codex-cli", "codex-app-server"}:
            raise ValueError("unsupported native backend")
        for key in ("model", "effort"):
            if not isinstance(self.config.get(key), str) or not self.config[key].strip():
                raise ValueError(key + " must be explicit")
        runtime_options = validate_runtime_options(self.config.get("runtimeOptions"),
                                                   self.config["model"], self.config["effort"])
        _positive(self.config.get("timeoutSeconds"), "timeoutSeconds")
        _positive(self.config.get("requestTimeoutSeconds"), "requestTimeoutSeconds")
        order_path = _absolute(self.config.get("actualOrderPath"), "actualOrderPath")
        order_bytes = order_path.read_bytes()
        order = json.loads(order_bytes)
        if (not isinstance(order, dict) or order.get("execution_authorized") is not True or
                order.get("configSha256") != self.config_sha or order.get("runId") != run["runId"] or
                not isinstance(order.get("actualOrderRef"), str) or not order["actualOrderRef"].strip()):
            raise ValueError("external order must authorize and bind configuration/run exactly")
        limit = order.get("maxTurns")
        if type(limit) is not int or limit <= 0:
            raise ValueError("maxTurns must be a positive integer; native helpers are separate")
        try:
            end = datetime.fromisoformat(order["expiresAt"].replace("Z", "+00:00"))
            if end.tzinfo is None:
                raise ValueError()
            remaining = (end - _now()).total_seconds()
        except (KeyError, TypeError, AttributeError, ValueError) as exc:
            raise ValueError("order requires an absolute timezone-aware expiresAt") from exc
        if remaining <= 0:
            raise ValueError("external order expired")
        store.mkdir(exist_ok=True)
        binding = {"configSha256": self.config_sha, "orderSha256": _digest(order_bytes), "runId": run["runId"]}
        binding_path = store / "binding.json"
        if binding_path.exists():
            if _read(binding_path) != binding:
                raise ValueError("execution store belongs to another config/order/run")
        else:
            _new(binding_path, _bytes(binding))
        for filename, data in (("config.json", self.config_bytes), ("order.json", order_bytes)):
            path = store / filename
            if not path.exists():
                _new(path, data)
            elif path.read_bytes() != data:
                raise ValueError("immutable execution binding changed: " + filename)
        spec = dict(self.config)
        spec["cwd"] = str(repo)
        spec["runtimeOptions"] = runtime_options
        spec["timeoutSeconds"] = min(spec["timeoutSeconds"], remaining)
        spec["requestTimeoutSeconds"] = min(spec["requestTimeoutSeconds"], remaining)
        return store, spec, order, binding

    def _record(self, run_id):
        try:
            canonical = str(uuid.UUID(run_id))
        except (ValueError, TypeError, AttributeError) as exc:
            raise ValueError("run_id must be a wrapper UUID") from exc
        if canonical != run_id:
            raise ValueError("run_id must be a canonical wrapper UUID")
        _, _, store = self._paths()
        folder = store / run_id
        if folder.is_symlink():
            raise ValueError("run folder cannot be a symlink")
        try:
            result = _read(folder / "status.json")
        except FileNotFoundError as exc:
            raise ValueError("unknown run_id") from exc
        if result.get("configSha256") != self.config_sha:
            raise ValueError("run belongs to another configuration")
        return folder, result

    def start(self, prompt):
        return self._submit(prompt, None)

    def resume(self, run_id, prompt):
        return self._submit(prompt, run_id)

    def _submit(self, prompt, parent_id):
        if not isinstance(prompt, str) or not prompt.strip():
            raise ValueError("prompt must be nonempty ordinary user text")
        with self._mutex:
            store, spec, order, binding = self._admit()
            parent = None
            if parent_id:
                parent_folder, parent = self._record(parent_id)
                if parent["state"] not in RESUMABLE or not parent.get("nativeSessionId"):
                    raise ValueError("resume requires a known terminal owned session; uncertain dispatch cannot be replayed")
                if (parent_folder / "resumed-by.json").exists():
                    raise ValueError("session already continued; resume its latest turn instead")
            run_id = str(uuid.uuid4())
            lock = store / "active.json"
            try:
                _new(lock, _bytes({"runId": run_id, "ownerPid": os.getpid(), "createdAt": _time()}))
            except FileExistsError as exc:
                raise ValueError("active or unresolved owner exists; inspect its evidence, do not replay") from exc
            admitted = False
            try:
                # Directories reserve attempts before dispatch, including failed starts.
                attempts = [p for p in store.iterdir() if p.is_dir()]
                if len(attempts) >= order["maxTurns"]:
                    raise ValueError("finite outer-turn allowance exhausted")
                folder = store / run_id
                folder.mkdir()
                request = {**binding, "runId": run_id, "parentRunId": parent_id,
                           "actualOrderRef": order["actualOrderRef"], "prompt": prompt,
                           "backend": spec["backend"], "requestedModel": spec["model"],
                           "requestedEffort": spec["effort"], "cwd": spec["cwd"],
                           "timeoutSeconds": spec["timeoutSeconds"], "createdAt": _time(),
                           "runtimeBinding": spec.get("runtimeBinding"),
                           "runtimeOptions": spec.get("runtimeOptions"),
                           "command": spec["command"], "filePins": spec["filePins"],
                           "preparedRunSha256": _digest((store.parent / "run.json").read_bytes()),
                           "stationControlSha256": _digest((Path(spec["cwd"]) / ".study/station.json").read_bytes())}
                _new(folder / "request.json", _bytes(request))
                if parent:
                    _new(parent_folder / "resumed-by.json", _bytes({"runId": run_id}))
                status = {**request, "state": "starting", "nativeSessionId": None,
                          "nativeTurnId": None, "usage": None, "taskAssessment": "NOT RUN",
                          "humanAcceptance": "not established", "updatedAt": _time()}
                _save(folder / "status.json", status)
                cancel = threading.Event()
                worker = threading.Thread(target=self._execute,
                                          args=(folder, spec, prompt, parent, status, cancel),
                                          name="conventional-" + run_id, daemon=True)
                self._workers[run_id] = (worker, cancel)
                try:
                    worker.start()
                except RuntimeError as exc:
                    # A Python worker was never started, so no backend was called.
                    status.update(state="failed", detail="worker not dispatched: " + str(exc), updatedAt=_time())
                    _new(folder / "result.json", _bytes({"state": "failed", "dispatchOccurred": False,
                                                        "detail": status["detail"]}))
                    _save(folder / "status.json", status)
                    del self._workers[run_id]
                    if parent:
                        # Retain the attempted claim in this failed attempt;
                        # the native parent was never continued.
                        claim = parent_folder / "resumed-by.json"
                        _new(folder / "resume-not-dispatched.json", claim.read_bytes())
                        claim.unlink()
                    return dict(status)
                admitted = True
                return dict(status)
            finally:
                if not admitted:
                    # Admission failed before a native dispatch; never remove run evidence.
                    lock.unlink()

    def _execute(self, folder, spec, prompt, parent, status, cancel):
        log = folder / "events.jsonl"
        sequence = 0
        event_mutex = threading.Lock()
        finished = threading.Event()

        def watch_cancel():
            while not finished.wait(0.1):
                if (folder / "cancel-request.json").exists():
                    cancel.set()

        watcher = threading.Thread(target=watch_cancel, daemon=True)
        watcher.start()

        def emit(event):
            nonlocal sequence
            # stdout, stderr and client wire readers may emit concurrently.
            with event_mutex:
                sequence += 1
                envelope = {"sequence": sequence, "observedAt": _time(), "event": event}
                with log.open("ab") as stream:
                    stream.write((json.dumps(envelope, ensure_ascii=False) + "\n").encode("utf-8"))
                    stream.flush()
                    os.fsync(stream.fileno())

        try:
            status["state"] = "running"
            status["updatedAt"] = _time()
            _save(folder / "status.json", status)
            if self._runner is None:
                from .backends import run
                runner = run
            else:
                runner = self._runner
            result = runner(spec, prompt, parent.get("nativeSessionId") if parent else None, emit, cancel)
            if not isinstance(result, dict) or result.get("state") not in RESULT_STATES:
                raise RuntimeError("backend returned no valid lifecycle result")
            result_record = {**result, "runtimeOptions": spec.get("runtimeOptions")}
            _new(folder / "result.json", _bytes(result_record))
            status.update({key: result.get(key) for key in ("state", "nativeSessionId", "nativeTurnId", "usage", "usageScope", "detail", "ownedProcessScope")})
        except Exception as exc:
            # After durable intent, an exception cannot establish absence of execution.
            status.update(state="uncertain", detail=type(exc).__name__ + ": " + str(exc))
            emit({"kind": "controller-error", "detail": status["detail"]})
        finally:
            finished.set()
            watcher.join()
            status["updatedAt"] = _time()
            try:
                _save(folder / "status.json", status)
            finally:
                if status["state"] in RESUMABLE:
                    lock = folder.parent / "active.json"
                    if _read(lock).get("runId") == folder.name:
                        lock.unlink()
                # Unknown ownership/request disposition retains the admission lock.

    def status(self, run_id):
        with self._mutex:
            folder, result = self._record(run_id)
            result["cancelRequested"] = (folder / "cancel-request.json").exists()
            result["controllerOwnership"] = "this_process" if run_id in self._workers else "unverified_in_this_process"
            if result["state"] in ACTIVE and run_id not in self._workers:
                result["ownershipNote"] = "May be active elsewhere or interrupted by controller loss; never replay automatically."
            return result

    def cancel(self, run_id):
        with self._mutex:
            folder, result = self._record(run_id)
            if result["state"] in ACTIVE:
                path = folder / "cancel-request.json"
                if not path.exists():
                    try:
                        _new(path, _bytes({"requestedAt": _time()}))
                    except FileExistsError:
                        pass
                if run_id in self._workers:
                    self._workers[run_id][1].set()
            return self.status(run_id)

    def wait(self, run_id):
        """Blocking CLI lifecycle; no repeated model requests or semantic coaching."""
        if run_id not in self._workers:
            return self.status(run_id)
        worker, cancel = self._workers[run_id]
        folder, _ = self._record(run_id)
        while worker.is_alive():
            if (folder / "cancel-request.json").exists():
                cancel.set()
            worker.join(0.1)
        return self.status(run_id)

    def close(self):
        with self._mutex:
            self._closed = True
            workers = list(self._workers.values())
            for worker, cancel in workers:
                if worker.is_alive():
                    cancel.set()
        # Backend interruption/cleanup is bounded; leave unresolved lock intact.
        deadline = time.monotonic() + 60
        for worker, _ in workers:
            worker.join(max(0, deadline - time.monotonic()))
