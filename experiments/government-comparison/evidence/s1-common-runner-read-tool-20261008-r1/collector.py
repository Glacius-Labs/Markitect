"""Fail-closed collector for the one frozen common-runner read-tool packet.

The transport owns framing and JSON-RPC envelopes. This module receives only
decoded result/notification members after that layer has selected them. It
keeps protocol identifiers and the single explicitly safe public probe value;
message, reasoning, command output, and configuration text are never logged.
"""

from __future__ import annotations

import json
import re
from collections.abc import Callable, Mapping
from typing import Any


_SCHEMAS = {
    "thread/start": "v2/ThreadStartResponse.json",
    "turn/start": "v2/TurnStartResponse.json",
    "thread/started": "v2/ThreadStartedNotification.json",
    "thread/status/changed": "v2/ThreadStatusChangedNotification.json",
    "thread/closed": "v2/ThreadClosedNotification.json",
    "thread/tokenUsage/updated": "v2/ThreadTokenUsageUpdatedNotification.json",
    "turn/started": "v2/TurnStartedNotification.json",
    "turn/completed": "v2/TurnCompletedNotification.json",
    "item/started": "v2/ItemStartedNotification.json",
    "item/completed": "v2/ItemCompletedNotification.json",
    "item/agentMessage/delta": "v2/AgentMessageDeltaNotification.json",
    "item/commandExecution/outputDelta": "v2/CommandExecutionOutputDeltaNotification.json",
    "item/reasoning/summaryTextDelta": "v2/ReasoningSummaryTextDeltaNotification.json",
    "item/reasoning/summaryPartAdded": "v2/ReasoningSummaryPartAddedNotification.json",
    "item/reasoning/textDelta": "v2/ReasoningTextDeltaNotification.json",
}

_ALLOWED_ITEM_TYPES = frozenset({"userMessage", "reasoning", "commandExecution", "agentMessage"})
_LOWER_HEX_128 = re.compile(r"[0-9a-f]{32}\Z")
_SHA256 = re.compile(r"[0-9a-f]{64}\Z")


