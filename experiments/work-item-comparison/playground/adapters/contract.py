"""Small, versioned contract between the Playground and method adapters.

Adapters own translation to a method's native setup and runtime. The Playground
owns cases, prompts, budgets, snapshots, and assessment. Returned lifecycle
states describe dispatch disposition; they do not assess the engineering task.
"""
from __future__ import annotations

from datetime import datetime
from copy import deepcopy
from typing import Protocol, runtime_checkable


SCHEMA = 1
CAPABILITIES = ("ensure_runtime", "start", "resume", "status", "cancel", "close")
RUNTIME_OWNERS = {"adapter", "external", "none"}
LIFECYCLE_STATES = {
    "ready", "blocked", "accepted", "starting", "running", "completed",
    "failed", "cancelled", "uncertain", "needs_input", "closed",
}
OBSERVATION_TYPES = {
    "runtime.ready", "runtime.blocked", "attempt.accepted", "attempt.started",
    "attempt.finished", "process.started", "process.stopped", "session.bound",
    "helper.started", "usage.reported", "adapter.warning",
}


class AdapterError(ValueError):
    """A manifest, adapter descriptor, or lifecycle result violates schema 1."""


@runtime_checkable
class Adapter(Protocol):
    def describe(self) -> dict: ...
    def setup(self, context: dict) -> dict: ...
    def ensure_runtime(self) -> dict: ...
    def start(self, prompt: str) -> dict: ...
    def resume(self, run_id: str, prompt: str) -> dict: ...
    def status(self, run_id: str) -> dict: ...
    def cancel(self, run_id: str) -> dict: ...
    def close(self) -> dict: ...


def _is_text(value):
    return isinstance(value, str) and bool(value.strip())


def validate_descriptor(value):
    """Validate and return the adapter's stable, non-evaluative identity."""
    if not isinstance(value, dict) or value.get("schema") != SCHEMA:
        raise AdapterError("adapter describe() must return schema 1 object")
    for key in ("id", "method", "version"):
        if not _is_text(value.get(key)):
            raise AdapterError(f"adapter descriptor requires nonempty {key}")
    caps = value.get("capabilities")
    if not isinstance(caps, dict) or set(caps) != set(CAPABILITIES):
        raise AdapterError("descriptor capabilities must declare every schema 1 operation")
    if any(type(caps[key]) is not bool for key in CAPABILITIES):
        raise AdapterError("descriptor capability values must be booleans")
    runtime = value.get("runtime")
    if (not isinstance(runtime, dict) or not _is_text(runtime.get("kind")) or
            runtime.get("owner") not in RUNTIME_OWNERS or not _is_text(runtime.get("scope"))):
        raise AdapterError("descriptor runtime requires kind, owner, and scope")
    pins = value.get("sourcePins")
    if not isinstance(pins, dict) or not pins:
        raise AdapterError("descriptor sourcePins must be a nonempty mapping")
    if any(not _is_text(path) or not _is_text(digest) for path, digest in pins.items()):
        raise AdapterError("descriptor sourcePins entries must be nonempty strings")
    return value


def validate_result(action, value):
    """Validate an operation envelope while retaining adapter-specific receipts."""
    if not isinstance(value, dict) or value.get("schema") != SCHEMA:
        raise AdapterError(f"{action}() must return a schema 1 result object")
    state = value.get("state")
    if state not in LIFECYCLE_STATES:
        raise AdapterError(f"{action}() returned an unknown lifecycle state")
    if action in {"start", "resume"} and state not in {
            "accepted", "blocked", "starting", "running", "completed", "failed",
            "cancelled", "uncertain", "needs_input"}:
        raise AdapterError(f"{action}() returned an invalid dispatch state")
    if action == "setup" and state not in {"ready", "blocked", "failed", "uncertain"}:
        raise AdapterError("setup() must report ready, blocked, failed, or uncertain")
    if action == "ensure_runtime" and state not in {"ready", "blocked", "failed", "uncertain"}:
        raise AdapterError("ensure_runtime() must report ready, blocked, failed, or uncertain")
    if action == "close" and state not in {"closed", "uncertain", "failed"}:
        raise AdapterError("close() must report closed, uncertain, or failed")
    if action in {"start", "resume", "status", "cancel"} and state != "blocked" and state in {
            "accepted", "starting", "running", "completed", "cancelled", "failed",
            "uncertain", "needs_input"}:
        if not _is_text(value.get("runId")):
            raise AdapterError(f"{action}() lifecycle result requires a runId")
    observations = value.get("observations", [])
    if not isinstance(observations, list):
        raise AdapterError("observations must be a list")
    for event in observations:
        timestamp = event.get("observedAt") if isinstance(event, dict) else None
        try:
            parsed_timestamp = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
            timestamp_valid = parsed_timestamp.tzinfo is not None
        except (AttributeError, TypeError, ValueError):
            timestamp_valid = False
        if (not isinstance(event, dict) or event.get("schema") != SCHEMA or
                event.get("type") not in OBSERVATION_TYPES or
                not timestamp_valid or not isinstance(event.get("data"), dict)):
            raise AdapterError("observation must have schema, known type, observedAt, and object data")
    return value


def deliver_observations(result, observer):
    """Call a passive event sink; preserve the result if observation delivery fails."""
    delivered = deepcopy(result)
    failures = list(delivered.get("observerFailures", []))
    for event in delivered.get("observations", []):
        try:
            observer(deepcopy(event))
        except Exception as exc:  # observational hooks must not control the study
            failures.append({"eventType": event.get("type"), "error": type(exc).__name__})
    if failures:
        delivered["observerFailures"] = failures
    return delivered