class Collector:
    """Collect one correlated thread/turn and verify its finite success proof.

    ``validate(member_name, value)`` must validate ``value`` against the exact
    frozen archive member and raise (or return ``False``) on failure. Profile
    and expected data are frozen by the caller; this object never derives a
    command, cwd, prompt, model, or sentinel.
    """

    def __init__(self, profile: Mapping[str, Any], expected: Mapping[str, Any],
                 validate: Callable[[str, Any], Any]):
        self._profile = profile
        self._expected = expected
        self._validate_callback = validate
        self._failed = False
        self._stop_reason: str | None = None
        self._thread_response_seen = False
        self._turn_response_seen = False
        self._thread_started_seen = False
        self._turn_started_seen = False
        self._turn_completed_seen = False
        self._thread_closed_seen = False
        self._postterminal_idle_seen = False
        self._notification_count = 0
        self._pending: list[dict[str, Any]] = []
        self._items: dict[str, dict[str, Any]] = {}
        self._thread_id: str | None = None
        self._turn_id: str | None = None
        self._thread_status: str | None = None
        self._latest_usage: dict[str, int] | None = None
        self._usage_seen = False
        self._usage_overshoot: int | None = None
        self._command_item_id: str | None = None
        self._observed_command_ids: set[str] = set()
        self._command_completed_event_seen = False
        self._answer_item_id: str | None = None
        self._final_candidate_item_id: str | None = None
        self._user_item_id: str | None = None
        self._safe_output: str | None = None
        self._safe_final: str | None = None
        self._max_pending = self._positive_limit("maxPendingLifecycleEvents", 64)
        self._max_items = self._positive_limit("maxTrackedItems", 16)
        self._max_notifications = self._positive_limit_from_limits("maxInboundNotifications", 2048)
        self._max_safe_output = self._positive_limit("safePublicOutputMaxBytes", 512)
        self._token_threshold = self._positive_limit_from_limits("tokenStopThreshold", 50000)
        self._cwd = self._text(profile, "cwd")
        self._command = self._text(profile, "command")
        self._turn_prompt = self._text(profile, "turnPrompt")
        sentinel = expected.get("sentinel")
        file_hash = expected.get("fileSha256")
        if not isinstance(sentinel, str) or not _LOWER_HEX_128.fullmatch(sentinel):
            self._stop("invalid-frozen-sentinel")
        if not isinstance(file_hash, str) or not _SHA256.fullmatch(file_hash):
            self._stop("invalid-frozen-file-hash")
        self._sentinel = sentinel if isinstance(sentinel, str) else ""
        self._file_hash = file_hash if isinstance(file_hash, str) else ""
        self._thread_start = profile.get("threadStart")
        self._turn_start = profile.get("turnStart")
        if not self._is_mapping(self._thread_start) or not self._is_mapping(self._turn_start):
            self._stop("invalid-frozen-start-profile")
        if self._command == "" or self._cwd == "" or self._turn_prompt == "":
            self._stop("invalid-frozen-profile")

    @staticmethod
    def _is_mapping(value: Any) -> bool:
        return isinstance(value, Mapping)

    @staticmethod
    def _text(value: Mapping[str, Any], key: str) -> str:
        result = value.get(key)
        return result if isinstance(result, str) else ""

    def _positive_limit(self, key: str, default: int) -> int:
        value = self._profile.get(key, default)
        return value if type(value) is int and 0 < value <= 1000000 else default

    def _positive_limit_from_limits(self, key: str, default: int) -> int:
        limits = self._profile.get("limits")
        value = limits.get(key) if self._is_mapping(limits) else None
        return value if type(value) is int and 0 < value <= 100000000 else default

    @staticmethod
    def _int64(value: Any, optional: bool = False) -> bool:
        if optional and value is None:
            return True
        return type(value) is int and -(1 << 63) <= value < (1 << 63)

    def _stop(self, reason: str) -> None:
        if not self._failed:
            self._failed = True
            self._stop_reason = reason

    def _validate(self, method: str, value: Any) -> bool:
        member = _SCHEMAS.get(method)
        if member is None:
            self._stop("unknown-method")
            return False
        try:
            result = self._validate_callback(member, value)
        except Exception:
            self._stop("schema-rejected")
            return False
        if result is False:
            self._stop("schema-rejected")
            return False
        return True

    def accept_response(self, method: str, result: Any) -> None:
        """Accept only the two frozen startup responses and reconcile IDs."""
        if self._failed:
            return
        if method not in ("thread/start", "turn/start"):
            self._stop("unexpected-response-method")
            return
        if not self._validate(method, result) or not self._is_mapping(result):
            if not self._failed:
                self._stop("malformed-response")
            return
        if method == "thread/start":
            if self._thread_response_seen or self._turn_response_seen:
                self._stop("duplicate-or-late-thread-response")
                return
            if not self._thread_response_gate(result):
                return
            self._thread_response_seen = True
            self._reconcile_pending()
            return
        if not self._thread_response_seen or self._turn_response_seen:
            self._stop("turn-response-before-thread-gate-or-duplicate")
            return
        if not self._turn_response_gate(result):
            return
        self._turn_response_seen = True
        self._reconcile_pending()

    def _thread_response_gate(self, result: Mapping[str, Any]) -> bool:
        request = self._thread_start
        if not self._is_mapping(request):
            self._stop("invalid-frozen-thread-start")
            return False
        params = request.get("params")
        if not self._is_mapping(params):
            self._stop("invalid-frozen-thread-start")
            return False
        thread = result.get("thread")
        if not self._is_mapping(thread):
            self._stop("thread-response-missing-thread")
            return False
        expected_model = params.get("model")
        expected_provider = params.get("modelProvider")
        expected_approval = params.get("approvalPolicy")
        if (result.get("cwd") != self._cwd or result.get("model") != expected_model
                or result.get("modelProvider") != expected_provider
                or result.get("approvalPolicy") != expected_approval):
            self._stop("thread-response-identity-or-policy-mismatch")
            return False
        if (thread.get("cwd") != self._cwd or thread.get("modelProvider") != expected_provider
                or thread.get("ephemeral") is not True or thread.get("parentThreadId") is not None
                or thread.get("forkedFromId") is not None):
            self._stop("thread-response-thread-mismatch")
            return False
        model_on_thread = thread.get("model")
        if model_on_thread not in (None, expected_model):
            self._stop("thread-response-model-mismatch")
            return False
        profile = result.get("activePermissionProfile")
        sandbox = result.get("sandbox")
        if profile is not None:
            if (not self._is_mapping(profile) or profile.get("id") != ":read-only"
                    or profile.get("extends") not in (None, "")):
                self._stop("thread-response-permission-profile-mismatch")
                return False
        elif not self._readonly_sandbox(sandbox):
            self._stop("thread-response-permission-provenance-unknown")
            return False
        if not self._readonly_sandbox(sandbox):
            self._stop("thread-response-effective-sandbox-mismatch")
            return False
        identifier = thread.get("id")
        if not isinstance(identifier, str) or not identifier or len(identifier) > 128:
            self._stop("thread-response-invalid-id")
            return False
        self._thread_id = identifier
        return True

    @staticmethod
    def _readonly_sandbox(sandbox: Any) -> bool:
        if not isinstance(sandbox, Mapping) or sandbox.get("type") != "readOnly":
            return False
        if sandbox.get("networkAccess") is True:
            return False
        return True

    def _turn_response_gate(self, result: Mapping[str, Any]) -> bool:
        request = self._turn_start
        if not self._is_mapping(request):
            self._stop("invalid-frozen-turn-start")
            return False
        params = request.get("params")
        if not self._is_mapping(params):
            self._stop("invalid-frozen-turn-start")
            return False
        turn = result.get("turn")
        if not self._is_mapping(turn):
            self._stop("turn-response-missing-turn")
            return False
        identifier = turn.get("id")
        status = turn.get("status")
        if not isinstance(identifier, str) or not identifier or len(identifier) > 128:
            self._stop("turn-response-invalid-id")
            return False
        if status not in ("inProgress", "completed"):
            self._stop("turn-response-invalid-status")
            return False
        if turn.get("error") is not None:
            self._stop("turn-response-has-error")
            return False
        if self._turn_id is not None and self._turn_id != identifier:
            self._stop("turn-response-id-conflicts-with-early-event")
            return False
        for pending in self._pending:
            early_turn = pending.get("turn")
            if pending.get("kind") in ("turn/started", "turn/completed") and self._is_mapping(early_turn):
                if early_turn.get("id") != identifier:
                    self._stop("turn-response-id-conflicts-with-pending-event")
                    return False
        self._turn_id = identifier
        if not self._process_turn_items(turn.get("items"), source="response"):
            return False
        return True

    def accept_notification(self, method: str, params: Any) -> None:
        """Validate, reduce, then correlate one notification; unknowns stop."""
        if self._failed:
            return
        self._notification_count += 1
        if self._notification_count > self._max_notifications:
            self._stop("notification-limit")
            return
        if not isinstance(method, str) or method not in _SCHEMAS or method in ("thread/start", "turn/start"):
            self._stop("unknown-notification")
            return
        if not self._validate(method, params) or not self._is_mapping(params):
            if not self._failed:
                self._stop("malformed-notification")
            return
        record = self._reduce_notification(method, params)
        if record is None or self._failed:
            return
        if self._turn_completed_seen:
            kind = record.get("kind")
            if kind == "thread/closed":
                pass
            elif kind == "thread/status/changed":
                status = record.get("status")
                if not self._is_mapping(status) or status.get("type") != "idle" or self._postterminal_idle_seen:
                    self._stop("notification-after-terminal-turn")
                    return
                self._postterminal_idle_seen = True
            elif kind == "thread/tokenUsage/updated":
                pass
            else:
                self._stop("notification-after-terminal-turn")
                return
        if self._thread_id is None or (record.get("turnId") and self._turn_id is None):
            self._queue(record)
            return
        if (record.get("kind") in ("item/agentMessage/delta", "item/commandExecution/outputDelta",
                                    "item/reasoning/summaryTextDelta", "item/reasoning/summaryPartAdded", "item/reasoning/textDelta")
                and record.get("itemId") not in self._items):
            self._queue(record)
            return
        self._apply(record)
        if not self._failed and self._pending:
            self._reconcile_pending()

    def _reduce_notification(self, method: str, params: Mapping[str, Any]) -> dict[str, Any] | None:
        if method == "thread/started":
            thread = params.get("thread")
            if not self._is_mapping(thread):
                self._stop("thread-started-missing-thread")
                return None
            return {"kind": method, "thread": self._thread_snapshot(thread)}
        if method == "thread/status/changed":
            status = params.get("status")
            if not self._is_mapping(status):
                self._stop("invalid-thread-status")
                return None
            return {"kind": method, "threadId": params.get("threadId"), "status": dict(status)}
        if method == "thread/closed":
            return {"kind": method, "threadId": params.get("threadId")}
        if method == "thread/tokenUsage/updated":
            usage = params.get("tokenUsage")
            if not self._is_mapping(usage):
                self._stop("invalid-usage")
                return None
            total = usage.get("total")
            if not self._is_mapping(total):
                self._stop("invalid-usage-total")
                return None
            if not self._int64(usage.get("modelContextWindow"), optional=True):
                self._stop("invalid-usage-context-window")
                return None
            fields = ("inputTokens", "outputTokens", "cachedInputTokens", "reasoningOutputTokens", "cacheWriteInputTokens", "totalTokens")
            normalized: dict[str, int] = {}
            for field in fields:
                value = total.get(field, 0 if field == "cacheWriteInputTokens" else None)
                if not self._int64(value) or value < 0:
                    self._stop("invalid-usage-counter")
                    return None
                normalized[field] = value
            return {"kind": method, "threadId": params.get("threadId"), "turnId": params.get("turnId"), "usage": normalized}
        if method in ("turn/started", "turn/completed"):
            turn = params.get("turn")
            if not self._is_mapping(turn):
                self._stop("turn-event-missing-turn")
                return None
            required_status = "inProgress" if method == "turn/started" else "completed"
            if turn.get("status") != required_status:
                self._stop("turn-event-status-mismatch")
                return None
            for field in ("startedAt", "completedAt", "durationMs"):
                if not self._int64(turn.get(field), optional=True):
                    self._stop("invalid-turn-timestamp")
                    return None
            return {"kind": method, "threadId": params.get("threadId"), "turnId": turn.get("id"), "turn": self._turn_snapshot(turn)}
        if method in ("item/started", "item/completed"):
            timestamp_key = "startedAtMs" if method == "item/started" else "completedAtMs"
            if not self._int64(params.get(timestamp_key)):
                self._stop("invalid-item-timestamp")
                return None
            item = params.get("item")
            if not self._is_mapping(item):
                self._stop("item-event-missing-item")
                return None
            clean = self._item_snapshot(item, parse_agent=False)
            if clean is None:
                return None
            return {"kind": method, "threadId": params.get("threadId"), "turnId": params.get("turnId"), "item": clean}
        if method in ("item/agentMessage/delta", "item/commandExecution/outputDelta"):
            item_id = params.get("itemId")
            delta = params.get("delta") if method.endswith("outputDelta") else None
            if method == "item/commandExecution/outputDelta":
                if not isinstance(delta, str) or len(delta.encode("utf-8")) > self._max_safe_output:
                    self._stop("invalid-command-output-delta")
                    return None
                tracked = self._items.get(item_id) if isinstance(item_id, str) else None
                candidate = tracked.get("outputDelta", "") if self._is_mapping(tracked) else ""
                for pending in self._pending:
                    if pending.get("kind") == method and pending.get("itemId") == item_id:
                        candidate += pending.get("delta", "")
                candidate += delta
                if len(candidate.encode("utf-8")) > self._max_safe_output or not self._output_prefix_allowed(candidate):
                    self._stop("command-output-delta-mismatch")
                    return None
            return {"kind": method, "threadId": params.get("threadId"), "turnId": params.get("turnId"), "itemId": item_id, "delta": delta}
        if method in ("item/reasoning/summaryTextDelta", "item/reasoning/summaryPartAdded", "item/reasoning/textDelta"):
            index_key = "contentIndex" if method.endswith("textDelta") and method == "item/reasoning/textDelta" else "summaryIndex"
            if not self._int64(params.get(index_key)):
                self._stop("invalid-reasoning-index")
                return None
            return {"kind": method, "threadId": params.get("threadId"), "turnId": params.get("turnId"), "itemId": params.get("itemId")}
        self._stop("unhandled-notification")
        return None

    @staticmethod
    def _thread_snapshot(thread: Mapping[str, Any]) -> dict[str, Any]:
        return {key: thread.get(key) for key in ("id", "cwd", "model", "modelProvider", "ephemeral", "parentThreadId", "forkedFromId")}

    def _turn_snapshot(self, turn: Mapping[str, Any]) -> dict[str, Any] | None:
        status = turn.get("status")
        if status not in ("inProgress", "completed") or turn.get("error") is not None:
            self._stop("turn-failed-or-interrupted")
            return None
        items = turn.get("items")
        if not isinstance(items, list) or len(items) > self._max_items:
            self._stop("invalid-turn-items")
            return None
        reduced: list[dict[str, Any]] = []
        for item in items:
            if not self._is_mapping(item):
                self._stop("invalid-turn-item")
                return None
            clean = self._item_snapshot(item, parse_agent=(status == "completed"))
            if clean is None:
                return None
            reduced.append(clean)
        return {"id": turn.get("id"), "status": status, "items": reduced}

    def _process_turn_items(self, items: Any, source: str) -> bool:
        if not isinstance(items, list) or len(items) > self._max_items:
            self._stop("invalid-turn-items")
            return False
        for item in items:
            if not self._is_mapping(item):
                self._stop("invalid-turn-item")
                return False
            clean = self._item_snapshot(item, parse_agent=False)
            if clean is None:
                return False
            if not self._record_item(clean, completed=False, source=source):
                return False
            if clean.get("type") == "userMessage":
                if self._user_item_id not in (None, clean.get("id")):
                    self._stop("multiple-user-message-items")
                    return False
                self._user_item_id = clean.get("id")
        return True

    def _item_snapshot(self, item: Mapping[str, Any], parse_agent: bool = True) -> dict[str, Any] | None:
        item_type = item.get("type")
        item_id = item.get("id")
        if item_type not in _ALLOWED_ITEM_TYPES:
            self._stop("forbidden-item-type")
            return None
        if not isinstance(item_id, str) or not item_id or len(item_id) > 128:
            self._stop("invalid-item-id")
            return None
        clean: dict[str, Any] = {"id": item_id, "type": item_type}
        if item_type == "commandExecution":
            self._observed_command_ids.add(item_id)
            if len(self._observed_command_ids) > 1:
                self._stop("multiple-command-items")
                return None
        if item_type == "userMessage":
            content = item.get("content")
            if not isinstance(content, list) or len(content) != 1 or not self._is_mapping(content[0]):
                self._stop("unexpected-user-message-content")
                return None
            piece = content[0]
            if piece.get("type") != "text" or piece.get("text") != self._turn_prompt:
                self._stop("user-message-prompt-mismatch")
                return None
            clean["promptMatches"] = True
        elif item_type == "commandExecution":
            command = item.get("command")
            cwd = item.get("cwd")
            status = item.get("status")
            source = item.get("source")
            exit_code = item.get("exitCode")
            output = item.get("aggregatedOutput")
            if command != self._command or cwd != self._cwd:
                self._stop("command-or-cwd-mismatch")
                return None
            if source not in (None, "agent"):
                self._stop("command-source-not-native-agent")
                return None
            if self._command_item_id not in (None, item_id):
                self._stop("multiple-command-items")
                return None
            self._command_item_id = item_id
            if status not in ("inProgress", "completed"):
                self._stop("command-failed-or-declined")
                return None
            if exit_code is not None and (type(exit_code) is not int or exit_code != 0):
                self._stop("command-nonzero-exit")
                return None
            clean.update({"command": command, "cwd": cwd, "status": status, "exitCode": exit_code, "source": source})
            if output is not None:
                if not isinstance(output, str) or len(output.encode("utf-8")) > self._max_safe_output:
                    self._stop("command-output-too-large-or-invalid")
                    return None
                if status == "completed":
                    if output.strip() != self._sentinel:
                        self._stop("command-output-mismatch")
                        return None
                    clean["output"] = self._sentinel
        elif item_type == "agentMessage":
            text = item.get("text")
            phase = item.get("phase")
            if not isinstance(text, str):
                self._stop("invalid-agent-message")
                return None
            if not parse_agent:
                clean["finalSentinel"] = None
                clean["messagePhase"] = phase
                return clean
            parsed = self._parse_final(text)
            if phase == "final" and parsed is None:
                self._stop("final-answer-mismatch")
                return None
            clean["finalSentinel"] = parsed
            clean["messagePhase"] = phase
        return clean

    def _parse_final(self, text: str) -> str | None:
        if len(text.encode("utf-8")) > self._max_safe_output:
            return None
        try:
            def unique_pairs(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
                result: dict[str, Any] = {}
                for key, value in pairs:
                    if key in result:
                        raise ValueError("duplicate JSON key")
                    result[key] = value
                return result

            parsed = json.loads(text, object_pairs_hook=unique_pairs)
        except (TypeError, ValueError):
            return None
        if not isinstance(parsed, dict) or set(parsed) != {"sentinel"}:
            return None
        value = parsed.get("sentinel")
        if value != self._sentinel or not isinstance(value, str):
            return None
        return value

    def _queue(self, record: dict[str, Any]) -> None:
        if len(self._pending) >= self._max_pending:
            self._stop("pending-event-limit")
            return
        self._pending.append(record)

    def _reconcile_pending(self) -> None:
        if self._failed:
            return
        waiting = self._pending
        self._pending = []
        for record in waiting:
            if (record.get("kind") in ("item/agentMessage/delta", "item/commandExecution/outputDelta",
                                        "item/reasoning/summaryTextDelta", "item/reasoning/summaryPartAdded", "item/reasoning/textDelta")
                    and record.get("itemId") not in self._items):
                self._pending.append(record)
                continue
            if self._turn_completed_seen and record.get("kind") not in (
                    "thread/closed", "thread/status/changed", "thread/tokenUsage/updated"):
                self._stop("pending-event-after-terminal-turn")
                return
            if self._turn_completed_seen and record.get("kind") == "thread/status/changed":
                status = record.get("status")
                if not self._is_mapping(status) or status.get("type") != "idle" or self._postterminal_idle_seen:
                    self._stop("pending-status-after-terminal-turn")
                    return
                self._postterminal_idle_seen = True
            self._apply(record)
            if self._failed:
                return

    def _apply(self, record: Mapping[str, Any]) -> None:
        kind = record.get("kind")
        if kind == "thread/started":
            snap = record.get("thread")
            if not self._is_mapping(snap):
                self._stop("invalid-thread-snapshot")
                return
            if self._thread_started_seen or snap.get("id") != self._thread_id:
                self._stop("thread-started-id-mismatch-or-duplicate")
                return
            if (snap.get("cwd") != self._cwd or snap.get("modelProvider") != self._thread_start["params"].get("modelProvider")
                    or snap.get("ephemeral") is not True or snap.get("parentThreadId") is not None or snap.get("forkedFromId") is not None):
                self._stop("thread-started-policy-mismatch")
                return
            if snap.get("model") not in (None, self._thread_start["params"].get("model")):
                self._stop("thread-started-model-mismatch")
                return
            self._thread_started_seen = True
            return
        if kind == "thread/status/changed":
            if not self._match_thread(record.get("threadId")):
                return
            status = record.get("status")
            status_type = status.get("type") if self._is_mapping(status) else None
            if status_type == "systemError":
                self._stop("thread-system-error")
                return
            if status_type == "active":
                flags = status.get("activeFlags")
                if not isinstance(flags, list) or flags:
                    self._stop("thread-waiting-or-unknown-active-flag")
                    return
            elif status_type not in ("idle",):
                self._stop("thread-status-not-allowed")
                return
            self._thread_status = status_type
            return
        if kind == "thread/closed":
            if not self._match_thread(record.get("threadId")):
                return
            if not self._turn_completed_seen or self._thread_closed_seen:
                self._stop("thread-closed-before-completion-or-duplicate")
                return
            self._thread_closed_seen = True
            return
        if kind == "thread/tokenUsage/updated":
            if not self._match_ids(record.get("threadId"), record.get("turnId")):
                return
            usage = record.get("usage")
            if not self._is_mapping(usage):
                self._stop("invalid-usage-counters")
                return
            self._latest_usage = dict(usage)
            self._usage_seen = True
            if self._latest_usage is not None:
                # Compare against the previous cumulative snapshot, never add
                # snapshots together. This also bounds post-terminal updates.
                previous = getattr(self, "_previous_usage", None)
                if previous is not None and any(usage[name] < previous[name] for name in usage):
                    self._stop("usage-counter-regressed")
                    return
                self._previous_usage = dict(usage)
            observed = usage["inputTokens"] + usage["outputTokens"]
            if observed > self._token_threshold:
                self._usage_overshoot = observed - self._token_threshold
                self._stop("token-observation-threshold")
            return
        if kind in ("turn/started", "turn/completed"):
            if not self._match_thread(record.get("threadId")):
                return
            turn = record.get("turn")
            if not self._is_mapping(turn) or not isinstance(turn.get("id"), str):
                self._stop("invalid-turn-snapshot")
                return
            if self._turn_id is None:
                self._turn_id = turn["id"]
            if turn.get("id") != self._turn_id:
                self._stop("turn-id-mismatch")
                return
            if kind == "turn/started":
                if self._turn_started_seen:
                    self._stop("duplicate-turn-started")
                    return
                self._turn_started_seen = True
                self._process_reduced_turn_items(turn.get("items"), "turn-start-snapshot")
                return
            if self._turn_completed_seen or turn.get("status") != "completed":
                self._stop("turn-not-completed-or-duplicate")
                return
            self._turn_completed_seen = True
            self._process_reduced_turn_items(turn.get("items"), "turn-completed-snapshot")
            return
        if kind in ("item/started", "item/completed"):
            if not self._match_ids(record.get("threadId"), record.get("turnId")):
                return
            item = record.get("item")
            if not self._is_mapping(item):
                self._stop("invalid-item-event")
                return
            completed = kind == "item/completed"
            if not self._record_item(item, completed=completed, source=kind):
                return
            return
        if kind in ("item/agentMessage/delta", "item/commandExecution/outputDelta",
                    "item/reasoning/summaryTextDelta", "item/reasoning/summaryPartAdded", "item/reasoning/textDelta"):
            if not self._match_ids(record.get("threadId"), record.get("turnId")):
                return
            item_id = record.get("itemId")
            if not isinstance(item_id, str):
                self._stop("invalid-delta-item-id")
                return
            tracked = self._items.get(item_id)
            if tracked is None:
                self._stop("delta-for-untracked-item")
                return
            if kind == "item/agentMessage/delta" and tracked["type"] != "agentMessage":
                self._stop("agent-delta-item-type-mismatch")
                return
            if kind == "item/commandExecution/outputDelta":
                if tracked["type"] != "commandExecution":
                    self._stop("command-output-delta-item-mismatch")
                    return
                delta = record.get("delta")
                if not isinstance(delta, str):
                    self._stop("invalid-command-output-delta")
                    return
                prior = tracked.get("outputDelta", "")
                combined = prior + delta
                if len(combined.encode("utf-8")) > self._max_safe_output:
                    self._stop("command-output-too-large")
                    return
                if not self._output_prefix_allowed(combined):
                    self._stop("command-output-delta-mismatch")
                    return
                tracked["outputDelta"] = combined
            elif kind.startswith("item/reasoning/") and tracked["type"] != "reasoning":
                self._stop("reasoning-delta-item-type-mismatch")
                return
            return
        self._stop("unhandled-notification")

    def _record_item(self, item: Mapping[str, Any], completed: bool, source: str, terminal_agent: bool = False) -> bool:
        item_id = item.get("id")
        item_type = item.get("type")
        if not isinstance(item_id, str) or item_type not in _ALLOWED_ITEM_TYPES:
            self._stop("invalid-item-record")
            return False
        if item_type == "commandExecution":
            if self._command_item_id not in (None, item_id):
                self._stop("multiple-command-items")
                return False
            # Reserve the sole command slot at first sight, not only after a
            # successful completion, so a second or parallel invocation stops.
            self._command_item_id = item_id
        current = self._items.get(item_id)
        if current is None:
            if len(self._items) >= self._max_items:
                self._stop("item-limit")
                return False
            current = {"id": item_id, "type": item_type, "started": False, "completed": False}
            self._items[item_id] = current
        elif current.get("type") != item_type:
            self._stop("item-id-type-conflict")
            return False
        if source == "item/started":
            if current["started"]:
                self._stop("duplicate-item-started")
                return False
            current["started"] = True
        if completed:
            if source == "item/completed" and current["completed"]:
                self._stop("duplicate-item-completed")
                return False
            if item_type == "commandExecution":
                if item.get("status") != "completed" or item.get("exitCode") != 0:
                    self._stop("command-completion-not-success")
                    return False
                output = item.get("output")
                if output is None:
                    output = current.get("outputDelta")
                if not isinstance(output, str) or output.strip() != self._sentinel:
                    self._stop("command-output-mismatch")
                    return False
                delta = current.get("outputDelta")
                if delta is not None and delta.strip() != self._sentinel:
                    self._stop("command-output-delta-mismatch")
                    return False
                if self._command_item_id not in (None, item_id):
                    self._stop("multiple-command-items")
                    return False
                self._command_item_id = item_id
                if source == "item/completed":
                    self._command_completed_event_seen = True
                self._safe_output = self._sentinel
                current["completedOutput"] = self._sentinel
            elif item_type == "agentMessage":
                final = item.get("finalSentinel")
                if final == self._sentinel and (item.get("messagePhase") == "final" or terminal_agent):
                    if self._answer_item_id not in (None, item_id):
                        self._stop("multiple-final-message-items")
                        return False
                    self._answer_item_id = item_id
                    self._final_candidate_item_id = item_id
                    self._safe_final = final
                elif item.get("messagePhase") == "final" or terminal_agent:
                    self._stop("final-answer-mismatch")
                    return False
            elif item_type == "userMessage":
                if not item.get("promptMatches"):
                    self._stop("user-message-prompt-mismatch")
                    return False
                if self._user_item_id not in (None, item_id):
                    self._stop("multiple-user-message-items")
                    return False
                self._user_item_id = item_id
            current["completed"] = True
        return True

    def _process_reduced_turn_items(self, items: Any, source: str) -> None:
        if not isinstance(items, list):
            self._stop("invalid-turn-items")
            return
        for item in items:
            if not self._is_mapping(item):
                self._stop("invalid-turn-item")
                return
            item_type = item.get("type")
            completed = source == "turn-completed-snapshot" and item_type == "agentMessage"
            terminal_agent = (source == "turn-completed-snapshot" and item_type == "agentMessage"
                              and item.get("id") == self._last_agent_id(items))
            self._record_item(item, completed=completed, source=source, terminal_agent=terminal_agent)
            if item_type == "userMessage":
                if self._user_item_id not in (None, item.get("id")):
                    self._stop("multiple-user-message-items")
                else:
                    self._user_item_id = item.get("id")
            if self._failed:
                return

    @staticmethod
    def _last_agent_id(items: list[Any]) -> Any:
        for item in reversed(items):
            if isinstance(item, Mapping) and item.get("type") == "agentMessage":
                return item.get("id")
        return None

    def _output_prefix_allowed(self, output: str) -> bool:
        candidate = output.strip()
        if candidate == "":
            return True
        if len(candidate) <= len(self._sentinel):
            return self._sentinel.startswith(candidate)
        return candidate.startswith(self._sentinel) and candidate[len(self._sentinel):].strip() == ""

    def _match_thread(self, thread_id: Any) -> bool:
        if not isinstance(thread_id, str) or thread_id != self._thread_id:
            self._stop("thread-id-mismatch")
            return False
        return True

    def _match_ids(self, thread_id: Any, turn_id: Any) -> bool:
        if not self._match_thread(thread_id):
            return False
        if not isinstance(turn_id, str) or turn_id != self._turn_id:
            return self._bad_turn_id(turn_id)
        return True

    def _bad_turn_id(self, turn_id: Any) -> bool:
        self._stop("turn-id-mismatch")
        return False

    @property
    def active_ids(self) -> dict[str, str]:
        """IDs the transport may use for one bounded interrupt during stop."""
        ids: dict[str, str] = {}
        if self._thread_id is not None:
            ids["threadId"] = self._thread_id
        if self._turn_id is not None and not self._turn_completed_seen:
            ids["turnId"] = self._turn_id
        return ids

    @property
    def failure_reason(self) -> str | None:
        """Stable non-sensitive reason for a transport to stop immediately."""
        return self._stop_reason if self._failed else None

    @property
    def terminal(self) -> bool:
        return self._turn_completed_seen or self._failed

    def finish(self) -> dict[str, Any]:
        """Return a sanitized bounded receipt; never include arbitrary text."""
        if self._pending and not self._failed:
            self._stop("unreconciled-pending-events")
        if not self._failed and not self._turn_completed_seen:
            self._stop("turn-not-completed")
        if not self._failed:
            for current in self._items.values():
                delta = current.get("outputDelta")
                output = current.get("completedOutput")
                if delta is not None and output is not None and delta.strip() != self._sentinel:
                    self._stop("command-output-delta-mismatch")
                    break
        if not self._failed:
            command = self._items.get(self._command_item_id) if self._command_item_id is not None else None
            if (not self._is_mapping(command) or command.get("type") != "commandExecution"
                    or not command.get("completed") or not self._command_completed_event_seen):
                self._stop("command-completion-event-missing")
        if not self._failed:
            if not (self._thread_response_seen and self._turn_response_seen and self._thread_started_seen
                    and self._turn_started_seen and self._command_item_id and self._answer_item_id
                    and self._user_item_id and self._safe_output == self._sentinel and self._safe_final == self._sentinel):
                self._stop("success-proof-incomplete")
        return {
            "status": "failed" if self._failed else "complete",
            "stopReason": self._stop_reason,
            "threadId": self._thread_id,
            "turnId": self._turn_id,
            "itemIds": {
                "userMessage": self._user_item_id,
                "commandExecution": self._command_item_id,
                "agentMessage": self._answer_item_id,
            },
            "safePublicOutput": self._safe_output,
            "finalSentinel": self._safe_final,
            "commandCompletionEventObserved": self._command_completed_event_seen,
            "commandExecutionItemCount": len(self._observed_command_ids),
            "usage": dict(self._latest_usage) if self._latest_usage is not None else None,
            "usageSeen": self._usage_seen,
            "usageOvershoot": self._usage_overshoot,
            "notificationCount": self._notification_count,
            "threadStatus": self._thread_status,
            "threadClosed": self._thread_closed_seen,
        }
